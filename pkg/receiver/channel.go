package receiver

import (
	"context"
	"fmt"
	"sync"

	"github.com/cretz/takecast/pkg/log"
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
}

type ChannelConfig struct {
	Receiver       Receiver
	Conn           Conn
	ConnectionInfo *ConnectRequestMessage
}

func NewChannel(config ChannelConfig) (Channel, error) {
	if config.Receiver == nil {
		return nil, fmt.Errorf("missing receiver")
	} else if config.Conn == nil {
		return nil, fmt.Errorf("missing conn")
	} else if config.ConnectionInfo == nil {
		return nil, fmt.Errorf("missing connection info")
	}
	c := &channel{recv: config.Receiver, conn: config.Conn, connectionInfo: config.ConnectionInfo}
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
			msgBld.SetNamespace("urn:x-cast:com.google.cast.receiver").MustSetJSONPayload(&ReceiverStatusResponseMessage{
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
		panic("TODO")
	case *PingRequestMessage:
		resp := &MessageHeader{Type: "PONG"}
		return new(MessageBuilder).ApplyReceived(msg.Raw).MustSetJSONPayload(resp).Send(c.conn)
	default:
		// For now, we'll just ignore unknown messages
		log.Debugf("Ignoring unknown message: %v", msg.Header().Raw)
		return nil
	}
}
