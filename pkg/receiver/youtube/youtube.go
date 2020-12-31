package youtube

import (
	"context"
	"fmt"
	"sync"

	"github.com/cretz/takecast/pkg/receiver"
	"github.com/google/uuid"
)

type YouTube interface {
	receiver.Application
}

type youTube struct {
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

const AppID = "233637DE"

func New(config Config) (YouTube, error) {
	y := &youTube{Config: config}
	if len(y.DefaultMetadata.AppIDs) == 0 {
		y.DefaultMetadata.AppIDs = []string{AppID}
	}
	if y.DefaultMetadata.DisplayName == "" {
		y.DefaultMetadata.DisplayName = "TakeCast YouTube"
	}
	if len(y.DefaultMetadata.SupportedNamespaces) == 0 {
		y.DefaultMetadata.SupportedNamespaces = []string{
			"urn:x-cast:com.google.youtube.mdx",
		}
	}
	if y.Log == nil {
		y.Log = receiver.NopLog()
	}
	y.metadata = y.newApplicationMetadata()
	return y, nil
}

func (y *youTube) Metadata() *receiver.ApplicationMetadata {
	y.metadataLock.RLock()
	defer y.metadataLock.RUnlock()
	return y.metadata
}

func (y *youTube) Start(ctx context.Context, appID string, appParams interface{}) error {
	y.metadataLock.Lock()
	defer y.metadataLock.Unlock()
	// If there's already a session ID, nothing to do
	if y.metadata.SessionID != "" {
		return nil
	}
	// Start session
	y.metadata = y.newApplicationMetadata()
	y.metadata.SessionID = uuid.New().String()
	return nil
}

func (y *youTube) Stop(ctx context.Context) error {
	y.metadataLock.Lock()
	defer y.metadataLock.Unlock()
	// If there's no session, nothing to do
	if y.metadata.SessionID == "" {
		return nil
	}
	// Start session
	y.metadata = y.newApplicationMetadata()
	return nil
}

func (y *youTube) newApplicationMetadata() *receiver.ApplicationMetadata {
	return &receiver.ApplicationMetadata{
		AppIDs:              y.DefaultMetadata.AppIDs,
		DisplayName:         y.DefaultMetadata.DisplayName,
		StatusText:          "Ready To Cast",
		SupportedNamespaces: y.DefaultMetadata.SupportedNamespaces,
	}
}

func (y *youTube) HandleMessage(ctx context.Context, conn receiver.Conn, msg receiver.RequestMessage) error {
	// TODO:
	fmt.Printf("!!TODO!! YOUTUBE GOT MSG: %T - %v\n", msg, msg.Header().Raw)
	return nil
}
