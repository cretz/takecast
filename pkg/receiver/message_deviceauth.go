package receiver

import (
	"fmt"

	"github.com/cretz/takecast/pkg/receiver/cast_channel"
	"google.golang.org/protobuf/proto"
)

type DeviceAuthRequestMessage struct {
	*RequestMessageHeader
	cast_channel.DeviceAuthMessage
}

func UnmarshalDeviceAuthRequestMessage(hdr *RequestMessageHeader) (RequestMessage, error) {
	d := &DeviceAuthRequestMessage{RequestMessageHeader: hdr}
	if err := proto.Unmarshal(hdr.Raw.PayloadBinary, d); err != nil {
		return nil, fmt.Errorf("failed unmarshaling auth message: %w", err)
	}
	return d, nil
}
