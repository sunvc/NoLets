package PushToTalk

import (
	"encoding/binary"
	"errors"
)

// PTT WebSocket binary frame protocol.
//
// Wire layout (big-endian):
//
//	Offset  Size  Field
//	0       2     magic         = 0x50 0x54 ("PT")
//	2       1     version       = 0x01
//	3       1     type
//	4       4     seq            (uint32, session-local monotonic)
//	8       4     timestamp_ms   (uint32, ms since session start)
//	12      N     payload        (variable; JSON for control frames, raw Opus packet for AUDIO)
const (
	FrameMagic0     byte = 0x50 // 'P'
	FrameMagic1     byte = 0x54 // 'T'
	FrameVersion    byte = 0x01
	FrameHeaderSize      = 12
)

// Frame types.
const (
	TypeHello          byte = 0x01
	TypeHelloAck       byte = 0x02
	TypeStart          byte = 0x10
	TypeStartBroadcast byte = 0x11
	TypeAudio          byte = 0x12
	TypeEnd            byte = 0x13
	TypeEndBroadcast   byte = 0x14
	TypeLeave          byte = 0x15
	TypeSubscribe      byte = 0x20
	TypeReplayBegin    byte = 0x21
	TypeReplayEnd      byte = 0x22
	TypePresence       byte = 0x30
	TypePing           byte = 0x40
	TypePong           byte = 0x41
	TypeError          byte = 0x50
)

// Well-known error codes carried in TypeError payload.
const (
	ErrCodeSessionExpired = 3002
	ErrCodeNoActiveStart  = 3003
	ErrCodeBadFrame       = 3004
	ErrCodeChannelDenied  = 3005
	ErrCodeInternal       = 3999
)

// Frame is the parsed representation of a wire frame.
type Frame struct {
	Type    byte
	Seq     uint32
	Ts      uint32
	Payload []byte
}

var (
	ErrFrameTooShort   = errors.New("ptt frame: too short")
	ErrBadMagic        = errors.New("ptt frame: bad magic")
	ErrUnknownVersion  = errors.New("ptt frame: unknown version")
	ErrPayloadTooLarge = errors.New("ptt frame: payload too large")
)

// MaxPayloadBytes bounds a single frame payload size. Configurable via WSMaxFrameBytes.
// A hard upper bound is kept here to guard against buggy/malicious peers.
const MaxPayloadBytes = 64 * 1024

// EncodeFrame builds a binary frame. It never returns an error but keeps the
// signature aligned with common expectations so callers can chain naturally.
// payload may be nil for empty control frames (e.g. Ping/Pong).
func EncodeFrame(t byte, seq, ts uint32, payload []byte) []byte {
	buf := make([]byte, FrameHeaderSize+len(payload))
	buf[0] = FrameMagic0
	buf[1] = FrameMagic1
	buf[2] = FrameVersion
	buf[3] = t
	binary.BigEndian.PutUint32(buf[4:8], seq)
	binary.BigEndian.PutUint32(buf[8:12], ts)
	if len(payload) > 0 {
		copy(buf[FrameHeaderSize:], payload)
	}
	return buf
}

// DecodeFrame parses a wire frame. The returned Payload aliases the input
// slice; callers that need to retain the payload past the read loop must copy
// it themselves. This keeps the fan-out hot path zero-copy.
func DecodeFrame(raw []byte) (*Frame, error) {
	if len(raw) < FrameHeaderSize {
		return nil, ErrFrameTooShort
	}
	if raw[0] != FrameMagic0 || raw[1] != FrameMagic1 {
		return nil, ErrBadMagic
	}
	if raw[2] != FrameVersion {
		return nil, ErrUnknownVersion
	}
	f := &Frame{
		Type:    raw[3],
		Seq:     binary.BigEndian.Uint32(raw[4:8]),
		Ts:      binary.BigEndian.Uint32(raw[8:12]),
		Payload: raw[FrameHeaderSize:],
	}
	if len(f.Payload) > MaxPayloadBytes {
		return nil, ErrPayloadTooLarge
	}
	return f, nil
}
