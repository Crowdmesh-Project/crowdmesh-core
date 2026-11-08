package frame

import (
	"bytes"
	"errors"
	"testing"
)

func TestFrame_RoundTripSuccess(t *testing.T) {
	payload := []byte("test message")
	original := Frame{
		Header: FrameHeader{
			Type:         TypeData,
			Version:      0,
			Ttl:          10,
			MessageId:    0x0123456789ABCDEF,
			RecipientTag: 0x01234567,
			PayloadLen:   uint8(len(payload)),
		},
		Payload: payload,
	}

	rawBytes, err := original.Serialize(nil)
	if err != nil {
		t.Fatalf("unexpected serialize error: %v", err)
	}

	parsed, err := ParseFrame(rawBytes)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if parsed.Header != original.Header {
		t.Errorf("unexpected header:\ngot:  %+v\nwant: %+v", parsed.Header, original.Header)
	}

	if !bytes.Equal(parsed.Payload, original.Payload) {
		t.Errorf("unexpected payload: got %s, want %s", parsed.Payload, original.Payload)
	}
}

func TestParseFrame_Errors(t *testing.T) {
	payload := []byte("test message")
	validFrame := Frame{
		Header: FrameHeader{
			Type:         TypeData,
			Version:      0,
			Ttl:          10,
			MessageId:    0x0123456789ABCDEF,
			RecipientTag: 0x01234567,
			PayloadLen:   uint8(len(payload)),
		},
		Payload: payload,
	}
	validBytes, err := validFrame.Serialize(nil)
	if err != nil {
		t.Fatalf("unexpected serialize error: %v", err)
	}

	t.Run("unsupported type", func(t *testing.T) {
		corruptedBytes := append([]byte(nil), validBytes...)
		version := corruptedBytes[0] & 0xF0
		for badType := byte(0x04); badType <= byte(0x0F); badType++ {
			corruptedBytes[0] = version | badType
			_, err := ParseFrame(corruptedBytes)
			if !errors.Is(err, ErrUnsupportedType) {
				t.Errorf("expected ErrUnsupportedType for invalid type %d, got %v", badType, err)
			}
		}
	})

	t.Run("unsupported version", func(t *testing.T) {
		corruptedBytes := append([]byte(nil), validBytes...)
		ftype := corruptedBytes[0] & 0x0F
		for badVersion := byte(0x01); badVersion <= byte(0x0F); badVersion++ {
			badVersionAligned := badVersion << 4
			corruptedBytes[0] = badVersionAligned | ftype
			_, err := ParseFrame(corruptedBytes)
			if !errors.Is(err, ErrUnsupportedVersion) {
				t.Errorf("expected ErrUnsupportedVersion for invalid version %d, got %v", badVersion, err)
			}
		}
	})

	t.Run("buffer too small", func(t *testing.T) {
		// Bufory od 0 do 14 bajtów są za krótkie na 15-bajtowy nagłówek
		for size := 0; size < 15; size++ {
			shortData := validBytes[:size]
			_, err := ParseFrame(shortData)
			if !errors.Is(err, ErrBufferTooSmall) {
				t.Errorf("expected ErrBufferTooSmall for size %d, got %v", size, err)
			}
		}
	})

	t.Run("payload truncated", func(t *testing.T) {
		// Nagłówek deklaruje 12 bajtów payloadu (razem 27B).
		// Każdy bufor od 15 do 26 bajtów ma ucięty payload:
		for size := 15; size < len(validBytes); size++ {
			truncatedData := validBytes[:size]
			_, err := ParseFrame(truncatedData)
			if !errors.Is(err, ErrPayloadTruncated) {
				t.Errorf("expected ErrPayloadTruncated for size %d, got %v", size, err)
			}
		}
	})
}

func TestSerialize_Errors(t *testing.T) {
	t.Run("payload too large", func(t *testing.T) {
		hugePayload := make([]byte, 256) // max allowed is 255
		f := Frame{
			Header: FrameHeader{
				Type:    TypeData,
				Version: 0,
			},
			Payload: hugePayload,
		}
		_, err := f.Serialize(nil)
		if !errors.Is(err, ErrFormatViolated) {
			t.Errorf("expected ErrFormatViolated for payload > 255 bytes, got %v", err)
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		f := Frame{
			Header: FrameHeader{
				Type:    FrameType(0x05),
				Version: 0,
			},
			Payload: []byte("test"),
		}
		_, err := f.Serialize(nil)
		if !errors.Is(err, ErrUnsupportedType) {
			t.Errorf("expected ErrUnsupportedType, got %v", err)
		}
	})

	t.Run("unsupported version", func(t *testing.T) {
		f := Frame{
			Header: FrameHeader{
				Type:    TypeData,
				Version: 9,
			},
			Payload: []byte("test"),
		}
		_, err := f.Serialize(nil)
		if !errors.Is(err, ErrUnsupportedVersion) {
			t.Errorf("expected ErrUnsupportedVersion, got %v", err)
		}
	})
}
func FuzzParseFrame(f *testing.F) {
	sampleFrame := Frame{
		Header: FrameHeader{
			Type:         TypeData,
			Version:      0,
			Ttl:          10,
			MessageId:    42,
			RecipientTag: 100,
		},
		Payload: []byte("corpus seed payload"),
	}
	validBytes, _ := sampleFrame.Serialize(nil)
	f.Add(validBytes)
	f.Add([]byte{})
	f.Add([]byte{0x00, 0x05, 0x01, 0x02})
	f.Add(make([]byte, 15))
	f.Add(make([]byte, 270))
	f.Fuzz(func(t *testing.T, data []byte) {
		parsed, err := ParseFrame(data)
		if err != nil {
			return 
		}
		_, err = parsed.Serialize(nil)
		if err != nil {
			t.Fatalf("serialize failed on parsed frame: %v", err)
		}
	})
}