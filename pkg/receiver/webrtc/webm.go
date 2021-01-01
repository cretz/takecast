package webrtc

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/at-wat/ebml-go/webm"
)

// Does not close session but does close writer. Always returns error, may be
// EOF on session end.
func SaveSessionToWebM(ctx context.Context, w io.WriteCloser, s *Session) error {
	defer w.Close()
	// Create webm tracks
	tracks := make([]webm.TrackEntry, 0, 2)
	if s.Audio != nil {
		if s.Audio.CodecName != "opus" {
			return fmt.Errorf("expected opus audio codec, got %v", s.Audio.CodecName)
		}
		track := webm.TrackEntry{
			Name:            s.Audio.Type,
			TrackNumber:     uint64(len(tracks) + 1),
			TrackUID:        uint64(s.Audio.SSRC),
			CodecID:         "A_OPUS",
			TrackType:       2,
			DefaultDuration: 20000000,
			Audio:           &webm.Audio{SamplingFrequency: 48000, Channels: 2},
		}
		if s.Audio.SampleRate > 0 {
			track.Audio.SamplingFrequency = s.Audio.SampleRate
		}
		if s.Audio.Channels > 0 {
			track.Audio.Channels = uint64(s.Audio.Channels)
		}
		tracks = append(tracks, track)
	}
	if s.Video != nil {
		if s.Audio.CodecName != "vp8" {
			return fmt.Errorf("expected vp8 audio codec, got %v", s.Video.CodecName)
		}
		track := webm.TrackEntry{
			Name:            s.Video.Type,
			TrackNumber:     uint64(len(tracks) + 1),
			TrackUID:        uint64(s.Video.SSRC),
			CodecID:         "A_VP8",
			TrackType:       1,
			DefaultDuration: 33333333,
			// TODO: Should I wait for first video frame to set dims?
			Video: &webm.Video{PixelHeight: 240, PixelWidth: 320},
		}
		// Use first resolution
		if len(s.Video.Resolutions) > 0 {
			track.Video.PixelHeight = uint64(s.Video.Resolutions[0].Height)
			track.Video.PixelWidth = uint64(s.Video.Resolutions[0].Width)
		}
		tracks = append(tracks, track)
	}
	// Create writers
	blockWriters, err := webm.NewSimpleBlockWriter(w, tracks)
	if err != nil {
		return err
	}
	defer func() {
		for _, blockWriter := range blockWriters {
			// Ignore error
			blockWriter.Close()
		}
	}()
	// Run in the background so context can close this
	errCh := make(chan error, 1)
	go func() { errCh <- pipeWebM(blockWriters, s) }()
	// Finish when context is done or error
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

// Doesn't close anything
func pipeWebM(w []webm.BlockWriteCloser, s *Session) error {
	// Create the framer
	framer := &Framer{AES: s.AES, AESIVMask: s.AESIVMask}
	// Keep reading/writing until error
	var p Packet
	var f Frame
	var audioTimestamp, videoTimestamp time.Duration
	for {
		// Get RTP packet, write to framer, try to read frame
		if err := s.Read(&p); err != nil {
			return err
		} else if p.RTP == nil {
			continue
		} else if err := framer.Write(p.RTP); err != nil {
			return err
		} else if ok, err := framer.Read(&f); err != nil {
			return err
		} else if !ok {
			continue
		}
		// Write to block writers
		if f.Audio {
			audioTimestamp += f.Duration
			if _, err := w[0].Write(true, int64(audioTimestamp/time.Millisecond), f.Data); err != nil {
				return fmt.Errorf("failed writing audio: %w", err)
			}
		} else {
			videoTimestamp += f.Duration
			if _, err := w[len(w)-1].Write(f.Data[0]&0x1 == 0, int64(videoTimestamp/time.Millisecond), f.Data); err != nil {
				return fmt.Errorf("failed writing video: %w", err)
			}
		}
	}
}
