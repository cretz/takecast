package receiver

type ConnectRequestMessage struct {
	*RequestMessageHeader
	ConnType   int                    `json:"connType"`
	Origin     map[string]interface{} `json:"origin"`
	SenderInfo map[string]interface{} `json:"senderInfo"`
	UserAgent  string                 `json:"userAgent"`
}

func UnmarshalConnectionRequestMessage(hdr *RequestMessageHeader) (RequestMessage, error) {
	switch hdr.Type {
	case "CONNECT":
		return UnmarshalJSONRequestMessage(&ConnectRequestMessage{RequestMessageHeader: hdr})
	default:
		return nil, nil
	}
}

type CloseMessage struct {
	MessageHeader
}

func NewCloseMessage() *CloseMessage {
	return &CloseMessage{MessageHeader: MessageHeader{Type: "CLOSE"}}
}
