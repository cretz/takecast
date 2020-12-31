package receiver

import (
	"context"
)

type Application interface {
	// Can trust not mutated after returned. This is a quick call, so callers can
	// call it frequently.
	Metadata() *ApplicationMetadata
	// Should not error if already started. Fast and non-blocking. Context can be
	// used for life of run even beyond call.
	Start(ctx context.Context, appID string, appParams interface{}) error
	// Should not error if not started. Fast and non-blocking.
	Stop(ctx context.Context) error
	// Handle the message
	HandleMessage(ctx context.Context, conn Conn, msg RequestMessage) error
}

type ApplicationMetadata struct {
	AppIDs              []string
	SessionID           string
	DisplayName         string
	StatusText          string
	SupportedNamespaces []string
}
