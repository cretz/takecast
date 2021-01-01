package webrtc

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"net"

	"github.com/cretz/takecast/pkg/receiver"
)

type Session struct {
	net.PacketConn
	ID        string
	Offer     *receiver.WebRTCOffer
	Answer    *receiver.WebRTCAnswer
	Audio     *receiver.WebRTCOfferStream
	Video     *receiver.WebRTCOfferStream
	AES       cipher.Block
	AESIVMask []byte
}

func StartSession(id string, offer *receiver.WebRTCOffer) (*Session, error) {
	s := &Session{
		ID:     id,
		Offer:  offer,
		Answer: &receiver.WebRTCAnswer{},
	}
	success := false
	defer func() {
		if !success {
			s.Close()
		}
	}()
	// Extract keys
	if aesKey, err := hex.DecodeString(offer.SupportedStreams[0].AESKey); err != nil {
		return nil, fmt.Errorf("bad key: %w", err)
	} else if s.AES, err = aes.NewCipher(aesKey); err != nil {
		return nil, fmt.Errorf("bad key: %w", err)
	} else if s.AESIVMask, err = hex.DecodeString(offer.SupportedStreams[0].AESIVMask); err != nil {
		return nil, fmt.Errorf("bad iv: %w", err)
	}
	// Listen UDP
	var err error
	if s.PacketConn, err = net.ListenUDP("udp", nil); err != nil {
		return nil, err
	}
	s.Answer.UDPPort = s.LocalAddr().(*net.UDPAddr).Port
	// Take first audio and video stream only
	for i, stream := range offer.SupportedStreams {
		// Check that all keys are the same
		if offer.SupportedStreams[0].AESKey != stream.AESKey || offer.SupportedStreams[0].AESIVMask != stream.AESIVMask {
			return nil, fmt.Errorf("mismatched key/salt")
		}
		// Set answer index and also set ssrc as one more than given
		if stream.Type == "audio_source" && s.Audio == nil {
			s.Audio = stream
		} else if stream.Type == "video_stream" && s.Video == nil {
			s.Video = stream
		} else {
			continue
		}
		s.Answer.SendIndexes = append(s.Answer.SendIndexes, i)
		s.Answer.SSRCs = append(s.Answer.SSRCs, stream.SSRC+1)
	}
	success = true
	return s, nil
}

func (s *Session) Read(p *Packet) error {
	// Use rtp packet Raw as scratch space. If it's under MTU capacity, create
	// new byte slice, otherwise resize.
	const mtu = 1460
	if cap(p.scratch.Raw) < mtu {
		p.scratch.Raw = make([]byte, mtu)
	} else {
		p.scratch.Raw = p.scratch.Raw[:mtu]
	}
	// Read
	n, _, err := s.ReadFrom(p.scratch.Raw)
	if err != nil {
		return err
	}
	return p.Unmarshal(p.scratch.Raw[:n])
}
