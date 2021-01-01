package receiver

type WebRTCOfferRequestMessage struct {
	*RequestMessageHeader
	SeqNum uint32       `json:"seqNum"`
	Offer  *WebRTCOffer `json:"offer"`
}

type WebRTCOffer struct {
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
	TimeBase   string  `json:"timeBase"`
	SampleRate float64 `json:"sampleRate"`
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

type WebRTCAnswerResponseMessage struct {
	MessageHeader
	SeqNum uint32             `json:"seqNum"`
	Result string             `json:"result,omitempty"`
	Answer *WebRTCAnswer      `json:"answer,omitempty"`
	Error  *WebRTCAnswerError `json:"error,omitempty"`
}

type WebRTCAnswer struct {
	Constraints          *WebRTCAnswerConstraints `json:"constraints,omitempty"`
	Display              *WebRTCAnswerDisplay     `json:"display,omitempty"`
	UDPPort              int                      `json:"udpPort,omitempty"`
	ReceiverGetStatus    bool                     `json:"receiverGetStatus,omitempty"`
	SendIndexes          []int                    `json:"sendIndexes,omitempty"`
	SSRCs                []uint32                 `json:"ssrcs,omitempty"`
	ReceiverRTCPEventLog []int                    `json:"receiverRtcpEventLog,omitempty"`
	ReceiverRTCPDSCP     []int                    `json:"receiverRtcpDscp,omitempty"`
	RTPExtensions        []string                 `json:"rtpExtensions,omitempty"`
}

type WebRTCAnswerError struct {
	Code        int    `json:"code,omitempty"`
	Description string `json:"description,omitempty"`
}

type WebRTCAnswerConstraints struct {
	Audio *WebRTCAnswerConstraintsAudio `json:"audio,omitempty"`
	Video *WebRTCAnswerConstraintsVideo `json:"video,omitempty"`
}

type WebRTCAnswerConstraintsAudio struct {
	MaxSampleRate int   `json:"maxSampleRate,omitempty"`
	MaxChannels   int   `json:"maxChannels,omitempty"`
	MinBitRate    int   `json:"minBitRate,omitempty"`
	MaxBitRate    int   `json:"maxBitRate,omitempty"`
	MaxDelay      int64 `json:"maxDelay,omitempty"`
}

type WebRTCAnswerConstraintsVideo struct {
	MaxPixelsPerSecond float64                 `json:"maxPixelsPerSecond,omitempty"`
	MinDimensions      *WebRTCAnswerDimensions `json:"minDimensions,omitempty"`
	MaxDimensions      *WebRTCAnswerDimensions `json:"maxDimensions,omitempty"`
	MinBitRate         int                     `json:"minBitRate,omitempty"`
	MaxBitRate         int                     `json:"maxBitRate,omitempty"`
	MaxDelay           int64                   `json:"maxDelay,omitempty"`
}

type WebRTCAnswerDimensions struct {
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`
	// As fraction
	FrameRate string `json:"frameRate,omitempty"`
}

type WebRTCAnswerDisplay struct {
	AspectRatio string `json:"aspectRatio,omitempty"`
	Dimensions  string `json:"dimensions,omitempty"`
	// TODO: Scaling 0|1|sender
}
