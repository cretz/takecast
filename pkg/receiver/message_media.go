package receiver

type GetMediaStatusRequestMessage struct {
	*RequestMessageHeader
}

func UnmarshalMediaRequestMessage(hdr *RequestMessageHeader) (RequestMessage, error) {
	switch hdr.Type {
	case "GET_STATUS":
		return &GetMediaStatusRequestMessage{RequestMessageHeader: hdr}, nil
	default:
		return nil, nil
	}
}
