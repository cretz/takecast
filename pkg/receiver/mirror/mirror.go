package mirror

import (
	"context"
	"fmt"
	"sync"

	"github.com/cretz/takecast/pkg/receiver"
	"github.com/cretz/takecast/pkg/receiver/webrtc"
	"github.com/google/uuid"
)

type Mirror interface {
	receiver.Application
}

type mirror struct {
	Config

	lock sync.RWMutex // Governs fields below
	// Entire field reassigned each time, nothing internal ever changed
	metadata *receiver.ApplicationMetadata
	session  *webrtc.Session
}

type Config struct {
	Log receiver.Log
	// Only used for AppIDs, DisplayName, and SupportedNamespaces. Defaults
	// provided for each when not set.
	DefaultMetadata receiver.ApplicationMetadata
	// Called async
	OnSession func(*webrtc.Session)
}

const (
	AppID          = "0F5096E8"
	AudioOnlyAppID = "85CDB22F"
)

func New(config Config) (Mirror, error) {
	m := &mirror{Config: config}
	if len(m.DefaultMetadata.AppIDs) == 0 {
		m.DefaultMetadata.AppIDs = []string{AppID, AudioOnlyAppID}
	}
	if m.DefaultMetadata.DisplayName == "" {
		m.DefaultMetadata.DisplayName = "TakeCast Mirror"
	}
	if len(m.DefaultMetadata.SupportedNamespaces) == 0 {
		m.DefaultMetadata.SupportedNamespaces = []string{receiver.NamespaceWebRTC, receiver.NamespaceRemoting}
	}
	if m.Log == nil {
		m.Log = receiver.NopLog()
	}
	m.metadata = m.newApplicationMetadata()
	return m, nil
}

func (m *mirror) Metadata() *receiver.ApplicationMetadata {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.metadata
}

func (m *mirror) Start(ctx context.Context, appID string, appParams interface{}) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	// If there's already a session ID, nothing to do
	if m.metadata.SessionID != "" {
		return nil
	}
	// Start session
	m.metadata = m.newApplicationMetadata()
	m.metadata.SessionID = uuid.New().String()
	return nil
}

func (m *mirror) Stop(ctx context.Context) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	// If there's no session, nothing to do
	if m.metadata.SessionID == "" {
		return nil
	}
	// Stop session if there
	if m.session != nil {
		m.session.Close()
		m.session = nil
	}
	// Create new metadata without session ID
	m.metadata = m.newApplicationMetadata()
	return nil
}

func (m *mirror) newApplicationMetadata() *receiver.ApplicationMetadata {
	return &receiver.ApplicationMetadata{
		AppIDs:              m.DefaultMetadata.AppIDs,
		DisplayName:         m.DefaultMetadata.DisplayName,
		StatusText:          "Ready To Cast",
		SupportedNamespaces: m.DefaultMetadata.SupportedNamespaces,
	}
}

func (m *mirror) HandleMessage(ctx context.Context, conn receiver.Conn, msg receiver.RequestMessage) error {
	switch msg := msg.(type) {
	case *receiver.WebRTCOfferRequestMessage:
		// Lock during the entire session creation
		session, err := func() (*webrtc.Session, error) {
			m.lock.Lock()
			defer m.lock.Unlock()
			// If there's already a session, error
			if m.session != nil {
				return nil, fmt.Errorf("session already exists")
			}
			// Create session
			var err error
			m.session, err = webrtc.StartSession(m.metadata.SessionID, msg.Offer)
			return m.session, err
		}()
		// Send response
		resp := &receiver.WebRTCAnswerResponseMessage{
			MessageHeader: receiver.MessageHeader{Type: "ANSWER"},
			SeqNum:        msg.SeqNum,
		}
		if err != nil {
			resp.Result = "error"
			resp.Error = &receiver.WebRTCAnswerError{Code: 88, Description: err.Error()}
		} else {
			if m.OnSession != nil {
				m.OnSession(session)
			}
			resp.Result = "ok"
			resp.Answer = session.Answer
		}
		// Send
		return new(receiver.MessageBuilder).ApplyReceived(msg.Raw).MustSetJSONPayload(resp).Send(conn)
	default:
		m.Log.Debugf("Ignoring unknown message: %v", msg.Header().Raw)
		return nil
	}
}
