package receiver

import "context"

type Application interface {
	// Can trust not mutated after returned. This is a quick call, so callers can
	// call it frequently.
	Metadata() *ApplicationMetadata
	// Should not error if already started
	Start(ctx context.Context, id string, params interface{}) error
	Stop(ctx context.Context) error
}

type ApplicationMetadata struct {
	AppIDs              []string
	SessionID           string
	DisplayName         string
	StatusText          string
	SupportedNamespaces []string
}
