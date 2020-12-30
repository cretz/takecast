package receiver

type PingRequestMessage struct {
	*RequestMessageHeader
}

func UnmarshalHeartbeatRequestMessage(hdr *RequestMessageHeader) (RequestMessage, error) {
	switch hdr.Type {
	case "PING":
		return &PingRequestMessage{RequestMessageHeader: hdr}, nil
	default:
		return nil, nil
	}
}
