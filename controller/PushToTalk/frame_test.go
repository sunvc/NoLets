package PushToTalk

import (
	"bytes"
	"testing"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	cases := []struct {
		name    string
		typ     byte
		seq     uint32
		ts      uint32
		payload []byte
	}{
		{"empty payload", TypePing, 0, 0, nil},
		{"control JSON", TypeHello, 1, 0, []byte(`{"id":"u1"}`)},
		{"audio-ish", TypeAudio, 1234, 5678, bytes.Repeat([]byte{0xaa}, 120)},
		{"max seq/ts", TypeAudio, 0xffffffff, 0xffffffff, []byte{1, 2, 3}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			raw := EncodeFrame(c.typ, c.seq, c.ts, c.payload)
			if len(raw) != FrameHeaderSize+len(c.payload) {
				t.Fatalf("bad length: got %d want %d", len(raw), FrameHeaderSize+len(c.payload))
			}
			f, err := DecodeFrame(raw)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if f.Type != c.typ || f.Seq != c.seq || f.Ts != c.ts {
				t.Fatalf("header mismatch: got type=%x seq=%d ts=%d want type=%x seq=%d ts=%d",
					f.Type, f.Seq, f.Ts, c.typ, c.seq, c.ts)
			}
			if !bytes.Equal(f.Payload, c.payload) && !(len(f.Payload) == 0 && len(c.payload) == 0) {
				t.Fatalf("payload mismatch")
			}
		})
	}
}

func TestDecodeErrors(t *testing.T) {
	if _, err := DecodeFrame(nil); err != ErrFrameTooShort {
		t.Fatalf("nil: got %v want ErrFrameTooShort", err)
	}
	if _, err := DecodeFrame(make([]byte, FrameHeaderSize-1)); err != ErrFrameTooShort {
		t.Fatalf("short: got %v want ErrFrameTooShort", err)
	}

	bad := make([]byte, FrameHeaderSize)
	bad[0] = 'X'
	bad[1] = 'Y'
	if _, err := DecodeFrame(bad); err != ErrBadMagic {
		t.Fatalf("bad magic: got %v", err)
	}

	badVer := EncodeFrame(TypePing, 0, 0, nil)
	badVer[2] = 0x02
	if _, err := DecodeFrame(badVer); err != ErrUnknownVersion {
		t.Fatalf("bad version: got %v", err)
	}

	huge := make([]byte, FrameHeaderSize+MaxPayloadBytes+1)
	huge[0] = FrameMagic0
	huge[1] = FrameMagic1
	huge[2] = FrameVersion
	huge[3] = TypeAudio
	if _, err := DecodeFrame(huge); err != ErrPayloadTooLarge {
		t.Fatalf("oversize: got %v", err)
	}
}
