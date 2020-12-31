package receiver

import (
	"context"
	"fmt"
	"sync"

	"github.com/cretz/takecast/pkg/receiver/cast_channel"
)

type Channel interface {
	ConnectionInfo() *ConnectRequestMessage
	// Closes conn internally when complete. Must always call this, and only call
	// it once (even if called with already-closed context to close conn).
	Run(context.Context) error
}

type channel struct {
	recv               Receiver
	conn               Conn
	connectionInfo     *ConnectRequestMessage
	connectionInfoLock sync.RWMutex
	runCalled          bool
	log                Log
}

type ChannelConfig struct {
	Receiver       Receiver
	Conn           Conn
	ConnectionInfo *ConnectRequestMessage
	Log            Log
}

func NewChannel(config ChannelConfig) (Channel, error) {
	if config.Receiver == nil {
		return nil, fmt.Errorf("missing receiver")
	} else if config.Conn == nil {
		return nil, fmt.Errorf("missing conn")
	} else if config.ConnectionInfo == nil {
		return nil, fmt.Errorf("missing connection info")
	}
	c := &channel{recv: config.Receiver, conn: config.Conn, connectionInfo: config.ConnectionInfo, log: config.Log}
	if c.log == nil {
		c.log = NopLog()
	}
	return c, nil
}

func (c *channel) ConnectionInfo() *ConnectRequestMessage {
	c.connectionInfoLock.RLock()
	defer c.connectionInfoLock.RUnlock()
	return c.connectionInfo
}

func (c *channel) Run(ctx context.Context) error {
	if c.runCalled {
		return fmt.Errorf("run already called")
	}
	c.runCalled = true
	defer c.conn.Close()
	// Accept status updates w/ a buffer of 10
	statusCh := make(chan *ReceiverStatus, 10)
	c.recv.AddStatusListener(statusCh)
	defer c.recv.RemoveStatusListener(statusCh)
	// Receive messages in the background
	msgCh := make(chan *cast_channel.CastMessage)
	errCh := make(chan error)
	go func() {
		for {
			msg, err := c.conn.Receive()
			if err != nil {
				select {
				case <-ctx.Done():
				case errCh <- err:
				}
				return
			}
			select {
			case <-ctx.Done():
				return
			case msgCh <- msg:
			}
		}
	}()
	// Process messages intentionally not async
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			return err
		case msg := <-msgCh:
			if reqMsg, err := UnmarshalRequestMessage(msg); err != nil {
				return err
			} else if err = c.handleMessage(ctx, reqMsg); err != nil {
				return err
			}
		case status := <-statusCh:
			// Since these are buffered, let's just keep reading while there are any
			// to get the latest
			for len(statusCh) > 0 {
				status = <-statusCh
			}
			// Use the same receive info as last connect request, but change the
			// namespace, then set payload (with no request ID) and send
			c.connectionInfoLock.RLock()
			msgBld := new(MessageBuilder).ApplyReceived(c.connectionInfo.Raw)
			c.connectionInfoLock.RUnlock()
			msgBld.SetNamespace(NamespaceReceiver).MustSetJSONPayload(&ReceiverStatusResponseMessage{
				MessageHeader: MessageHeader{Type: "RECEIVER_STATUS"},
				Status:        status,
			}).Send(c.conn)
		}
	}
}

func (c *channel) handleMessage(ctx context.Context, msg RequestMessage) error {
	switch msg := msg.(type) {
	case *ConnectRequestMessage:
		// Update the connection info
		c.connectionInfoLock.Lock()
		c.connectionInfo = msg
		c.connectionInfoLock.Unlock()
		return nil
	case *GetAppAvailabilityRequestMessage:
		// Send app availability
		resp := &GetAppAvailabilityResponseMessage{
			MessageHeader: MessageHeader{Type: "GET_APP_AVAILABILITY", RequestID: msg.RequestID},
			Availability:  make(map[string]string, len(msg.AppIDs)),
		}
		for _, appID := range msg.AppIDs {
			if c.recv.ApplicationByID(appID) != nil {
				resp.Availability[appID] = "APP_AVAILABLE"
			} else {
				resp.Availability[appID] = "APP_UNAVAILABLE"
			}
		}
		return new(MessageBuilder).ApplyReceived(msg.Raw).MustSetJSONPayload(resp).Send(c.conn)
	case *GetReceiverStatusRequestMessage:
		// Send the current status
		resp := &ReceiverStatusResponseMessage{
			MessageHeader: MessageHeader{Type: "RECEIVER_STATUS", RequestID: msg.RequestID},
			Status:        c.recv.Status(),
		}
		return new(MessageBuilder).ApplyReceived(msg.Raw).MustSetJSONPayload(resp).Send(c.conn)
	case *LaunchRequestMessage:
		return c.recv.SwitchToApplication(ctx, c, msg.AppID, msg.AppParams)
	case *PingRequestMessage:
		resp := &MessageHeader{Type: "PONG"}
		return new(MessageBuilder).ApplyReceived(msg.Raw).MustSetJSONPayload(resp).Send(c.conn)
	case *StopRequestMessage:
		// Send back invalid request if not the right session ID
		if s := c.recv.Status(); len(s.Applications) == 0 || s.Applications[0].SessionID != msg.SessionID {
			resp := &InvalidRequestResponseMessage{
				MessageHeader: MessageHeader{Type: "INVALID_REQUEST", RequestID: msg.RequestID},
				Reason:        "INVALID_SESSION_ID",
			}
			return new(MessageBuilder).ApplyReceived(msg.Raw).MustSetJSONPayload(resp).Send(c.conn)
		}
		// Switch app to nothing
		return c.recv.SwitchToApplication(ctx, c, "", nil)
	default:
		// Grab the current application and if the message is a supported namespace
		// then let the app handle it
		if app := c.recv.CurrentApplication(); app != nil {
			for _, supportedNamespace := range app.Metadata().SupportedNamespaces {
				if supportedNamespace == msg.Header().Raw.GetNamespace() {
					return app.HandleMessage(ctx, c.conn, msg)
				}
			}
		}
		// For now, we'll just ignore unknown messages
		c.log.Debugf("Ignoring unknown message: %v", msg.Header().Raw)
		return nil
	}
}
