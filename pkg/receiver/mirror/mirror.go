package mirror

import (
	"context"
	"fmt"
	"sync"

	"github.com/cretz/takecast/pkg/receiver"
	"github.com/google/uuid"
)

type Mirror interface {
	receiver.Application
}

type mirror struct {
	Config
	// Entire field reassigned each time, nothing internal ever changed
	metadata     *receiver.ApplicationMetadata
	metadataLock sync.RWMutex
}

type Config struct {
	// Only used for AppIDs, DisplayName, and SupportedNamespaces. Defaults
	// provided for each when not set.
	DefaultMetadata receiver.ApplicationMetadata
	Log             receiver.Log
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
	m.metadataLock.RLock()
	defer m.metadataLock.RUnlock()
	return m.metadata
}

func (m *mirror) Start(ctx context.Context, appID string, appParams interface{}) error {
	m.metadataLock.Lock()
	defer m.metadataLock.Unlock()
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
	m.metadataLock.Lock()
	defer m.metadataLock.Unlock()
	// If there's no session, nothing to do
	if m.metadata.SessionID == "" {
		return nil
	}
	// Start session
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
	// TODO:
	fmt.Printf("!!TODO!! MIRROR GOT MSG: %T - %v\n", msg, msg.Header().Raw)
	return nil
}
