package webrtc

import (
	"crypto/cipher"
	"encoding/binary"
	"time"

	"github.com/pion/rtp"
)

type Framer struct {
	AES       cipher.Block
	AESIVMask []byte
}

type Frame struct {
	ID       int64
	Audio    bool
	Data     []byte
	Duration time.Duration
}

// Only one write call at a time. Does not reference the packet after this call.
func (f *Framer) Write(p *rtp.Packet) error {
	panic("TODO")
}

// Only one Read call at a time. Returns true if the frame is valid. Completely
// resets the frame each time (no leftover data if reusing the same instance).
func (f *Framer) Read(frame *Frame) (bool, error) {
	panic("TODO")
}

func (f *Framer) decrypt(frame *Frame) {
	// IV is calculated first from putting lower-32 uint32 as big-endian int to
	// bytes 8 through 12
	var iv [16]byte
	binary.BigEndian.PutUint32(iv[8:], uint32(frame.ID))
	// Then xoring the mask
	for i := 0; i < len(iv); i++ {
		iv[i] ^= f.AESIVMask[i]
	}
	// Now AES-CTR
	cipher.NewCTR(f.AES, iv[:]).XORKeyStream(frame.Data, frame.Data)
}
