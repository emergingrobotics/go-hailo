# Plan: Fix Network Group Activation Gap

## Executive Summary

The Go Hailo driver has all the low-level infrastructure in place (IOCTL wrappers, HEF parsing, VStream management) but is missing the critical **network group activation sequence** that tells the firmware to actually start the neural network accelerator. This document provides a step-by-step plan to implement the missing functionality.

## Root Cause Analysis

### Current Behavior (Go Code)

When `ConfiguredNetworkGroup.Activate()` is called in `pkg/device/network_group.go:117-135`:

```go
func (ng *ConfiguredNetworkGroup) Activate() (*ActivatedNetworkGroup, error) {
    // ... state checks ...

    // TODO: Actually configure the device via IOCTL
    // This would involve:
    // 1. Writing action lists for preliminary config
    // 2. Programming VDMA descriptors
    // 3. Enabling VDMA channels

    ng.state = StateActivated  // Just sets a flag!
    return &ActivatedNetworkGroup{configured: ng}, nil
}
```

**Problem**: The code only changes an internal state flag. No firmware commands are sent. The device hardware remains unconfigured.

### Expected Behavior (Official HailoRT)

The official library performs this sequence in `VdmaConfigCoreOp::activate_impl()` and `ResourcesManager`:

1. **Write action lists** to device memory via `HAILO_WRITE_ACTION_LIST` IOCTL
2. **Send network group header** to firmware via `FwControl` (includes action list DMA address)
3. **Send context information** to firmware via `FwControl`
4. **Enable context switch state machine** via `Control::enable_core_op()`:
   - Packs a `CONTROL_PROTOCOL__change_context_switch_status_request_t` message
   - Sends it via `FwControl` IOCTL with opcode `HAILO_CONTROL_OPCODE_CHANGE_CONTEXT_SWITCH_STATUS`
   - Status = `CONTROL_PROTOCOL__CONTEXT_SWITCH_STATUS_ENABLED`

Only after step 4 does the firmware activate the VDMA channels at the hardware level.

---

## Implementation Plan

### Phase 1: Control Protocol Implementation

#### Step 1.1: Create Control Protocol Package

Create `pkg/control/protocol.go` with:

**File: `pkg/control/protocol.go`**

```go
package control

// Control protocol opcodes (from control_protocol.h)
const (
    OpcodeIdentify                     = 0
    OpcodeSetNetworkGroupHeader        = 32 // HAILO_CONTROL_OPCODE_CONTEXT_SWITCH_SET_NETWORK_GROUP_HEADER
    OpcodeSetContextInfo               = 33 // HAILO_CONTROL_OPCODE_CONTEXT_SWITCH_SET_CONTEXT_INFO
    OpcodeChangeContextSwitchStatus    = 36 // HAILO_CONTROL_OPCODE_CHANGE_CONTEXT_SWITCH_STATUS
    OpcodeClearConfiguredApps          = 71 // HAILO_CONTROL_OPCODE_CONTEXT_SWITCH_CLEAR_CONFIGURED_APPS
)

// Context switch status values
const (
    ContextSwitchStatusReset   = 0
    ContextSwitchStatusEnabled = 1
    ContextSwitchStatusPaused  = 2
)

// CPU IDs for firmware communication
const (
    CpuIdAppCpu  = 0 // Application CPU (CPU_ID_APP_CPU)
    CpuIdCoreCpu = 1 // Core CPU (CPU_ID_CORE_CPU) - used for context switch
)

// Protocol version
const ProtocolVersion = 2
```

**Reference**: `/repos/hailort/common/include/control_protocol.h:86-167` for opcode definitions.

#### Step 1.2: Implement Request Header Packing

The control protocol uses a specific header format:

```go
// RequestHeader matches CONTROL_PROTOCOL__request_header_t
// Note: #pragma pack(1) applies - no padding
type RequestHeader struct {
    Version  uint32 // Protocol version (2)
    Flags    uint32 // ACK flag in bit 0
    Sequence uint32 // Incrementing sequence number
    Opcode   uint32 // Command opcode
}

// PackRequestHeader creates a control protocol request header
func PackRequestHeader(sequence, opcode uint32) []byte {
    buf := make([]byte, 16)
    binary.LittleEndian.PutUint32(buf[0:4], ProtocolVersion)
    binary.LittleEndian.PutUint32(buf[4:8], 0) // flags = 0 (no ACK)
    binary.LittleEndian.PutUint32(buf[8:12], sequence)
    binary.LittleEndian.PutUint32(buf[12:16], opcode)
    return buf
}
```

**Reference**: `/repos/hailort/common/include/control_protocol.h:276-286`

#### Step 1.3: Implement Context Switch Status Request

```go
// ChangeContextSwitchStatusRequest packs the enable/reset command
// Matches CONTROL_PROTOCOL__change_context_switch_status_request_t
func PackChangeContextSwitchStatusRequest(sequence uint32, status, appIndex uint8,
    dynamicBatchSize, batchCount uint16) []byte {

    header := PackRequestHeader(sequence, OpcodeChangeContextSwitchStatus)

    // Parameter count (4 parameters)
    paramCount := make([]byte, 4)
    binary.LittleEndian.PutUint32(paramCount, 4)

    // Each parameter: uint32 length + data
    // Parameter 1: state_machine_status (1 byte)
    param1 := packParameter([]byte{status})

    // Parameter 2: application_index (1 byte)
    param2 := packParameter([]byte{appIndex})

    // Parameter 3: dynamic_batch_size (2 bytes)
    batchSizeBuf := make([]byte, 2)
    binary.LittleEndian.PutUint16(batchSizeBuf, dynamicBatchSize)
    param3 := packParameter(batchSizeBuf)

    // Parameter 4: batch_count (2 bytes)
    batchCountBuf := make([]byte, 2)
    binary.LittleEndian.PutUint16(batchCountBuf, batchCount)
    param4 := packParameter(batchCountBuf)

    // Combine all parts
    result := append(header, paramCount...)
    result = append(result, param1...)
    result = append(result, param2...)
    result = append(result, param3...)
    result = append(result, param4...)

    return result
}

func packParameter(data []byte) []byte {
    lenBuf := make([]byte, 4)
    binary.LittleEndian.PutUint32(lenBuf, uint32(len(data)))
    return append(lenBuf, data...)
}
```

**Reference**: `/repos/hailort/common/include/control_protocol.h:1040-1049`

#### Step 1.4: Implement High-Level Control Functions

```go
// EnableCoreOp enables the context switch state machine for a network group
func EnableCoreOp(device *driver.DeviceFile, sequence uint32, networkGroupIndex uint8,
    dynamicBatchSize, batchCount uint16) error {

    request := PackChangeContextSwitchStatusRequest(
        sequence,
        ContextSwitchStatusEnabled,
        networkGroupIndex,
        dynamicBatchSize,
        batchCount,
    )

    // MD5 is typically zeros for this command
    var md5 [16]byte

    response, _, err := device.FwControl(request, md5, 5000, driver.CpuIdCpu1) // CPU_ID_CORE_CPU
    if err != nil {
        return fmt.Errorf("enable_core_op failed: %w", err)
    }

    // Parse and validate response
    return validateResponse(response, sequence, OpcodeChangeContextSwitchStatus)
}

// ResetContextSwitchStateMachine resets the state machine
func ResetContextSwitchStateMachine(device *driver.DeviceFile, sequence uint32) error {
    const ignoreNetworkGroupIndex = 255
    const ignoreDynamicBatchSize = 0
    const defaultBatchCount = 0

    request := PackChangeContextSwitchStatusRequest(
        sequence,
        ContextSwitchStatusReset,
        ignoreNetworkGroupIndex,
        ignoreDynamicBatchSize,
        defaultBatchCount,
    )

    var md5 [16]byte
    response, _, err := device.FwControl(request, md5, 5000, driver.CpuIdCpu1)
    if err != nil {
        return fmt.Errorf("reset_context_switch_state_machine failed: %w", err)
    }

    return validateResponse(response, sequence, OpcodeChangeContextSwitchStatus)
}
```

**Reference**: `/repos/hailort/libhailort/src/device_common/control.cpp:2744-2758`

---

### Phase 2: Action List Serialization

The HEF already extracts `ConfigAction` structures. These must be serialized to the binary format the firmware expects.

#### Step 2.1: Action List Binary Format

Each action is serialized with a header + parameters:

```go
// ActionHeader is 8 bytes
type ActionHeader struct {
    Type      uint8    // Action type enum
    _         [3]byte  // Padding
    Timestamp uint32   // Typically 0
}

// SerializeAction converts a ConfigAction to binary
func SerializeAction(action *hef.ConfigAction) []byte {
    header := ActionHeader{
        Type:      uint8(mapActionType(action.Type)),
        Timestamp: 0,
    }

    // Header bytes
    buf := make([]byte, 8)
    buf[0] = header.Type
    // bytes 1-3 are padding (zeros)
    binary.LittleEndian.PutUint32(buf[4:8], header.Timestamp)

    // Action-specific parameters
    params := serializeActionParams(action)

    return append(buf, params...)
}
```

**Reference**: `/repos/hailort/hailort/libhailort/src/hef/context_switch_actions.cpp` for serialization logic.

#### Step 2.2: Build Context Info Chunks

Action lists are sent to firmware in chunks:

```go
const MaxContextSize = 4096 // CONTROL_PROTOCOL__MAX_CONTEXT_SIZE

// ContextInfoChunk matches CONTROL_PROTOCOL__context_switch_context_info_chunk_t
type ContextInfoChunk struct {
    IsFirstChunk bool
    IsLastChunk  bool
    ContextType  uint8 // 0=preliminary, 1=dynamic
    Data         []byte
}

// BuildContextInfoChunks splits serialized actions into firmware chunks
func BuildContextInfoChunks(serializedActions []byte, contextType uint8) []ContextInfoChunk {
    // Split into chunks of MaxContextSize
    var chunks []ContextInfoChunk

    for i := 0; i < len(serializedActions); i += MaxContextSize {
        end := i + MaxContextSize
        if end > len(serializedActions) {
            end = len(serializedActions)
        }

        chunk := ContextInfoChunk{
            IsFirstChunk: i == 0,
            IsLastChunk:  end == len(serializedActions),
            ContextType:  contextType,
            Data:         serializedActions[i:end],
        }
        chunks = append(chunks, chunk)
    }

    return chunks
}
```

**Reference**: `/repos/hailort/common/include/control_protocol.h:1508-1514`

---

### Phase 3: Network Group Activation Sequence

#### Step 3.1: Update ConfiguredNetworkGroup

**File: `pkg/device/network_group.go`**

```go
type ConfiguredNetworkGroup struct {
    device             *Device
    hef                *hef.Hef
    info               *hef.NetworkGroupInfo
    state              NetworkGroupState
    inputs             []StreamInfo
    outputs            []StreamInfo
    mu                 sync.RWMutex

    // New fields for activation
    networkGroupIndex  uint8
    actionListDmaAddr  uint64
    controlSequence    uint32
}
```

#### Step 3.2: Implement Full Activate Method

```go
func (ng *ConfiguredNetworkGroup) Activate() (*ActivatedNetworkGroup, error) {
    ng.mu.Lock()
    defer ng.mu.Unlock()

    if ng.state != StateConfigured && ng.state != StateDeactivated {
        return nil, ErrInvalidState
    }

    deviceFile := ng.device.DeviceFile()

    // Step 1: Serialize and write action lists
    actionListData, err := ng.buildActionListBuffer()
    if err != nil {
        return nil, fmt.Errorf("failed to build action list: %w", err)
    }

    if len(actionListData) > 0 {
        dmaAddr, err := deviceFile.WriteActionList(actionListData)
        if err != nil {
            return nil, fmt.Errorf("failed to write action list: %w", err)
        }
        ng.actionListDmaAddr = dmaAddr
    }

    // Step 2: Send network group header to firmware
    // (includes action list DMA address)
    if err := ng.sendNetworkGroupHeader(); err != nil {
        return nil, fmt.Errorf("failed to set network group header: %w", err)
    }

    // Step 3: Send context info to firmware
    if err := ng.sendContextInfo(); err != nil {
        return nil, fmt.Errorf("failed to set context info: %w", err)
    }

    // Step 4: Enable the context switch state machine
    ng.controlSequence++
    err = control.EnableCoreOp(
        deviceFile,
        ng.controlSequence,
        ng.networkGroupIndex,
        1, // dynamic_batch_size = 1
        0, // batch_count = 0 (infinite)
    )
    if err != nil {
        return nil, fmt.Errorf("failed to enable core op: %w", err)
    }

    ng.state = StateActivated
    return &ActivatedNetworkGroup{configured: ng}, nil
}
```

#### Step 3.3: Implement Deactivate

```go
func (ang *ActivatedNetworkGroup) Deactivate() error {
    ang.mu.Lock()
    defer ang.mu.Unlock()

    if ang.deactivated {
        return nil
    }

    deviceFile := ang.configured.device.DeviceFile()

    // Reset the context switch state machine
    ang.configured.controlSequence++
    err := control.ResetContextSwitchStateMachine(
        deviceFile,
        ang.configured.controlSequence,
    )
    if err != nil {
        return fmt.Errorf("failed to reset state machine: %w", err)
    }

    ang.configured.mu.Lock()
    ang.configured.state = StateDeactivated
    ang.configured.mu.Unlock()

    ang.deactivated = true
    return nil
}
```

---

### Phase 4: Response Validation

#### Step 4.1: Parse Response Header

```go
// ResponseHeader matches CONTROL_PROTOCOL__response_header_t
type ResponseHeader struct {
    Version     uint32
    Flags       uint32
    Sequence    uint32
    Opcode      uint32
    MajorStatus uint32
    MinorStatus uint32
}

func validateResponse(response []byte, expectedSeq, expectedOpcode uint32) error {
    if len(response) < 24 {
        return fmt.Errorf("response too short: %d bytes", len(response))
    }

    header := ResponseHeader{
        Version:     binary.LittleEndian.Uint32(response[0:4]),
        Flags:       binary.LittleEndian.Uint32(response[4:8]),
        Sequence:    binary.LittleEndian.Uint32(response[8:12]),
        Opcode:      binary.LittleEndian.Uint32(response[12:16]),
        MajorStatus: binary.LittleEndian.Uint32(response[16:20]),
        MinorStatus: binary.LittleEndian.Uint32(response[20:24]),
    }

    if header.Sequence != expectedSeq {
        return fmt.Errorf("sequence mismatch: expected %d, got %d", expectedSeq, header.Sequence)
    }

    if header.Opcode != expectedOpcode {
        return fmt.Errorf("opcode mismatch: expected %d, got %d", expectedOpcode, header.Opcode)
    }

    if header.MajorStatus != 0 {
        return fmt.Errorf("firmware error: major=%d minor=%d", header.MajorStatus, header.MinorStatus)
    }

    return nil
}
```

**Reference**: `/repos/hailort/common/include/control_protocol.h:293-297`

---

## File Changes Summary

### New Files to Create

1. **`pkg/control/protocol.go`** - Control protocol constants and message packing
2. **`pkg/control/messages.go`** - High-level control functions (EnableCoreOp, etc.)
3. **`pkg/control/response.go`** - Response parsing and validation
4. **`pkg/control/actions.go`** - Action list serialization

### Files to Modify

1. **`pkg/device/network_group.go`**
   - Add new fields to `ConfiguredNetworkGroup` struct
   - Implement full `Activate()` method with firmware commands
   - Implement full `Deactivate()` method with state machine reset

2. **`pkg/device/device.go`**
   - Add `DeviceFile()` accessor method to expose driver.DeviceFile
   - Add control sequence counter

3. **`pkg/hef/types.go`**
   - Add action type to firmware action type mapping

---

## Comparison: Go vs Official HailoRT

| Step | Official HailoRT | Current Go | After Fix |
|------|------------------|------------|-----------|
| 1. Parse HEF | `Hef::parse()` | `hef.ParseFromFile()` | No change |
| 2. Create VStreams | `VStream::create()` | `stream.NewInputVStream()` | No change |
| 3. Configure network | `configure()` | N/A (missing) | `Activate()` step 1-3 |
| 4. Enable state machine | `enable_core_op()` | N/A (missing) | `control.EnableCoreOp()` |
| 5. Launch transfers | `VdmaLaunchTransfer` | `channel.LaunchTransfer()` | No change |
| 6. Wait for completion | `VdmaInterruptsWait` | `channel.WaitForInterrupt()` | No change |
| 7. Deactivate | `reset_context_switch_state_machine()` | N/A (missing) | `control.ResetContextSwitchStateMachine()` |

---

## Testing Plan

### Unit Tests

1. **`pkg/control/protocol_test.go`**
   - Test header packing produces correct bytes
   - Test parameter packing with known values
   - Test response parsing with mock data

2. **`pkg/control/messages_test.go`**
   - Test EnableCoreOp request format matches expected
   - Test ResetContextSwitchStateMachine request format

### Integration Tests

1. **Test with actual device**
   - Verify `EnableCoreOp` doesn't return error
   - Verify `ResetContextSwitchStateMachine` works
   - Run person-detector and verify output

---

## Implementation Order

1. **Phase 1** (Control Protocol) - Required first as foundation
   - Step 1.1: Create package structure
   - Step 1.2: Request header packing
   - Step 1.3: Context switch status request
   - Step 1.4: High-level control functions

2. **Phase 2** (Action Lists) - Can be simplified initially
   - For single-context networks, may be able to skip action list writing
   - Start with minimal implementation, enhance later

3. **Phase 3** (Network Group) - Ties everything together
   - Step 3.1: Add struct fields
   - Step 3.2: Implement Activate
   - Step 3.3: Implement Deactivate

4. **Phase 4** (Validation) - Critical for debugging
   - Response parsing for error detection

---

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Message format incorrect | Device error/hang | Compare byte-by-byte with C++ implementation using debug logging |
| Missing action list fields | Network misconfigured | Start with simple network (yolov8n), add features incrementally |
| Sequence number issues | Response validation fails | Implement proper sequence tracking per device |
| Timeout during FwControl | Activation fails | Use appropriate timeout (5000ms typical) |

---

## Appendix: Key Reference Files

### Official HailoRT (C++)

- `/repos/hailort/common/include/control_protocol.h` - Protocol structures
- `/repos/hailort/libhailort/src/device_common/control.cpp:2707-2758` - enable_core_op, change_context_switch_status
- `/repos/hailort/libhailort/src/device_common/control_protocol.cpp:1808-1849` - pack_change_context_switch_status_request
- `/repos/hailort/libhailort/src/vdma/driver/hailort_driver.cpp:450-481` - fw_control IOCTL wrapper
- `/repos/hailort/libhailort/src/core_op/resource_manager/resource_manager.cpp:672-678` - enable_state_machine
- `/repos/hailort/libhailort/src/vdma/vdma_config_core_op.cpp:69` - activate_impl

### Current Go Implementation

- `/go-hailo/pkg/driver/ioctl.go:261-283` - FwControl exists but unused
- `/go-hailo/pkg/driver/ioctl.go:309-321` - WriteActionList exists but unused
- `/go-hailo/pkg/device/network_group.go:117-135` - Activate with TODO
- `/go-hailo/pkg/hef/types.go:238-276` - ConfigAction types (already parsed)
