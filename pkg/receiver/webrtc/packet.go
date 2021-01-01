package webrtc

import (
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

type Packet struct {
	RTP  *rtp.Packet
	RTCP []rtcp.Packet

	scratch rtp.Packet
}

func (p *Packet) Unmarshal(b []byte) error {
	var err error
	p.RTP, p.RTCP = nil, nil
	if isRTP, isRTCP := packetType(b); isRTP {
		p.RTP = &p.scratch
		err = p.RTP.Unmarshal(b)
	} else if isRTCP {
		p.RTCP, err = rtcp.Unmarshal(b)
	}
	return err
}

func packetType(b []byte) (rtp, rtcp bool) {
	// Logic from pion/webrtc/internal/mux/muxfunc.go
	if len(b) < 2 || b[0] < 128 || b[0] > 191 {
		return
	}
	rtcp = b[1] >= 192 && b[1] <= 223
	return !rtcp, rtcp
}
