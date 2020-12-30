package mirror

import (
	"context"

	"github.com/cretz/takecast/pkg/receiver"
)

type Mirror interface {
	receiver.Application
}

type mirror struct {
	// TODO
	// metadata *receiver.ApplicationMetadata
	// metadataLock sync.RWMutex
}

type MirrorConfig struct {
	// TODO
	// DefaultMetadata receiver.ApplicationMetadata
}

const (
	AppID          = "0F5096E8"
	AudioOnlyAppID = "85CDB22F"
)

func New(config MirrorConfig) (Mirror, error) {
	// TODO
	return &mirror{}, nil
}

func (m *mirror) Metadata() *receiver.ApplicationMetadata {
	panic("TODO")
}

func (m *mirror) Start(ctx context.Context, id string, params interface{}) error {
	panic("TODO")
}

func (m *mirror) Stop(ctx context.Context) error {
	panic("TODO")
}
