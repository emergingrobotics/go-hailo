// Package control implements the Hailo firmware control protocol.
// This package handles communication with the Hailo device firmware
// for operations like enabling/disabling the neural network accelerator.
package control

import (
	"encoding/binary"
	"fmt"

	"github.com/anthropics/purple-hailo/pkg/driver"
)

// Control protocol opcodes from control_protocol.h
// These are zero-indexed enum values from CONTROL_PROTOCOL__OPCODES_VARIABLES
const (
	OpcodeIdentify                      = 0  // HAILO_CONTROL_OPCODE_IDENTIFY
	OpcodeWriteMemory                   = 1  // HAILO_CONTROL_OPCODE_WRITE_MEMORY
	OpcodeReadMemory                    = 2  // HAILO_CONTROL_OPCODE_READ_MEMORY
	OpcodeConfigStream                  = 3  // HAILO_CONTROL_OPCODE_CONFIG_STREAM
	OpcodeOpenStream                    = 4  // HAILO_CONTROL_OPCODE_OPEN_STREAM
	OpcodeCloseStream                   = 5  // HAILO_CONTROL_OPCODE_CLOSE_STREAM
	OpcodeReset                         = 7  // HAILO_CONTROL_OPCODE_RESET
	OpcodeSetNetworkGroupHeader         = 32 // HAILO_CONTROL_OPCODE_CONTEXT_SWITCH_SET_NETWORK_GROUP_HEADER (line 33)
	OpcodeSetContextInfo                = 33 // HAILO_CONTROL_OPCODE_CONTEXT_SWITCH_SET_CONTEXT_INFO (line 34)
	OpcodeDownloadContextActionList     = 36 // HAILO_CONTROL_OPCODE_DOWNLOAD_CONTEXT_ACTION_LIST (line 37)
	OpcodeChangeContextSwitchStatus     = 37 // HAILO_CONTROL_OPCODE_CHANGE_CONTEXT_SWITCH_STATUS (line 38)
	OpcodeCoreIdentify                  = 42 // HAILO_CONTROL_OPCODE_CORE_IDENTIFY (line 43)
	OpcodeClearConfiguredApps           = 71 // HAILO_CONTROL_OPCODE_CONTEXT_SWITCH_CLEAR_CONFIGURED_APPS (line 158)
)

// Context switch status values from control_protocol.h
const (
	ContextSwitchStatusReset   uint8 = 0 // CONTROL_PROTOCOL__CONTEXT_SWITCH_STATUS_RESET
	ContextSwitchStatusEnabled uint8 = 1 // CONTROL_PROTOCOL__CONTEXT_SWITCH_STATUS_ENABLED
	ContextSwitchStatusPaused  uint8 = 2 // CONTROL_PROTOCOL__CONTEXT_SWITCH_STATUS_PAUSED
)

// CPU IDs for firmware communication
const (
	CpuIdAppCpu  driver.CpuId = 0 // Application CPU (CPU_ID_APP_CPU)
	CpuIdCoreCpu driver.CpuId = 1 // Core CPU (CPU_ID_CORE_CPU) - used for context switch
)

// Protocol constants
const (
	ProtocolVersion    = 2    // CONTROL_PROTOCOL__PROTOCOL_VERSION
	MaxControlLength   = 1500 // CONTROL_PROTOCOL__MAX_CONTROL_LENGTH
	DefaultTimeoutMs   = 5000 // 5 second timeout for firmware commands
	MaxContextSize     = 4096 // CONTROL_PROTOCOL__MAX_CONTEXT_SIZE
	RequestHeaderSize  = 16   // Size of CONTROL_PROTOCOL__request_header_t
	ResponseHeaderSize = 24   // Size of CONTROL_PROTOCOL__response_header_t
)

// Context types from CONTROL_PROTOCOL__context_switch_context_type_t
const (
	ContextTypePreliminary   uint8 = 0 // CONTROL_PROTOCOL__CONTEXT_SWITCH_CONTEXT_TYPE_PRELIMINARY
	ContextTypeDynamic       uint8 = 1 // CONTROL_PROTOCOL__CONTEXT_SWITCH_CONTEXT_TYPE_DYNAMIC
	ContextTypeBatchSwitching uint8 = 2 // CONTROL_PROTOCOL__CONTEXT_SWITCH_CONTEXT_TYPE_BATCH_SWITCHING
	ContextTypeActivation    uint8 = 3 // CONTROL_PROTOCOL__CONTEXT_SWITCH_CONTEXT_TYPE_ACTIVATION
)

// ContextInfoChunk represents a chunk of context configuration data
// Matches CONTROL_PROTOCOL__context_switch_context_info_chunk_t
type ContextInfoChunk struct {
	IsFirstChunk bool
	IsLastChunk  bool
	ContextType  uint8
	Data         []byte // The action list data for this context
}

// Special values
const (
	IgnoreNetworkGroupIndex uint8  = 255
	IgnoreDynamicBatchSize  uint16 = 0
	DefaultBatchCount       uint16 = 0 // 0 means infinite
	InfiniteBatchCount      uint16 = 0
)

// RequestHeader represents the control protocol request header.
// Matches CONTROL_PROTOCOL__request_header_t (packed, 16 bytes)
type RequestHeader struct {
	Version  uint32 // Protocol version (2)
	Flags    uint32 // ACK flag in bit 0
	Sequence uint32 // Incrementing sequence number
	Opcode   uint32 // Command opcode
}

// ResponseHeader represents the control protocol response header.
// Matches CONTROL_PROTOCOL__response_header_t (packed, 24 bytes)
type ResponseHeader struct {
	Version     uint32
	Flags       uint32
	Sequence    uint32
	Opcode      uint32
	MajorStatus uint32
	MinorStatus uint32
}

// PackRequestHeader creates a control protocol request header as bytes.
// Note: The Hailo firmware uses network byte order (big-endian) for all fields.
func PackRequestHeader(sequence, opcode uint32) []byte {
	buf := make([]byte, RequestHeaderSize)
	binary.BigEndian.PutUint32(buf[0:4], ProtocolVersion)
	binary.BigEndian.PutUint32(buf[4:8], 0) // flags = 0 (no ACK request)
	binary.BigEndian.PutUint32(buf[8:12], sequence)
	binary.BigEndian.PutUint32(buf[12:16], opcode)
	return buf
}

// packParameter packs a parameter with its length prefix.
// Format: uint32 length (big-endian) + data bytes
func packParameter(data []byte) []byte {
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(data)))
	return append(lenBuf, data...)
}

// PackChangeContextSwitchStatusRequest creates the request to enable/disable the state machine.
// Matches CONTROL_PROTOCOL__change_context_switch_status_request_t
// Note: All multi-byte fields use network byte order (big-endian).
func PackChangeContextSwitchStatusRequest(sequence uint32, status, appIndex uint8,
	dynamicBatchSize, batchCount uint16) []byte {

	header := PackRequestHeader(sequence, OpcodeChangeContextSwitchStatus)

	// Parameter count (4 parameters) - big-endian
	paramCount := make([]byte, 4)
	binary.BigEndian.PutUint32(paramCount, 4)

	// Parameter 1: state_machine_status (1 byte)
	param1 := packParameter([]byte{status})

	// Parameter 2: application_index (1 byte)
	param2 := packParameter([]byte{appIndex})

	// Parameter 3: dynamic_batch_size (2 bytes, big-endian)
	batchSizeBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(batchSizeBuf, dynamicBatchSize)
	param3 := packParameter(batchSizeBuf)

	// Parameter 4: batch_count (2 bytes, big-endian)
	batchCountBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(batchCountBuf, batchCount)
	param4 := packParameter(batchCountBuf)

	// Combine all parts
	result := append(header, paramCount...)
	result = append(result, param1...)
	result = append(result, param2...)
	result = append(result, param3...)
	result = append(result, param4...)

	return result
}

// ParseResponseHeader parses a response header from bytes.
// Note: The Hailo firmware uses network byte order (big-endian) for all fields.
func ParseResponseHeader(data []byte) (*ResponseHeader, error) {
	if len(data) < ResponseHeaderSize {
		return nil, fmt.Errorf("response too short: %d bytes, need %d", len(data), ResponseHeaderSize)
	}

	return &ResponseHeader{
		Version:     binary.BigEndian.Uint32(data[0:4]),
		Flags:       binary.BigEndian.Uint32(data[4:8]),
		Sequence:    binary.BigEndian.Uint32(data[8:12]),
		Opcode:      binary.BigEndian.Uint32(data[12:16]),
		MajorStatus: binary.BigEndian.Uint32(data[16:20]),
		MinorStatus: binary.BigEndian.Uint32(data[20:24]),
	}, nil
}

// MaxNetworksPerNetworkGroup is the max networks in a network group (v4.20.0)
const MaxNetworksPerNetworkGroup = 8

// ApplicationHeader represents CONTROL_PROTOCOL__application_header_t
// This matches the v4.20.0 firmware format (41 bytes packed)
// Note: v4.21.0+ changed this struct significantly (batch_size became single, added config_channels)
type ApplicationHeader struct {
	DynamicContextsCount      uint16
	PreliminaryRunAsap        bool
	BatchRegisterConfig       bool
	CanFastBatchSwitch        bool
	IsAbbaleSupported         bool
	NetworksCount             uint8
	CsmBufferSize             uint16
	BatchSize                 [MaxNetworksPerNetworkGroup]uint16 // Array of batch sizes per network (v4.20.0)
	ExternalActionListAddress uint32
	BoundaryChannelsBitmap    [3]uint32
}

// ApplicationHeaderSizeV420 is the size of ApplicationHeader for v4.20.0 firmware
const ApplicationHeaderSizeV420 = 41

// PackApplicationHeader packs the application header for SET_NETWORK_GROUP_HEADER
// Note: The struct fields use LITTLE-ENDIAN because the firmware memcpy's the raw struct
// (unlike the request header which uses network byte order)
// This matches the v4.20.0 firmware format (41 bytes packed)
func PackApplicationHeader(header *ApplicationHeader) []byte {
	// Total struct size is exactly 41 bytes for v4.20.0 (matching C struct with #pragma pack(1))
	buf := make([]byte, ApplicationHeaderSizeV420)
	offset := 0

	// dynamic_contexts_count (2 bytes) - little-endian
	binary.LittleEndian.PutUint16(buf[offset:], header.DynamicContextsCount)
	offset += 2

	// infer_features (3 bools = 3 bytes)
	if header.PreliminaryRunAsap {
		buf[offset] = 1
	}
	offset++
	if header.BatchRegisterConfig {
		buf[offset] = 1
	}
	offset++
	if header.CanFastBatchSwitch {
		buf[offset] = 1
	}
	offset++

	// validation_features (1 bool = 1 byte)
	if header.IsAbbaleSupported {
		buf[offset] = 1
	}
	offset++

	// networks_count (1 byte)
	buf[offset] = header.NetworksCount
	offset++

	// csm_buffer_size (2 bytes) - little-endian
	binary.LittleEndian.PutUint16(buf[offset:], header.CsmBufferSize)
	offset += 2

	// batch_size array (8 x 2 bytes = 16 bytes) - v4.20.0 format
	for i := 0; i < MaxNetworksPerNetworkGroup; i++ {
		binary.LittleEndian.PutUint16(buf[offset:], header.BatchSize[i])
		offset += 2
	}

	// external_action_list_address (4 bytes) - little-endian
	binary.LittleEndian.PutUint32(buf[offset:], header.ExternalActionListAddress)
	offset += 4

	// boundary_channels_bitmap (3 x 4 bytes = 12 bytes) - little-endian
	for i := 0; i < 3; i++ {
		binary.LittleEndian.PutUint32(buf[offset:], header.BoundaryChannelsBitmap[i])
		offset += 4
	}

	return buf
}

// PackSetNetworkGroupHeaderRequest creates the SET_NETWORK_GROUP_HEADER request
// The format is: request_header + param_count(1) + application_header_length + application_header
func PackSetNetworkGroupHeaderRequest(sequence uint32, appHeader *ApplicationHeader) []byte {
	header := PackRequestHeader(sequence, OpcodeSetNetworkGroupHeader)

	// Pack the application header (32 bytes in little-endian)
	appHeaderBytes := PackApplicationHeader(appHeader)

	// Parameter count = 1 (big-endian, as per protocol)
	paramCount := make([]byte, 4)
	binary.BigEndian.PutUint32(paramCount, 1)

	// application_header_length (4 bytes, big-endian per BYTE_ORDER__htonl in SDK)
	// SDK uses sizeof(application_header) = 32
	appHeaderLen := make([]byte, 4)
	structLen := uint32(len(appHeaderBytes)) // 32 bytes
	binary.BigEndian.PutUint32(appHeaderLen, structLen)

	// Combine: header + param_count + length + raw_struct
	result := append(header, paramCount...)
	result = append(result, appHeaderLen...)
	result = append(result, appHeaderBytes...)

	return result
}

// InvalidExternalActionListAddress indicates action lists are internal (not in DDR)
// Value is 0 per context_switch_defs.h: CONTEXT_SWITCH_DEFS__INVALID_DDR_CONTEXTS_BUFFER_ADDRESS
const InvalidExternalActionListAddress = 0

// MaxContextNetworkDataSize is the maximum data size per SET_CONTEXT_INFO control message
const MaxContextNetworkDataSize = MaxControlLength - RequestHeaderSize - 4 - 17 // header + param_count + fixed params

// PackSetContextInfoRequest creates a SET_CONTEXT_INFO request
// Matches CONTROL_PROTOCOL__context_switch_set_context_info_request_t
func PackSetContextInfoRequest(sequence uint32, chunk *ContextInfoChunk) []byte {
	header := PackRequestHeader(sequence, OpcodeSetContextInfo)

	// Parameter count = 4 (big-endian)
	paramCount := make([]byte, 4)
	binary.BigEndian.PutUint32(paramCount, 4)

	// Parameter 1: is_first_chunk_per_context (1 byte)
	isFirst := byte(0)
	if chunk.IsFirstChunk {
		isFirst = 1
	}
	param1 := packParameter([]byte{isFirst})

	// Parameter 2: is_last_chunk_per_context (1 byte)
	isLast := byte(0)
	if chunk.IsLastChunk {
		isLast = 1
	}
	param2 := packParameter([]byte{isLast})

	// Parameter 3: context_type (1 byte)
	param3 := packParameter([]byte{chunk.ContextType})

	// Parameter 4: context_network_data (variable length)
	param4 := packParameter(chunk.Data)

	// Combine all parts
	result := append(header, paramCount...)
	result = append(result, param1...)
	result = append(result, param2...)
	result = append(result, param3...)
	result = append(result, param4...)

	return result
}

// ValidateResponse checks that a response matches expectations and indicates success.
func ValidateResponse(response []byte, expectedSeq, expectedOpcode uint32) error {
	header, err := ParseResponseHeader(response)
	if err != nil {
		return err
	}

	if header.Sequence != expectedSeq {
		return fmt.Errorf("sequence mismatch: expected %d, got %d", expectedSeq, header.Sequence)
	}

	if header.Opcode != expectedOpcode {
		return fmt.Errorf("opcode mismatch: expected %d, got %d", expectedOpcode, header.Opcode)
	}

	if header.MajorStatus != 0 {
		return fmt.Errorf("firmware error: major_status=%d minor_status=%d",
			header.MajorStatus, header.MinorStatus)
	}

	return nil
}
