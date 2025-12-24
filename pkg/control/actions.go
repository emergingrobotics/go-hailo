// Package control implements action list serialization for Hailo firmware.
// This file handles conversion of HEF protobuf actions to the binary format
// expected by the firmware's context switch state machine.
package control

import (
	"encoding/binary"
	"fmt"

	"github.com/anthropics/purple-hailo/pkg/hef"
)

// Firmware action types from context_switch_defs.h
// CONTEXT_SWITCH_DEFS__ACTION_TYPE_t
const (
	FwActionTypeFetchCfgChannelDescriptors    uint8 = 0
	FwActionTypeTriggerSequencer              uint8 = 1
	FwActionTypeFetchDataFromVdmaChannel      uint8 = 2
	FwActionTypeEnableLcuDefault              uint8 = 3
	FwActionTypeEnableLcuNonDefault           uint8 = 4
	FwActionTypeDisableLcu                    uint8 = 5
	FwActionTypeActivateBoundaryInput         uint8 = 6
	FwActionTypeActivateBoundaryOutput        uint8 = 7
	FwActionTypeActivateInterContextInput     uint8 = 8
	FwActionTypeActivateInterContextOutput    uint8 = 9
	FwActionTypeActivateDdrBufferInput        uint8 = 10
	FwActionTypeActivateDdrBufferOutput       uint8 = 11
	FwActionTypeDeactivateVdmaChannel         uint8 = 12
	FwActionTypeChangeVdmaToStreamMapping     uint8 = 13
	FwActionTypeAddDdrPairInfo                uint8 = 14
	FwActionTypeDdrBufferingStart             uint8 = 15
	FwActionTypeLcuInterrupt                  uint8 = 16
	FwActionTypeSequencerDoneInterrupt        uint8 = 17
	FwActionTypeInputChannelTransferDone      uint8 = 18
	FwActionTypeOutputChannelTransferDone     uint8 = 19
	FwActionTypeModuleConfigDoneInterrupt     uint8 = 20
	FwActionTypeApplicationChangeInterrupt    uint8 = 21
	FwActionTypeActivateCfgChannel            uint8 = 22
	FwActionTypeDeactivateCfgChannel          uint8 = 23
	FwActionTypeRepeatedAction                uint8 = 24
	FwActionTypeWaitForDmaIdle                uint8 = 25
	FwActionTypeWaitForNms                    uint8 = 26
	FwActionTypeFetchCcwBursts                uint8 = 27
	FwActionTypeValidateVdmaChannel           uint8 = 28
	FwActionTypeBurstCreditsTaskStart         uint8 = 29
	FwActionTypeBurstCreditsTaskReset         uint8 = 30
	FwActionTypeDdrBufferingReset             uint8 = 31
	FwActionTypeOpenBoundaryInputChannel      uint8 = 32
	FwActionTypeOpenBoundaryOutputChannel     uint8 = 33
	FwActionTypeEnableNms                     uint8 = 34
	FwActionTypeWriteDataByType               uint8 = 35
	FwActionTypeSwitchLcuBatch                uint8 = 36
	FwActionTypeChangeBoundaryInputBatch      uint8 = 37
	FwActionTypePauseVdmaChannel              uint8 = 38
	FwActionTypeResumeVdmaChannel             uint8 = 39
	FwActionTypeActivateCacheInput            uint8 = 40
	FwActionTypeActivateCacheOutput           uint8 = 41
	FwActionTypeWaitForCacheUpdated           uint8 = 42
	FwActionTypeSleep                         uint8 = 43
	FwActionTypeHalt                          uint8 = 44
)

// TimestampInitValue is the initial timestamp value used by firmware
// Actions start with this value and count down
const TimestampInitValue uint32 = 0xFFFFFFFF

// Default LCU constants
const (
	EnableLcuDefaultKernelAddress uint16 = 1
	EnableLcuDefaultKernelCount   uint32 = 2
)

// PackedLcuId packs cluster_index and lcu_index into a single byte
// Format: bits 0-3 = lcu_index, bits 4-6 = cluster_index
func PackedLcuId(clusterIndex, lcuIndex uint32) uint8 {
	return uint8((lcuIndex & 0x0F) | ((clusterIndex & 0x07) << 4))
}

// PackedVdmaChannelId packs engine_index and vdma_channel_index into a single byte
// Format: bits 0-4 = vdma_channel_index, bits 5-6 = engine_index
func PackedVdmaChannelId(engineIndex, vdmaChannelIndex uint32) uint8 {
	return uint8((vdmaChannelIndex & 0x1F) | ((engineIndex & 0x03) << 5))
}

// ActionHeader represents the common action header (5 bytes)
// Matches CONTEXT_SWITCH_DEFS__common_action_header_t
type ActionHeader struct {
	ActionType uint8
	Timestamp  uint32
}

// PackActionHeader packs the action header into bytes
func PackActionHeader(actionType uint8, timestamp uint32) []byte {
	buf := make([]byte, 5)
	buf[0] = actionType
	binary.LittleEndian.PutUint32(buf[1:5], timestamp)
	return buf
}

// SerializeEnableLcuDefault serializes an enable LCU default action
// Matches CONTEXT_SWITCH_DEFS__enable_lcu_action_default_data_t
func SerializeEnableLcuDefault(clusterIndex, lcuIndex, networkIndex uint32, timestamp uint32) []byte {
	header := PackActionHeader(FwActionTypeEnableLcuDefault, timestamp)
	// Data: packed_lcu_id (1 byte) + network_index (1 byte) = 2 bytes
	data := []byte{
		PackedLcuId(clusterIndex, lcuIndex),
		uint8(networkIndex),
	}
	return append(header, data...)
}

// SerializeEnableLcuNonDefault serializes an enable LCU non-default action
// Matches CONTEXT_SWITCH_DEFS__enable_lcu_action_non_default_data_t
func SerializeEnableLcuNonDefault(clusterIndex, lcuIndex, networkIndex uint32,
	kernelDoneAddress uint16, kernelDoneCount uint32, timestamp uint32) []byte {
	header := PackActionHeader(FwActionTypeEnableLcuNonDefault, timestamp)
	// Data: packed_lcu_id (1) + network_index (1) + kernel_done_address (2) + kernel_done_count (4) = 8 bytes
	data := make([]byte, 8)
	data[0] = PackedLcuId(clusterIndex, lcuIndex)
	data[1] = uint8(networkIndex)
	binary.LittleEndian.PutUint16(data[2:4], kernelDoneAddress)
	binary.LittleEndian.PutUint32(data[4:8], kernelDoneCount)
	return append(header, data...)
}

// SerializeDisableLcu serializes a disable LCU action
// Matches CONTEXT_SWITCH_DEFS__disable_lcu_action_data_t
func SerializeDisableLcu(clusterIndex, lcuIndex uint32, timestamp uint32) []byte {
	header := PackActionHeader(FwActionTypeDisableLcu, timestamp)
	// Data: packed_lcu_id (1 byte)
	data := []byte{PackedLcuId(clusterIndex, lcuIndex)}
	return append(header, data...)
}

// SequencerConfig represents the sequencer configuration
// Matches CONTEXT_SWITCH_DEFS__sequencer_config_t
type SequencerConfig struct {
	InitialL3Cut    uint8
	InitialL3Offset uint16
	ActiveApu       uint32
	ActiveIa        uint32
	ActiveSc        uint64
	ActiveL2        uint64
	L2Offset0       uint64
	L2Offset1       uint64
}

// SerializeTriggerSequencer serializes a trigger sequencer action
// Matches CONTEXT_SWITCH_DEFS__trigger_sequencer_action_data_t
func SerializeTriggerSequencer(clusterIndex uint8, config *SequencerConfig, timestamp uint32) []byte {
	header := PackActionHeader(FwActionTypeTriggerSequencer, timestamp)
	// Data: cluster_index (1) + sequencer_config (43 bytes) = 44 bytes
	data := make([]byte, 44)
	data[0] = clusterIndex
	// Pack sequencer_config_t
	data[1] = config.InitialL3Cut
	binary.LittleEndian.PutUint16(data[2:4], config.InitialL3Offset)
	binary.LittleEndian.PutUint32(data[4:8], config.ActiveApu)
	binary.LittleEndian.PutUint32(data[8:12], config.ActiveIa)
	binary.LittleEndian.PutUint64(data[12:20], config.ActiveSc)
	binary.LittleEndian.PutUint64(data[20:28], config.ActiveL2)
	binary.LittleEndian.PutUint64(data[28:36], config.L2Offset0)
	binary.LittleEndian.PutUint64(data[36:44], config.L2Offset1)
	return append(header, data...)
}

// SerializeFetchCcwBursts serializes a fetch CCW bursts action
// Matches CONTEXT_SWITCH_DEFS__fetch_ccw_bursts_action_data_t
func SerializeFetchCcwBursts(ccwBursts uint16, configStreamIndex uint8, timestamp uint32) []byte {
	header := PackActionHeader(FwActionTypeFetchCcwBursts, timestamp)
	// Data: ccw_bursts (2) + config_stream_index (1) = 3 bytes
	data := make([]byte, 3)
	binary.LittleEndian.PutUint16(data[0:2], ccwBursts)
	data[2] = configStreamIndex
	return append(header, data...)
}

// SerializeEnableNms serializes an enable NMS action
// Matches CONTEXT_SWITCH_DEFS__enable_nms_action_t
func SerializeEnableNms(nmsUnitIndex, networkIndex uint8, numberOfClasses, burstSize uint16,
	divisionFactor uint8, timestamp uint32) []byte {
	header := PackActionHeader(FwActionTypeEnableNms, timestamp)
	// Data: nms_unit_index (1) + network_index (1) + number_of_classes (2) + burst_size (2) + division_factor (1) = 7 bytes
	data := make([]byte, 7)
	data[0] = nmsUnitIndex
	data[1] = networkIndex
	binary.LittleEndian.PutUint16(data[2:4], numberOfClasses)
	binary.LittleEndian.PutUint16(data[4:6], burstSize)
	data[6] = divisionFactor
	return append(header, data...)
}

// SerializeWriteDataByType serializes a write data by type action
// Matches CONTEXT_SWITCH_DEFS__write_data_by_type_action_t
func SerializeWriteDataByType(address uint32, dataType uint8, dataValue uint32,
	shift uint8, mask uint32, networkIndex uint8, timestamp uint32) []byte {
	header := PackActionHeader(FwActionTypeWriteDataByType, timestamp)
	// Data: address (4) + data_type (1) + data (4) + shift (1) + mask (4) + network_index (1) = 15 bytes
	data := make([]byte, 15)
	binary.LittleEndian.PutUint32(data[0:4], address)
	data[4] = dataType
	binary.LittleEndian.PutUint32(data[5:9], dataValue)
	data[9] = shift
	binary.LittleEndian.PutUint32(data[10:14], mask)
	data[14] = networkIndex
	return append(header, data...)
}

// SerializeSwitchLcuBatch serializes a switch LCU batch action
// Matches CONTEXT_SWITCH_DEFS__switch_lcu_batch_action_data_t
func SerializeSwitchLcuBatch(clusterIndex, lcuIndex, networkIndex uint32,
	kernelDoneCount uint32, timestamp uint32) []byte {
	header := PackActionHeader(FwActionTypeSwitchLcuBatch, timestamp)
	// Data: packed_lcu_id (1) + network_index (1) + kernel_done_count (4) = 6 bytes
	data := make([]byte, 6)
	data[0] = PackedLcuId(clusterIndex, lcuIndex)
	data[1] = uint8(networkIndex)
	binary.LittleEndian.PutUint32(data[2:6], kernelDoneCount)
	return append(header, data...)
}

// SerializeSleep serializes a sleep action
// Matches CONTEXT_SWITCH_DEFS__sleep_action_data_t
func SerializeSleep(sleepTimeUs uint32, timestamp uint32) []byte {
	header := PackActionHeader(FwActionTypeSleep, timestamp)
	// Data: sleep_time (4 bytes)
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data[0:4], sleepTimeUs)
	return append(header, data...)
}

// SerializeHalt serializes a halt action
func SerializeHalt(timestamp uint32) []byte {
	header := PackActionHeader(FwActionTypeHalt, timestamp)
	// Data: none
	return header
}

// SerializeModuleConfigDoneInterrupt serializes a module config done interrupt action
// Matches CONTEXT_SWITCH_DEFS__module_config_done_interrupt_data_t
func SerializeModuleConfigDoneInterrupt(moduleIndex uint8, timestamp uint32) []byte {
	header := PackActionHeader(FwActionTypeModuleConfigDoneInterrupt, timestamp)
	// Data: module_index (1 byte)
	return append(header, moduleIndex)
}

// SerializeSequencerDoneInterrupt serializes a sequencer done interrupt action
// Matches CONTEXT_SWITCH_DEFS__sequencer_interrupt_data_t
func SerializeSequencerDoneInterrupt(sequencerIndex uint8, timestamp uint32) []byte {
	header := PackActionHeader(FwActionTypeSequencerDoneInterrupt, timestamp)
	// Data: sequencer_index (1 byte)
	return append(header, sequencerIndex)
}

// ActionListBuilder helps build action lists for contexts
type ActionListBuilder struct {
	actions   []byte
	timestamp uint32
}

// NewActionListBuilder creates a new action list builder
func NewActionListBuilder() *ActionListBuilder {
	return &ActionListBuilder{
		actions:   nil,
		timestamp: TimestampInitValue,
	}
}

// AddAction adds a serialized action to the list
func (b *ActionListBuilder) AddAction(action []byte) {
	b.actions = append(b.actions, action...)
}

// Build returns the complete action list
func (b *ActionListBuilder) Build() []byte {
	return b.actions
}

// Len returns the current length of the action list
func (b *ActionListBuilder) Len() int {
	return len(b.actions)
}

// ConvertHefActionToFirmware converts a HEF ConfigAction to firmware binary format
// Returns the serialized action bytes
func ConvertHefActionToFirmware(action *hef.ConfigAction, timestamp uint32) ([]byte, error) {
	switch action.Type {
	case hef.ActionTypeEnableLcu:
		if action.EnableLcu == nil {
			return nil, fmt.Errorf("EnableLcu action missing parameters")
		}
		p := action.EnableLcu
		// Decide between default and non-default based on kernel_done values
		if p.KernelDoneAddress == uint32(EnableLcuDefaultKernelAddress) &&
			p.KernelDoneCount == EnableLcuDefaultKernelCount {
			return SerializeEnableLcuDefault(p.ClusterIndex, p.LcuIndex, p.NetworkIndex, timestamp), nil
		}
		return SerializeEnableLcuNonDefault(p.ClusterIndex, p.LcuIndex, p.NetworkIndex,
			uint16(p.KernelDoneAddress), p.KernelDoneCount, timestamp), nil

	case hef.ActionTypeDisableLcu:
		if action.DisableLcu == nil {
			return nil, fmt.Errorf("DisableLcu action missing parameters")
		}
		p := action.DisableLcu
		return SerializeDisableLcu(p.ClusterIndex, p.LcuIndex, timestamp), nil

	case hef.ActionTypeEnableSequencer:
		if action.EnableSequencer == nil {
			return nil, fmt.Errorf("EnableSequencer action missing parameters")
		}
		p := action.EnableSequencer
		config := &SequencerConfig{
			InitialL3Cut:    uint8(p.InitialL3Index),
			InitialL3Offset: uint16(p.InitialL3Offset),
			ActiveApu:       p.ActiveApuBitmap,
			ActiveIa:        p.ActiveIaBitmap,
			ActiveSc:        p.ActiveScBitmap,
			ActiveL2:        p.ActiveL2Bitmap,
			// L2 offsets are packed from l2_write values
			L2Offset0: uint64(p.L2Write0) | (uint64(p.L2Write1) << 32),
			L2Offset1: uint64(p.L2Write2) | (uint64(p.L2Write3) << 32),
		}
		return SerializeTriggerSequencer(uint8(p.ClusterIndex), config, timestamp), nil

	case hef.ActionTypeWriteDataCcw:
		// WriteDataCcw actions are NOT written to the firmware action list!
		// They use a separate config buffer DMA mechanism.
		// The SDK says: "WriteDataCcwActions aren't written to the FW's action list."
		// Skip these actions - they need to be handled via config buffers.
		return nil, nil

	case hef.ActionTypeEnableNms:
		if action.EnableNms == nil {
			return nil, fmt.Errorf("EnableNms action missing parameters")
		}
		p := action.EnableNms
		return SerializeEnableNms(uint8(p.NmsUnitIndex), uint8(p.NetworkIndex),
			uint16(p.NumberOfClasses), uint16(p.BurstSize), uint8(p.DivisionFactor), timestamp), nil

	case hef.ActionTypeWriteDataByType:
		if action.WriteDataByType != nil {
			p := action.WriteDataByType
			dataValue := uint32(0)
			if len(p.Data) >= 4 {
				dataValue = binary.LittleEndian.Uint32(p.Data[0:4])
			}
			return SerializeWriteDataByType(uint32(p.Address), uint8(p.DataType), dataValue,
				uint8(p.Shift), p.Mask, uint8(p.NetworkIndex), timestamp), nil
		}
		// Fallback if no structured params
		if len(action.Data) < 4 {
			return nil, fmt.Errorf("WriteDataByType action data too short")
		}
		address := uint32(action.Address)
		dataValue := binary.LittleEndian.Uint32(action.Data[0:4])
		return SerializeWriteDataByType(address, 0, dataValue, 0, 0xFFFFFFFF, 0, timestamp), nil

	case hef.ActionTypeSwitchLcuBatch:
		if action.SwitchLcuBatch == nil {
			return nil, fmt.Errorf("SwitchLcuBatch action missing parameters")
		}
		p := action.SwitchLcuBatch
		// kernel_done_count defaults to 2 (EnableLcuDefaultKernelCount)
		return SerializeSwitchLcuBatch(p.ClusterIndex, p.LcuIndex, p.NetworkIndex,
			EnableLcuDefaultKernelCount, timestamp), nil

	case hef.ActionTypeNone:
		// No action needed
		return nil, nil

	case hef.ActionTypeWaitForSequencer:
		// This is an interrupt/wait action
		clusterIndex := uint8(action.Address)
		return SerializeSequencerDoneInterrupt(clusterIndex, timestamp), nil

	case hef.ActionTypeWaitForModuleConfigDone:
		moduleIndex := uint8(action.Address)
		return SerializeModuleConfigDoneInterrupt(moduleIndex, timestamp), nil

	case hef.ActionTypeWriteData:
		// Direct memory write - not typically used in action lists
		return nil, fmt.Errorf("WriteData action not supported in context switch")

	case hef.ActionTypeAllowInputDataflow:
		// AllowInputDataflow actions configure VDMA boundary channels for data flow.
		// They require runtime DMA channel mapping that's not available from just the HEF.
		// The SDK configures these dynamically during runtime stream setup.
		// Skip these actions - they need to be handled via the DMA subsystem.
		return nil, nil

	default:
		return nil, fmt.Errorf("unsupported action type: %d", action.Type)
	}
}

// BuildContextActionList builds the action list for a context from HEF operations
// This is a simplified version that handles basic action types
func BuildContextActionList(ops []hef.ConfigOperation) ([]byte, error) {
	builder := NewActionListBuilder()
	timestamp := TimestampInitValue

	for _, op := range ops {
		for _, action := range op.Actions {
			actionBytes, err := ConvertHefActionToFirmware(&action, timestamp)
			if err != nil {
				// Log but don't fail for unsupported actions
				continue
			}
			if actionBytes != nil {
				builder.AddAction(actionBytes)
				timestamp-- // Decrement timestamp for next action
			}
		}
	}

	return builder.Build(), nil
}

// BuildEmptyActionList creates a minimal action list for testing
// Contains just a halt action
func BuildEmptyActionList() []byte {
	return SerializeHalt(TimestampInitValue)
}
