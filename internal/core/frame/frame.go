package frame

import (
	"avia-mesh/internal/config"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
)

type FrameType uint8

type FrameHeader struct {
	Type         FrameType
	Version      uint8
	Ttl          uint8
	MessageId    uint64
	RecipientTag uint32
	PayloadLen   uint8
}

type Frame struct {
	Header  FrameHeader
	Payload []byte
}

const (
	TypeData         FrameType = 0x0
	TypeBeacon       FrameType = 0x1
	TypeSyncRequest  FrameType = 0x2
	TypeSyncResponse FrameType = 0x3
)

var (
	ErrFormatViolated     = errors.New("frame format violated")
	ErrUnsupportedType    = errors.New("unsupported frame type")
	ErrUnsupportedVersion = errors.New("unsupported frame version")
	ErrBufferTooSmall     = errors.New("received buffer is too small")
	ErrPayloadTruncated   = errors.New("payload shorter than stated in header")
)

func (t FrameType) IsValid() bool {
	switch t {
	case TypeData, TypeBeacon, TypeSyncRequest, TypeSyncResponse:
		return true
	default:
		return false
	}
}

func (f *Frame) Serialize(dst []byte) ([]byte, error) {
	err := f.validate()
	if err != nil {
		return nil, err
	}
	f.Header.PayloadLen = uint8(len(f.Payload))
	b0 := (f.Header.Version << 4) | (uint8(f.Header.Type) & 0x0F)
	dst = append(dst, b0)
	dst = append(dst, f.Header.Ttl)

	var bufId [8]byte
	binary.BigEndian.PutUint64(bufId[:], f.Header.MessageId)
	dst = append(dst, bufId[:]...)

	var bufTag [4]byte
	binary.BigEndian.PutUint32(bufTag[:], f.Header.RecipientTag)
	dst = append(dst, bufTag[:]...)

	dst = append(dst, f.Header.PayloadLen)

	dst = append(dst, f.Payload...)
	return dst, nil
}

func ParseFrame(in []byte) (*Frame, error) {
	if len(in) < 15 {
		return nil, fmt.Errorf(
			"%w: buffer too small: requires at least 15 bytes for header, got %d",
			ErrBufferTooSmall, len(in))
	}
	
	headerPayloadLen := in[14]
	actualPayloadLen := len(in) - 15
	
	if actualPayloadLen < int(headerPayloadLen){
		return nil, fmt.Errorf("%w: incomplete payload, got %d bytes instead of %d stated in the header",
			ErrPayloadTruncated, actualPayloadLen, headerPayloadLen)
	}

	// log.warn(payload longer than stated in the header will be cropped)

	ftype := in[0] & 0x0F

	version := in[0] >> 4
	frame := Frame{
		Header: FrameHeader{
			Type:         FrameType(ftype),
			Version:      version,
			Ttl:          in[1],
			MessageId:    binary.BigEndian.Uint64(in[2:10]),
			RecipientTag: binary.BigEndian.Uint32(in[10:14]),
			PayloadLen:   headerPayloadLen,
		},
		Payload: in[15 : 15+int(headerPayloadLen)],
	}
	validationErr := frame.validate()
	if validationErr != nil {
		return nil, validationErr
	}
	return &frame, nil
}

func (f *Frame) validate() error {
	if len(f.Payload) > 255 {
		return fmt.Errorf("%w: payload is 255 bytes only", ErrFormatViolated)
	}

	if (uint8(f.Header.Type) >> 4) != 0 {
		return fmt.Errorf("%w: frame type is 4 higher bits only, got 0x%X", ErrFormatViolated, f.Header.Type)
	}

	if !f.Header.Type.IsValid() {
		return ErrUnsupportedType
	}

	if (f.Header.Version >> 4) != 0 {
		return fmt.Errorf("%w: version is 4 lower bits only, got 0x%X", ErrFormatViolated, f.Header.Version)
	}

	if !slices.Contains(config.SupportedVersions(), f.Header.Version) {
		return fmt.Errorf("%w: unsupported version, got %d", ErrUnsupportedVersion, f.Header.Version)
	}

	return nil
}
