package receiver

import (
	"encoding/json"
	"fmt"

	"github.com/cretz/takecast/pkg/receiver/cast_channel"
)

type Message interface {
	Header() *MessageHeader
}

type MessageHeader struct {
	Type      string `json:"type"`
	RequestID int    `json:"requestId,omitempty"`
}

func (m *MessageHeader) Header() *MessageHeader { return m }

type RequestMessage interface {
	Header() *RequestMessageHeader
}

type RequestMessageHeader struct {
	MessageHeader
	Raw *cast_channel.CastMessage `json:"-"`
}

func (r *RequestMessageHeader) Header() *RequestMessageHeader { return r }

type UnknownRequestMessage struct {
	*RequestMessageHeader
}

func (r *RequestMessageHeader) UnmarshalHeader() error {
	if r.Raw.PayloadUtf8 == nil {
		return fmt.Errorf("missing string payload")
	}
	return json.Unmarshal([]byte(*r.Raw.PayloadUtf8), r)
}

func UnmarshalJSONRequestMessage(r RequestMessage) (RequestMessage, error) {
	return r, json.Unmarshal([]byte(*r.Header().Raw.PayloadUtf8), r)
}

func UnmarshalRequestMessage(raw *cast_channel.CastMessage) (msg RequestMessage, err error) {
	// Only auth doesn't have a JSON payload
	hdr := &RequestMessageHeader{Raw: raw}
	if hdr.Raw.GetNamespace() != "urn:x-cast:com.google.cast.tp.deviceauth" {
		if err = hdr.UnmarshalHeader(); err != nil {
			return nil, err
		}
	}
	switch ns := hdr.Raw.GetNamespace(); ns {
	case "urn:x-cast:com.google.cast.media":
		msg, err = UnmarshalMediaRequestMessage(hdr)
	case "urn:x-cast:com.google.cast.receiver":
		msg, err = UnmarshalReceiverRequestMessage(hdr)
	case "urn:x-cast:com.google.cast.tp.connection":
		msg, err = UnmarshalConnectionRequestMessage(hdr)
	case "urn:x-cast:com.google.cast.tp.deviceauth":
		msg, err = UnmarshalDeviceAuthRequestMessage(hdr)
	case "urn:x-cast:com.google.cast.tp.heartbeat":
		msg, err = UnmarshalHeartbeatRequestMessage(hdr)
	}
	if msg == nil && err == nil {
		msg = &UnknownRequestMessage{RequestMessageHeader: hdr}
	}
	return
}
