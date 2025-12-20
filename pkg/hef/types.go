package hef

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// HEF file format constants
const (
	HefMagic        = 0x01484546 // "FEH\x01" in little-endian
	HefHeaderSizeV0 = 32
	HefHeaderSizeV1 = 32
	HefHeaderSizeV2 = 40
	HefHeaderSizeV3 = 56

	HefVersionV0 = 0
	HefVersionV1 = 1
	HefVersionV2 = 2
	HefVersionV3 = 3
)

// Errors
var (
	ErrInvalidMagic    = errors.New("invalid HEF magic number")
	ErrUnsupportedVersion = errors.New("unsupported HEF version")
	ErrTruncatedHeader = errors.New("truncated HEF header")
	ErrInvalidChecksum = errors.New("invalid HEF checksum")
	ErrTruncatedData   = errors.New("truncated HEF data")
)

// HefHeader represents the common HEF header fields
type HefHeader struct {
	Magic        uint32
	Version      uint32
	HefProtoSize uint32
}

// HefHeaderV0 extends HefHeader with V0-specific fields
type HefHeaderV0 struct {
	HefHeader
	Reserved    uint32
	ExpectedMD5 [16]byte
}

// HefHeaderV1 extends HefHeader with V1-specific fields
type HefHeaderV1 struct {
	HefHeader
	Crc      uint32
	CcwsSize uint64
	Reserved uint32
}

// HefHeaderV2 extends HefHeader with V2-specific fields
type HefHeaderV2 struct {
	HefHeader
	Xxh3Hash  uint64
	CcwsSize  uint64
	Reserved1 uint64
	Reserved2 uint64
}

// HefHeaderV3 extends HefHeader with V3-specific fields
type HefHeaderV3 struct {
	HefHeader
	Xxh3Hash              uint64
	CcwsSizeWithPadding   uint64
	HefPaddingSize        uint32
	_                     uint32 // padding
	AdditionalInfoSize    uint64
	Reserved1             uint64
	Reserved2             uint64
}

// DeviceArchitecture represents the target Hailo device
type DeviceArchitecture uint32

const (
	ArchHailo8   DeviceArchitecture = 0
	ArchHailo8P  DeviceArchitecture = 1
	ArchHailo8R  DeviceArchitecture = 2
	ArchHailo8L  DeviceArchitecture = 3
	ArchHailo15H DeviceArchitecture = 103
	ArchHailo15M DeviceArchitecture = 4
)

func (a DeviceArchitecture) String() string {
	switch a {
	case ArchHailo8:
		return "Hailo-8"
	case ArchHailo8P:
		return "Hailo-8P"
	case ArchHailo8R:
		return "Hailo-8R"
	case ArchHailo8L:
		return "Hailo-8L"
	case ArchHailo15H:
		return "Hailo-15H"
	case ArchHailo15M:
		return "Hailo-15M"
	default:
		return fmt.Sprintf("Unknown(%d)", a)
	}
}

// FormatType represents tensor data types
type FormatType uint32

const (
	FormatTypeAuto    FormatType = 0
	FormatTypeUint8   FormatType = 1
	FormatTypeUint16  FormatType = 2
	FormatTypeFloat32 FormatType = 3
)

func (f FormatType) String() string {
	switch f {
	case FormatTypeAuto:
		return "auto"
	case FormatTypeUint8:
		return "uint8"
	case FormatTypeUint16:
		return "uint16"
	case FormatTypeFloat32:
		return "float32"
	default:
		return fmt.Sprintf("Unknown(%d)", f)
	}
}

// FormatOrder represents tensor memory layout
type FormatOrder uint32

const (
	FormatOrderNHWC           FormatOrder = 1
	FormatOrderNCHW           FormatOrder = 11
	FormatOrderNV12           FormatOrder = 13
	FormatOrderHailoNmsByClass FormatOrder = 100
	FormatOrderHailoNmsByScore FormatOrder = 101
)

func (f FormatOrder) String() string {
	switch f {
	case FormatOrderNHWC:
		return "NHWC"
	case FormatOrderNCHW:
		return "NCHW"
	case FormatOrderNV12:
		return "NV12"
	case FormatOrderHailoNmsByClass:
		return "NMS_BY_CLASS"
	case FormatOrderHailoNmsByScore:
		return "NMS_BY_SCORE"
	default:
		return fmt.Sprintf("Unknown(%d)", f)
	}
}

// StreamDirection represents input or output
type StreamDirection uint32

const (
	StreamDirectionInput  StreamDirection = 0
	StreamDirectionOutput StreamDirection = 1
)

// ImageShape3D represents a 3D tensor shape
type ImageShape3D struct {
	Height   uint32
	Width    uint32
	Features uint32
}

// Format represents tensor format information
type Format struct {
	Type  FormatType
	Order FormatOrder
}

// QuantInfo represents quantization parameters
type QuantInfo struct {
	ZeroPoint float32
	Scale     float32
	LimMin    float32
	LimMax    float32
}

// NmsShape represents NMS layer shape information
type NmsShape struct {
	MaxBboxesPerClass  uint32
	NumberOfClasses    uint32
	MaxAccumulatedMask uint32
}

// StreamInfo represents low-level stream information
type StreamInfo struct {
	Name        string
	Direction   StreamDirection
	Format      Format
	Shape       ImageShape3D
	HwShape     ImageShape3D
	HwFrameSize uint64
	QuantInfo   QuantInfo
}

// VStreamInfo represents high-level virtual stream information
type VStreamInfo struct {
	Name        string
	NetworkName string
	Direction   StreamDirection
	Format      Format
	Shape       ImageShape3D
	QuantInfo   QuantInfo
	NmsShape    NmsShape
	IsNms       bool
}

// NetworkGroupInfo represents information about a network group
type NetworkGroupInfo struct {
	Name               string
	InputStreams       []StreamInfo
	OutputStreams      []StreamInfo
	InputVStreams      []VStreamInfo
	OutputVStreams     []VStreamInfo
	BottleneckFps      float64
	IsMultiContext     bool
}

// NetworkInfo represents information about a single network
type NetworkInfo struct {
	Name        string
	GroupName   string
}

// Hef represents a parsed HEF file
type Hef struct {
	Version         uint32
	DeviceArch      DeviceArchitecture
	NetworkGroups   []NetworkGroupInfo
	Hash            string
	rawData         []byte
}

// ParseHeader parses the HEF header from raw bytes
func ParseHeader(data []byte) (*HefHeader, error) {
	if len(data) < 12 {
		return nil, ErrTruncatedHeader
	}

	header := &HefHeader{
		Magic:        binary.LittleEndian.Uint32(data[0:4]),
		Version:      binary.LittleEndian.Uint32(data[4:8]),
		HefProtoSize: binary.LittleEndian.Uint32(data[8:12]),
	}

	if header.Magic != HefMagic {
		return nil, fmt.Errorf("%w: got 0x%08X, expected 0x%08X", ErrInvalidMagic, header.Magic, HefMagic)
	}

	return header, nil
}

// HeaderSize returns the header size for a given HEF version
func HeaderSize(version uint32) (int, error) {
	switch version {
	case HefVersionV0:
		return HefHeaderSizeV0, nil
	case HefVersionV1:
		return HefHeaderSizeV1, nil
	case HefVersionV2:
		return HefHeaderSizeV2, nil
	case HefVersionV3:
		return HefHeaderSizeV3, nil
	default:
		return 0, fmt.Errorf("%w: version %d", ErrUnsupportedVersion, version)
	}
}

// ParseHeaderV0 parses a V0 header
func ParseHeaderV0(data []byte) (*HefHeaderV0, error) {
	if len(data) < HefHeaderSizeV0 {
		return nil, ErrTruncatedHeader
	}

	baseHeader, err := ParseHeader(data)
	if err != nil {
		return nil, err
	}

	if baseHeader.Version != HefVersionV0 {
		return nil, fmt.Errorf("expected V0, got V%d", baseHeader.Version)
	}

	header := &HefHeaderV0{
		HefHeader: *baseHeader,
		Reserved:  binary.LittleEndian.Uint32(data[12:16]),
	}
	copy(header.ExpectedMD5[:], data[16:32])

	return header, nil
}

// ParseHeaderV2 parses a V2 header
func ParseHeaderV2(data []byte) (*HefHeaderV2, error) {
	if len(data) < HefHeaderSizeV2 {
		return nil, ErrTruncatedHeader
	}

	baseHeader, err := ParseHeader(data)
	if err != nil {
		return nil, err
	}

	if baseHeader.Version != HefVersionV2 {
		return nil, fmt.Errorf("expected V2, got V%d", baseHeader.Version)
	}

	header := &HefHeaderV2{
		HefHeader: *baseHeader,
		Xxh3Hash:  binary.LittleEndian.Uint64(data[12:20]),
		CcwsSize:  binary.LittleEndian.Uint64(data[20:28]),
		Reserved1: binary.LittleEndian.Uint64(data[28:36]),
	}

	return header, nil
}

// ParseHeaderV3 parses a V3 header
func ParseHeaderV3(data []byte) (*HefHeaderV3, error) {
	if len(data) < HefHeaderSizeV3 {
		return nil, ErrTruncatedHeader
	}

	baseHeader, err := ParseHeader(data)
	if err != nil {
		return nil, err
	}

	if baseHeader.Version != HefVersionV3 {
		return nil, fmt.Errorf("expected V3, got V%d", baseHeader.Version)
	}

	header := &HefHeaderV3{
		HefHeader:             *baseHeader,
		Xxh3Hash:              binary.LittleEndian.Uint64(data[12:20]),
		CcwsSizeWithPadding:   binary.LittleEndian.Uint64(data[20:28]),
		HefPaddingSize:        binary.LittleEndian.Uint32(data[28:32]),
		AdditionalInfoSize:    binary.LittleEndian.Uint64(data[36:44]),
		Reserved1:             binary.LittleEndian.Uint64(data[44:52]),
	}

	return header, nil
}
