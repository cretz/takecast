package receiver

type WebRTCOfferRequestMessage struct {
	*RequestMessageHeader
	// mirroring | remoting
	CastMode          string               `json:"castMode"`
	ReceiverGetStatus bool                 `json:"receiverGetStatus"`
	SupportedStreams  []*WebRTCOfferStream `json:"supportedStreams"`
}

type WebRTCOfferStream struct {
	// audio_source | video_source
	Type           string `json:"type"`
	Index          int    `json:"index"`
	Channels       int    `json:"channels"`
	RTPProfile     string `json:"rtpProfile"`
	RTPPayloadType int    `json:"rtpPayloadType"`
	SSRC           uint32 `json:"ssrc"`
	// Hex-encoded bytes
	AESKey string `json:"aesKey"`
	// Hex-encoded bytes
	AESIVMask string `json:"aesIvMask"`
	// Format: "1/N"
	TimeBase string `json:"timeBase"`
	// In milliseconds
	TargetDelay          int                            `json:"targetDelay"`
	ReceiverRTCPEventLog bool                           `json:"receiverRtcpEventLog"`
	BitRate              int                            `json:"bitRate"`
	CodecName            string                         `json:"codecName"`
	Resolutions          []*WebRTCOfferStreamResolution `json:"resolutions"`
	// As fraction
	MaxFrameRate      string `json:"maxFrameRate"`
	Profile           string `json:"profile"`
	Protection        string `json:"protection"`
	MaxBitRate        int    `json:"maxBitRate"`
	Level             string `json:"level"`
	ErrorRecoveryMode string `json:"errorRecoveryMode"`
}

type WebRTCOfferStreamResolution struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func UnmarshalWebRTCRequestMessage(hdr *RequestMessageHeader) (RequestMessage, error) {
	switch hdr.Type {
	case "OFFER":
		return UnmarshalJSONRequestMessage(&WebRTCOfferRequestMessage{RequestMessageHeader: hdr})
	default:
		return nil, nil
	}
}
