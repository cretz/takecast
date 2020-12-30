package receiver

type GetAppAvailabilityRequestMessage struct {
	*RequestMessageHeader
	AppIDs []string `json:"appId"`
}

type GetReceiverStatusRequestMessage struct {
	*RequestMessageHeader
}

type LaunchRequestMessage struct {
	*RequestMessageHeader
}

func UnmarshalReceiverRequestMessage(hdr *RequestMessageHeader) (RequestMessage, error) {
	switch hdr.Type {
	case "GET_APP_AVAILABILITY":
		return UnmarshalJSONRequestMessage(&GetAppAvailabilityRequestMessage{RequestMessageHeader: hdr})
	case "GET_STATUS":
		return &GetReceiverStatusRequestMessage{RequestMessageHeader: hdr}, nil
	case "LAUNCH":
		return &LaunchRequestMessage{RequestMessageHeader: hdr}, nil
	default:
		return nil, nil
	}
}

type ReceiverStatusResponseMessage struct {
	MessageHeader
	Status *ReceiverStatus `json:"status"`
}

type ReceiverStatus struct {
	Applications  []*ApplicationStatus `json:"applications,omitempty"`
	IsActiveInput bool                 `json:"isActiveInput,omitempty"`
	Volume        *Volume              `json:"volume,omitempty"`
}

type ApplicationStatus struct {
	AppID          string                        `json:"appId,omitempty"`
	UniversalAppID string                        `json:"universalAppId,omitempty"`
	DisplayName    string                        `json:"displayName,omitempty"`
	Namespaces     []*ApplicationStatusNamespace `json:"namespaces"`
	SessionID      string                        `json:"sessionId,omitempty"`
	StatusText     string                        `json:"statusText,omitempty"`
	TransportID    string                        `json:"transportId,omitempty"`
	AppType        string                        `json:"appType,omitempty"`
	// TODO:
	// senderApps array
	// appImages array
}

type ApplicationStatusNamespace struct {
	Name string `json:"name"`
}

type Volume struct {
	Level float64 `json:"level,omitempty"`
	Muted bool    `json:"muted"`
}

type GetAppAvailabilityResponseMessage struct {
	MessageHeader
	Availability map[string]string `json:"availability"`
}
