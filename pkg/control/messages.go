package control

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/anthropics/purple-hailo/pkg/driver"
)

// computeRequestMD5 calculates the MD5 hash of the request buffer.
// The official HailoRT calculates this for integrity verification.
func computeRequestMD5(request []byte) [16]byte {
	return md5.Sum(request)
}

// SetNetworkGroupHeader sends the network group header to configure a network group.
// This MUST be called before EnableCoreOp.
//
// NOTE: This uses the v4.20.0 firmware format (41 bytes) which differs from v4.21.0+.
// The key differences in v4.20.0:
// - batch_size is an array of MAX_NETWORKS_PER_NETWORK_GROUP (8) uint16 values
// - No config_channels_count or config_channel_info fields
// If using newer firmware (v4.21.0+), the struct format would need to be updated.
func SetNetworkGroupHeader(device *driver.DeviceFile, sequence uint32, appHeader *ApplicationHeader) error {
	request := PackSetNetworkGroupHeaderRequest(sequence, appHeader)

	reqLen := len(request)
	printLen := reqLen
	if printLen > 64 {
		printLen = 64
	}
	log.Printf("[control] SetNetworkGroupHeader: seq=%d, requestLen=%d, dynamicContexts=%d, networksCount=%d, appHeaderSize=%d (v4.20.0)",
		sequence, reqLen, appHeader.DynamicContextsCount, appHeader.NetworksCount, ApplicationHeaderSizeV420)
	log.Printf("[control] SetNetworkGroupHeader: request hex: % 02x", request[:printLen])

	reqMD5 := computeRequestMD5(request)

	response, _, err := device.FwControl(request, reqMD5, DefaultTimeoutMs, CpuIdCoreCpu)
	if err != nil {
		log.Printf("[control] SetNetworkGroupHeader: FwControl failed: %v", err)
		return fmt.Errorf("set_network_group_header FwControl failed: %w", err)
	}

	log.Printf("[control] SetNetworkGroupHeader: response len=%d, hex: % 02x", len(response), response)

	if err := ValidateResponse(response, sequence, OpcodeSetNetworkGroupHeader); err != nil {
		log.Printf("[control] SetNetworkGroupHeader: validation failed: %v", err)
		return fmt.Errorf("set_network_group_header validation failed: %w", err)
	}

	log.Printf("[control] SetNetworkGroupHeader: success")
	return nil
}

// DefaultSGPageSize is the default scatter-gather page size for CSM buffer
const DefaultSGPageSize = 512

// CreateDefaultApplicationHeader creates a minimal application header
// This is used when we don't have full HEF metadata available
// Note: This uses the v4.20.0 firmware format (batch_size is an array, no config_channels)
func CreateDefaultApplicationHeader(dynamicContextsCount uint16) *ApplicationHeader {
	header := &ApplicationHeader{
		DynamicContextsCount:      dynamicContextsCount,
		PreliminaryRunAsap:        false,
		BatchRegisterConfig:       false,
		CanFastBatchSwitch:        false,
		IsAbbaleSupported:         false,
		NetworksCount:             1,
		CsmBufferSize:             DefaultSGPageSize, // Must be non-zero
		ExternalActionListAddress: InvalidExternalActionListAddress,
	}
	// Set batch size to 1 for first network
	header.BatchSize[0] = 1
	return header
}

// EnableCoreOp enables the context switch state machine for a network group.
// This is the critical function that tells the firmware to start processing.
// Maps to Control::enable_core_op() in the official HailoRT.
func EnableCoreOp(device *driver.DeviceFile, sequence uint32, networkGroupIndex uint8,
	dynamicBatchSize, batchCount uint16) error {

	request := PackChangeContextSwitchStatusRequest(
		sequence,
		ContextSwitchStatusEnabled,
		networkGroupIndex,
		dynamicBatchSize,
		batchCount,
	)

	log.Printf("[control] EnableCoreOp: seq=%d, ngIndex=%d, batchSize=%d, batchCount=%d, requestLen=%d",
		sequence, networkGroupIndex, dynamicBatchSize, batchCount, len(request))

	// Calculate MD5 of request for firmware integrity check
	reqMD5 := computeRequestMD5(request)

	response, _, err := device.FwControl(request, reqMD5, DefaultTimeoutMs, CpuIdCoreCpu)
	if err != nil {
		log.Printf("[control] EnableCoreOp: FwControl failed: %v", err)
		return fmt.Errorf("enable_core_op FwControl failed: %w", err)
	}

	log.Printf("[control] EnableCoreOp: response len=%d", len(response))

	// Validate response
	if err := ValidateResponse(response, sequence, OpcodeChangeContextSwitchStatus); err != nil {
		log.Printf("[control] EnableCoreOp: validation failed: %v", err)
		return fmt.Errorf("enable_core_op validation failed: %w", err)
	}

	log.Printf("[control] EnableCoreOp: success")
	return nil
}

// ResetContextSwitchStateMachine resets the context switch state machine.
// This should be called when deactivating a network group.
// Maps to Control::reset_context_switch_state_machine() in the official HailoRT.
func ResetContextSwitchStateMachine(device *driver.DeviceFile, sequence uint32) error {
	request := PackChangeContextSwitchStatusRequest(
		sequence,
		ContextSwitchStatusReset,
		IgnoreNetworkGroupIndex,
		IgnoreDynamicBatchSize,
		DefaultBatchCount,
	)

	reqMD5 := computeRequestMD5(request)

	response, _, err := device.FwControl(request, reqMD5, DefaultTimeoutMs, CpuIdCoreCpu)
	if err != nil {
		return fmt.Errorf("reset_context_switch_state_machine FwControl failed: %w", err)
	}

	if err := ValidateResponse(response, sequence, OpcodeChangeContextSwitchStatus); err != nil {
		return fmt.Errorf("reset_context_switch_state_machine validation failed: %w", err)
	}

	return nil
}

// Reset types from control_protocol.h
const (
	ResetTypeChip       = 0 // Full chip reset
	ResetTypeNNCore     = 1 // NN Core reset
	ResetTypeSoft       = 2 // Soft reset
	ResetTypeForcedSoft = 3 // Forced soft reset
)

// Reset sends a reset command to the device.
// resetType: 0=Chip, 1=NNCore, 2=Soft, 3=ForcedSoft
func Reset(device *driver.DeviceFile, sequence uint32, resetType uint8) error {
	header := PackRequestHeader(sequence, OpcodeReset)

	// Parameter 1: reset_type
	paramCount := make([]byte, 4)
	binary.BigEndian.PutUint32(paramCount, 1) // 1 parameter

	resetTypeLen := make([]byte, 4)
	binary.BigEndian.PutUint32(resetTypeLen, 1) // length = 1 byte
	resetTypeValue := []byte{resetType}

	request := append(header, paramCount...)
	request = append(request, resetTypeLen...)
	request = append(request, resetTypeValue...)

	reqMD5 := computeRequestMD5(request)

	log.Printf("[control] Reset: seq=%d, resetType=%d", sequence, resetType)

	// Reset command goes to APP CPU
	// Note: The firmware may not respond after reset, so we accept timeout
	response, _, err := device.FwControl(request, reqMD5, DefaultTimeoutMs, CpuIdAppCpu)
	if err != nil {
		// Timeout is expected for reset command
		log.Printf("[control] Reset: FwControl returned: %v (may be expected for reset)", err)
		return nil // Don't return error for expected timeout
	}

	if len(response) >= ResponseHeaderSize {
		// Don't validate opcode for reset - response might have different opcode
		header, _ := ParseResponseHeader(response)
		if header != nil && header.MajorStatus != 0 {
			return fmt.Errorf("reset failed: status=%d", header.MajorStatus)
		}
	}

	log.Printf("[control] Reset: success")
	return nil
}

// SoftReset sends a soft reset command to the device.
func SoftReset(device *driver.DeviceFile, sequence uint32) error {
	return Reset(device, sequence, ResetTypeSoft)
}

// ResetNNCore sends a NN Core reset command to the device.
// This specifically resets the neural network processing core.
func ResetNNCore(device *driver.DeviceFile, sequence uint32) error {
	return Reset(device, sequence, ResetTypeNNCore)
}

// ClearConfiguredApps clears all configured applications from the device.
// This can be called before configuring a new network group.
func ClearConfiguredApps(device *driver.DeviceFile, sequence uint32) error {
	header := PackRequestHeader(sequence, OpcodeClearConfiguredApps)

	// No parameters for this command
	paramCount := make([]byte, 4)
	// paramCount is 0

	request := append(header, paramCount...)

	reqMD5 := computeRequestMD5(request)

	response, _, err := device.FwControl(request, reqMD5, DefaultTimeoutMs, CpuIdCoreCpu)
	if err != nil {
		return fmt.Errorf("clear_configured_apps FwControl failed: %w", err)
	}

	if err := ValidateResponse(response, sequence, OpcodeClearConfiguredApps); err != nil {
		return fmt.Errorf("clear_configured_apps validation failed: %w", err)
	}

	return nil
}

// Identify sends the basic identify command to APP CPU to get firmware version.
// This is the simplest firmware command and should always work if the device is functioning.
func Identify(device *driver.DeviceFile, sequence uint32) error {
	header := PackRequestHeader(sequence, OpcodeIdentify)

	// No parameters
	paramCount := make([]byte, 4)

	request := append(header, paramCount...)

	log.Printf("[control] Identify: seq=%d, opcode=%d, requestLen=%d (APP CPU)", sequence, OpcodeIdentify, len(request))

	reqMD5 := computeRequestMD5(request)

	// Use APP CPU (CPU0) for basic identify
	response, _, err := device.FwControl(request, reqMD5, DefaultTimeoutMs, CpuIdAppCpu)
	if err != nil {
		log.Printf("[control] Identify: FwControl failed: %v", err)
		return fmt.Errorf("identify FwControl failed: %w", err)
	}

	log.Printf("[control] Identify: response len=%d", len(response))

	if err := ValidateResponse(response, sequence, OpcodeIdentify); err != nil {
		log.Printf("[control] Identify: validation failed: %v", err)
		return fmt.Errorf("identify validation failed: %w", err)
	}

	log.Printf("[control] Identify: success")
	return nil
}

// SetContextInfo sends a context info chunk to configure a context.
// This is called after SetNetworkGroupHeader and before EnableCoreOp.
// Multiple chunks may be needed for large contexts.
func SetContextInfo(device *driver.DeviceFile, sequence uint32, chunk *ContextInfoChunk) error {
	request := PackSetContextInfoRequest(sequence, chunk)

	log.Printf("[control] SetContextInfo: seq=%d, type=%d, isFirst=%v, isLast=%v, dataLen=%d",
		sequence, chunk.ContextType, chunk.IsFirstChunk, chunk.IsLastChunk, len(chunk.Data))

	reqMD5 := computeRequestMD5(request)

	response, _, err := device.FwControl(request, reqMD5, DefaultTimeoutMs, CpuIdCoreCpu)
	if err != nil {
		log.Printf("[control] SetContextInfo: FwControl failed: %v", err)
		return fmt.Errorf("set_context_info FwControl failed: %w", err)
	}

	log.Printf("[control] SetContextInfo: response len=%d", len(response))

	if err := ValidateResponse(response, sequence, OpcodeSetContextInfo); err != nil {
		log.Printf("[control] SetContextInfo: validation failed: %v", err)
		return fmt.Errorf("set_context_info validation failed: %w", err)
	}

	log.Printf("[control] SetContextInfo: success")
	return nil
}

// SendContextInfoChunks sends multiple context info chunks for a context type.
// This handles splitting large contexts into multiple control messages.
func SendContextInfoChunks(device *driver.DeviceFile, startSequence *uint32, contextType uint8, data []byte) error {
	if len(data) == 0 {
		// Send empty context (required for some context types)
		*startSequence++
		chunk := &ContextInfoChunk{
			IsFirstChunk: true,
			IsLastChunk:  true,
			ContextType:  contextType,
			Data:         []byte{},
		}
		return SetContextInfo(device, *startSequence, chunk)
	}

	// Split into chunks if needed
	offset := 0
	for offset < len(data) {
		*startSequence++
		remaining := len(data) - offset
		chunkSize := remaining
		if chunkSize > MaxContextNetworkDataSize {
			chunkSize = MaxContextNetworkDataSize
		}

		chunk := &ContextInfoChunk{
			IsFirstChunk: offset == 0,
			IsLastChunk:  offset+chunkSize >= len(data),
			ContextType:  contextType,
			Data:         data[offset : offset+chunkSize],
		}

		if err := SetContextInfo(device, *startSequence, chunk); err != nil {
			return err
		}

		offset += chunkSize
	}

	return nil
}

// IdentifyCore sends the core identify command to get firmware version.
// This is useful for verifying firmware communication works.
func IdentifyCore(device *driver.DeviceFile, sequence uint32) error {
	header := PackRequestHeader(sequence, OpcodeCoreIdentify)

	// No parameters
	paramCount := make([]byte, 4)

	request := append(header, paramCount...)

	log.Printf("[control] IdentifyCore: seq=%d, opcode=%d, requestLen=%d (CORE CPU)", sequence, OpcodeCoreIdentify, len(request))

	reqMD5 := computeRequestMD5(request)

	response, _, err := device.FwControl(request, reqMD5, DefaultTimeoutMs, CpuIdCoreCpu)
	if err != nil {
		log.Printf("[control] IdentifyCore: FwControl failed: %v", err)
		return fmt.Errorf("identify_core FwControl failed: %w", err)
	}

	log.Printf("[control] IdentifyCore: response len=%d", len(response))

	if err := ValidateResponse(response, sequence, OpcodeCoreIdentify); err != nil {
		log.Printf("[control] IdentifyCore: validation failed: %v", err)
		return fmt.Errorf("identify_core validation failed: %w", err)
	}

	log.Printf("[control] IdentifyCore: success")
	return nil
}
