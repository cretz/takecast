package receiver

import (
	"encoding/json"

	"github.com/cretz/takecast/pkg/receiver/cast_channel"
	"google.golang.org/protobuf/proto"
)

type MessageBuilder struct {
	cast_channel.CastMessage
}

func (m *MessageBuilder) Ref() *cast_channel.CastMessage { return &m.CastMessage }

func (m *MessageBuilder) Send(c interface {
	Send(*cast_channel.CastMessage) error
}) error {
	return c.Send(&m.CastMessage)
}

func (m *MessageBuilder) ApplyReceived(r *cast_channel.CastMessage) *MessageBuilder {
	m.ProtocolVersion = r.ProtocolVersion
	m.SourceId = r.DestinationId
	m.DestinationId = r.SourceId
	m.Namespace = r.Namespace
	return m
}

func (m *MessageBuilder) SetNamespace(n string) *MessageBuilder {
	m.Namespace = &n
	return m
}

func (m *MessageBuilder) MustSetProtoPayload(p proto.Message) *MessageBuilder {
	b, err := proto.Marshal(p)
	if err != nil {
		panic(err)
	}
	return m.SetBinaryPayload(b)
}

func (m *MessageBuilder) SetBinaryPayload(b []byte) *MessageBuilder {
	payloadType := cast_channel.CastMessage_BINARY
	m.PayloadType = &payloadType
	m.PayloadUtf8 = nil
	m.PayloadBinary = b
	return m
}

func (m *MessageBuilder) MustSetJSONPayload(j interface{}) *MessageBuilder {
	b, err := json.Marshal(j)
	if err != nil {
		panic(err)
	}
	return m.SetStringPayload(string(b))
}

func (m *MessageBuilder) SetStringPayload(s string) *MessageBuilder {
	payloadType := cast_channel.CastMessage_STRING
	m.PayloadType = &payloadType
	m.PayloadUtf8 = &s
	m.PayloadBinary = nil
	return m
}
