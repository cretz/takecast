package receiver

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/cretz/takecast/pkg/receiver/cast_channel"
)

type Receiver interface {
	// Will close connection on failure, otherwise closing channel closes conn.
	// Context is only for connect, not lifetime of channel.
	ConnectChannel(context.Context, Conn) (Channel, error)
	RegisterApplication(Application) error
	// Stops the app if running
	UnregisterApplication(context.Context, Application) error
	// Nil if not found
	ApplicationByID(string) Application
	// If appID is empty, just stops current one. Channel not required if appID is
	// empty, required otherwise.
	SwitchToApplication(ctx context.Context, ch Channel, appID string, params interface{}) error
	CurrentApplication() Application
	// Result should not be mutated (without being cloned first)
	Status() *ReceiverStatus
	// Channel should have buffer, sent to non-blocking
	AddStatusListener(chan<- *ReceiverStatus)
	RemoveStatusListener(chan<- *ReceiverStatus)
	// Does not close conns/channels
	Close() error
}

var ErrReceiverClosed = errors.New("receiver closed")

type receiver struct {
	config ReceiverConfig
	log    Log
	ctx    context.Context
	cancel context.CancelFunc

	lock            sync.RWMutex // Governs all fields below
	apps            map[string]Application
	status          *ReceiverStatus // Always completely replaced, never mutated
	statusListeners map[chan<- *ReceiverStatus]struct{}
}

type ReceiverConfig struct {
	// Default is NewChannel
	NewChannel func(ChannelConfig) (Channel, error)
	Log        Log
}

func NewReceiver(config ReceiverConfig) Receiver {
	r := &receiver{
		config: config,
		log:    config.Log,
		apps:   map[string]Application{},
		status: &ReceiverStatus{
			IsActiveInput: true,
			Volume: &Volume{
				Level: 1,
			},
		},
		statusListeners: map[chan<- *ReceiverStatus]struct{}{},
	}
	if r.log == nil {
		r.log = NopLog()
	}
	if r.config.NewChannel == nil {
		r.config.NewChannel = NewChannel
	}
	r.ctx, r.cancel = context.WithCancel(context.Background())
	return r
}

func (r *receiver) ConnectChannel(ctx context.Context, conn Conn) (Channel, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	success := false
	defer func() {
		if !success {
			conn.Close()
		}
	}()
	// Do in background so it can be canceled
	var connInfo *ConnectRequestMessage
	var connErr error
	connDoneCh := make(chan struct{})
	go func() {
		defer close(connDoneCh)
		connInfo, connErr = r.connect(conn)
	}()
	// Wait until done
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-r.ctx.Done():
		return nil, ErrReceiverClosed
	case <-connDoneCh:
		if connErr != nil {
			return nil, connErr
		}
		success = true
		return r.config.NewChannel(ChannelConfig{
			Receiver:       r,
			Conn:           conn,
			ConnectionInfo: connInfo,
			Log:            r.log,
		})
	}
}

func (r *receiver) connect(conn Conn) (*ConnectRequestMessage, error) {
	msg, err := receiveRequestMessage(conn)
	if err != nil {
		return nil, err
	}
	// If it's auth, handle that
	if authReq, _ := msg.(*DeviceAuthRequestMessage); authReq != nil {
		r.log.Debugf("Received auth request: %v", authReq)
		authResp, err := conn.Auth(authReq)
		if err != nil {
			// Send auth error (ignoring send error) and fail this
			new(MessageBuilder).ApplyReceived(authReq.Raw).MustSetProtoPayload(
				&cast_channel.DeviceAuthMessage{Error: &cast_channel.AuthError{}}).Send(conn)
			return nil, fmt.Errorf("failed auth: %w", err)
		}
		// Send auth response
		err = new(MessageBuilder).ApplyReceived(authReq.Raw).MustSetProtoPayload(
			&cast_channel.DeviceAuthMessage{Response: authResp}).Send(conn)
		if err != nil {
			return nil, fmt.Errorf("failed sending auth: %w", err)
		}
		// Now fetch message again
		if msg, err = receiveRequestMessage(conn); err != nil {
			return nil, err
		}
	}
	// Require it be a connection message
	connReq, _ := msg.(*ConnectRequestMessage)
	if connReq == nil {
		return nil, fmt.Errorf("expected connect message, got: %v", msg)
	}
	return connReq, nil
}

func (r *receiver) RegisterApplication(app Application) error {
	if r.ctx.Err() != nil {
		return ErrReceiverClosed
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	// Make sure all app IDs available before trying
	for _, appID := range app.Metadata().AppIDs {
		if r.apps[appID] != nil {
			return fmt.Errorf("application ID %v already registered", appID)
		}
	}
	for _, appID := range app.Metadata().AppIDs {
		r.apps[appID] = app
	}
	return nil
}

func (r *receiver) UnregisterApplication(ctx context.Context, app Application) error {
	if r.ctx.Err() != nil {
		return ErrReceiverClosed
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	for _, appID := range app.Metadata().AppIDs {
		// If the app is running, stop it first
		if len(r.status.Applications) > 0 && appID == r.status.Applications[0].AppID {
			if err := r.switchToApplicationUnlocked(ctx, nil, "", nil); err != nil {
				return err
			}
		}
		delete(r.apps, appID)
	}
	return nil
}

func (r *receiver) ApplicationByID(id string) Application {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.apps[id]
}

func (r *receiver) SwitchToApplication(ctx context.Context, ch Channel, appID string, params interface{}) error {
	if r.ctx.Err() != nil {
		return ErrReceiverClosed
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.switchToApplicationUnlocked(ctx, ch, appID, params)
}

func (r *receiver) switchToApplicationUnlocked(ctx context.Context, ch Channel, appID string, params interface{}) error {
	if appID != "" && r.apps[appID] == nil {
		return fmt.Errorf("unrecognized application ID %v", appID)
	} else if appID != "" && ch == nil {
		return fmt.Errorf("must have channel when giving application to switch to")
	}
	// Stop the current app if there is one not that's not the same ID
	if len(r.status.Applications) > 0 && r.status.Applications[0].AppID != appID {
		r.log.Debugf("Stopping application %v", r.status.Applications[0].AppID)
		if err := r.apps[r.status.Applications[0].AppID].Stop(ctx); err != nil {
			return fmt.Errorf("failed stopping current application: %w", err)
		}
	}
	// Start the requested app if any but not already the current
	if appID != "" && (len(r.status.Applications) == 0 || r.status.Applications[0].AppID != appID) {
		r.log.Debugf("Starting application %v", appID)
		if err := r.apps[appID].Start(ctx, appID, params); err != nil {
			return fmt.Errorf("failed starting application: %w", err)
		}
	}
	// Rebuild the status every time, no matter what
	newStatus := &ReceiverStatus{
		IsActiveInput: r.status.IsActiveInput,
		Volume: &Volume{
			Level: r.status.Volume.Level,
			Muted: r.status.Volume.Muted,
		},
	}
	if appID != "" {
		appMeta := r.apps[appID].Metadata()
		appStatus := &ApplicationStatus{
			TransportID:    ch.ConnectionInfo().Raw.GetSourceId(),
			SessionID:      appMeta.SessionID,
			AppID:          appID,
			UniversalAppID: appID,
			DisplayName:    appMeta.DisplayName,
			StatusText:     appMeta.StatusText,
			// TODO: IsIdleScreen
			Namespaces: make([]*ApplicationStatusNamespace, len(appMeta.SupportedNamespaces)),
		}
		for i, namespace := range appMeta.SupportedNamespaces {
			appStatus.Namespaces[i] = &ApplicationStatusNamespace{Name: namespace}
		}
		newStatus.Applications = []*ApplicationStatus{appStatus}
	}
	r.status = newStatus
	// Send status updates non-blocking
	for ch := range r.statusListeners {
		select {
		case ch <- newStatus:
		default:
		}
	}
	return nil
}

func (r *receiver) CurrentApplication() Application {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if len(r.status.Applications) == 0 {
		return nil
	}
	return r.apps[r.status.Applications[0].AppID]
}

func (r *receiver) Status() *ReceiverStatus {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.status
}

func (r *receiver) AddStatusListener(ch chan<- *ReceiverStatus) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.statusListeners[ch] = struct{}{}
}

func (r *receiver) RemoveStatusListener(ch chan<- *ReceiverStatus) {
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.statusListeners, ch)
}

func (r *receiver) Close() error {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.switchToApplicationUnlocked(r.ctx, nil, "", nil)
	r.status.Applications = nil
	r.apps = nil
	r.statusListeners = nil
	r.cancel()
	return nil
}
