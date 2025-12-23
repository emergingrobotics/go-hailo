# Fix Plan: IOCTL Structure Mismatches

## Executive Summary

The Go implementation has critical mismatches in IOCTL structure layouts compared to the official
Hailo driver (hailort-drivers v4.23.0). These mismatches cause the kernel driver to receive
corrupted data, leading to failures or undefined behavior.

## Root Cause Analysis

All structures in the Hailo driver use `#pragma pack(1)` (no padding). The Go implementation
attempted to create packed structures but made several mistakes:

1. **Missing fields** in `PackedDescListProgramParams`
2. **Wrong field layout** in `PackedVdmaTransferBuffer`
3. **Wrong buffer count** in `PackedVdmaLaunchTransferParams`
4. **Wrong irq_data size** in `PackedVdmaInterruptsWaitParams`
5. **Wrong IOCTL direction** for several commands

---

## Issues and Fixes

### Issue 1: PackedDescListProgramParams Missing Fields

**Location**: `pkg/driver/packed.go:161-191`

**Current (WRONG - 43 bytes)**:
```go
// Missing batch_size (4 bytes) and stride (4 bytes)
type PackedDescListProgramParams [43]byte
```

**Expected (CORRECT - 51 bytes)**:
```
Offset  Size  Field
0       8     buffer_handle
8       8     buffer_size
16      8     buffer_offset
24      4     batch_size        <-- MISSING
28      8     desc_handle
36      1     channel_index
37      4     starting_desc
41      1     should_bind
42      4     last_interrupts_domain
46      1     is_debug
47      4     stride            <-- MISSING
Total: 51 bytes
```

**Fix**:
```go
type PackedDescListProgramParams [51]byte

func NewPackedDescListProgramParams(bufferHandle, bufferSize, bufferOffset uint64,
    batchSize uint32, descHandle uintptr, channelIndex uint8, startingDesc uint32,
    shouldBind bool, lastInterruptsDomain InterruptsDomain, isDebug bool,
    stride uint32) *PackedDescListProgramParams {
    var p PackedDescListProgramParams
    binary.LittleEndian.PutUint64(p[0:8], bufferHandle)
    binary.LittleEndian.PutUint64(p[8:16], bufferSize)
    binary.LittleEndian.PutUint64(p[16:24], bufferOffset)
    binary.LittleEndian.PutUint32(p[24:28], batchSize)         // NEW
    binary.LittleEndian.PutUint64(p[28:36], uint64(descHandle))
    p[36] = channelIndex
    binary.LittleEndian.PutUint32(p[37:41], startingDesc)
    if shouldBind { p[41] = 1 }
    binary.LittleEndian.PutUint32(p[42:46], uint32(lastInterruptsDomain))
    if isDebug { p[46] = 1 }
    binary.LittleEndian.PutUint32(p[47:51], stride)            // NEW
    return &p
}
```

---

### Issue 2: PackedVdmaTransferBuffer Wrong Layout

**Location**: `pkg/driver/packed.go:196-210`

**Current (WRONG)**:
```go
// struct hailo_vdma_transfer_buffer {
//     size_t mapped_buffer_handle;  // 8 bytes  <-- WRONG!
//     uint32_t offset;              // 4 bytes  <-- WRONG!
//     uint32_t size;                // 4 bytes
// };
```

**Expected (from hailo_ioctl_common.h)**:
```c
struct hailo_vdma_transfer_buffer {
    enum hailo_dma_buffer_type buffer_type; // 4 bytes, offset 0
    uintptr_t addr_or_fd;                   // 8 bytes, offset 4
    uint32_t size;                          // 4 bytes, offset 12
};
// Total: 16 bytes
```

**Fix**:
```go
type PackedVdmaTransferBuffer [16]byte

func NewPackedVdmaTransferBuffer(bufferType DmaBufferType, addrOrFd uintptr, size uint32) *PackedVdmaTransferBuffer {
    var p PackedVdmaTransferBuffer
    binary.LittleEndian.PutUint32(p[0:4], uint32(bufferType))
    binary.LittleEndian.PutUint64(p[4:12], uint64(addrOrFd))
    binary.LittleEndian.PutUint32(p[12:16], size)
    return &p
}
```

---

### Issue 3: PackedVdmaLaunchTransferParams Wrong Size and Output Fields

**Location**: `pkg/driver/packed.go:212-259`

**Current (WRONG - 65 bytes with 2 buffers and output fields)**:
```go
const MaxBuffersPerSingleTransfer = 2  // WRONG! Should be 8
type PackedVdmaLaunchTransferParams [65]byte
// Includes descs_programed and launch_transfer_status as output - WRONG!
```

**Expected (153 bytes with 8 buffers, NO output fields)**:
```
Offset  Size  Field
0       1     engine_index
1       1     channel_index
2       8     desc_handle
10      4     starting_desc
14      1     should_bind
15      1     buffers_count
16      128   buffers[8]        (8 * 16 bytes each)
144     4     first_interrupts_domain
148     4     last_interrupts_domain
152     1     is_debug
Total: 153 bytes (NO OUTPUT FIELDS)
```

**Fix**:
```go
const MaxBuffersPerSingleTransfer = 8  // HAILO_MAX_BUFFERS_PER_SINGLE_TRANSFER

type PackedVdmaLaunchTransferParams [153]byte

func NewPackedVdmaLaunchTransferParams(engineIndex, channelIndex uint8, descHandle uintptr,
    startingDesc uint32, shouldBind bool, buffers []PackedVdmaTransferBuffer,
    firstDomain, lastDomain InterruptsDomain, isDebug bool) *PackedVdmaLaunchTransferParams {
    var p PackedVdmaLaunchTransferParams
    p[0] = engineIndex
    p[1] = channelIndex
    binary.LittleEndian.PutUint64(p[2:10], uint64(descHandle))
    binary.LittleEndian.PutUint32(p[10:14], startingDesc)
    if shouldBind { p[14] = 1 }
    p[15] = uint8(len(buffers))
    for i, buf := range buffers {
        if i >= MaxBuffersPerSingleTransfer { break }
        copy(p[16+i*16:16+(i+1)*16], buf[:])
    }
    binary.LittleEndian.PutUint32(p[144:148], uint32(firstDomain))
    binary.LittleEndian.PutUint32(p[148:152], uint32(lastDomain))
    if isDebug { p[152] = 1 }
    return &p
}

// Remove DescsProgramed() and LaunchTransferStatus() - they don't exist!
```

---

### Issue 4: PackedVdmaInterruptsWaitParams Wrong irq_data Size

**Location**: `pkg/driver/packed.go:130-159`

**Current (WRONG - 685 bytes with 7 bytes per irq_data)**:
```go
type PackedVdmaInterruptsWaitParams [685]byte
```

**Expected (301 bytes with 3 bytes per irq_data)**:
```
Offset  Size  Field
0       12    channels_bitmap_per_engine[3]  (in)
12      1     channels_count                 (out)
13      288   irq_data[96]                   (96 * 3 bytes) (out)
Total: 301 bytes
```

**Fix**:
```go
// Each irq_data entry is 3 bytes: engine_index(1) + channel_index(1) + data(1)
const PackedVdmaInterruptsWaitParamsSize = 12 + 1 + (MaxVdmaChannelsPerEngine * MaxVdmaEngines * 3)
// = 12 + 1 + (32 * 3 * 3) = 301 bytes

type PackedVdmaInterruptsWaitParams [301]byte

func (p *PackedVdmaInterruptsWaitParams) IrqData(idx int) (engineIndex, channelIndex, data uint8) {
    offset := 13 + idx*3  // 3 bytes per entry, not 7!
    return p[offset], p[offset+1], p[offset+2]
}
```

---

### Issue 5: Wrong IOCTL Direction Codes

**Location**: `pkg/driver/ioctl.go:139-145`

**Current (WRONG)**:
```go
ioctlDescListProgram     = IoR(...)   // Correct
ioctlVdmaLaunchTransfer  = IoWR(...)  // WRONG! Should be IoR
ioctlWriteActionList     = IoWR(...)  // WRONG! Should be IoW
```

**Expected**:
```go
ioctlDescListProgram     = IoR(int(HailoVdmaIoctlMagic), IoctlDescListProgram, 51)   // size=51
ioctlVdmaLaunchTransfer  = IoR(int(HailoVdmaIoctlMagic), IoctlVdmaLaunchTransfer, 153)  // IoR, NOT IoWR!
ioctlWriteActionList     = IoW(int(HailoNncIoctlMagic), IoctlWriteActionList, 24)  // IoW, NOT IoWR!
```

---

### Issue 6: DescListProgram Function Signature

**Location**: `pkg/driver/ioctl.go:289-293`

The function is missing `batchSize` and `stride` parameters.

**Current**:
```go
func (d *DeviceFile) DescListProgram(bufferHandle, bufferSize, bufferOffset uint64,
    descHandle uintptr, channelIndex uint8, startingDesc uint32, shouldBind bool,
    lastInterruptsDomain InterruptsDomain, isDebug bool) error
```

**Fix**:
```go
func (d *DeviceFile) DescListProgram(bufferHandle, bufferSize, bufferOffset uint64,
    batchSize uint32, descHandle uintptr, channelIndex uint8, startingDesc uint32,
    shouldBind bool, lastInterruptsDomain InterruptsDomain, isDebug bool,
    stride uint32) error
```

---

### Issue 7: VdmaLaunchTransfer Function Return Value

**Location**: `pkg/driver/ioctl.go:296-303`

The kernel does NOT return `descs_programed` or `launch_transfer_status`. The IOCTL is read-only.

**Current (WRONG)**:
```go
func (d *DeviceFile) VdmaLaunchTransfer(...) (uint32, int32, error) {
    // ...
    return params.DescsProgramed(), params.LaunchTransferStatus(), nil
}
```

**Fix**:
```go
func (d *DeviceFile) VdmaLaunchTransfer(...) error {
    params := NewPackedVdmaLaunchTransferParams(...)
    return d.ioctl(ioctlVdmaLaunchTransfer, unsafe.Pointer(params))
    // No return values - the kernel doesn't write back!
}
```

---

## Implementation Steps

### Step 1: Fix Packed Structure Sizes (HIGH PRIORITY)

1. Update `PackedDescListProgramParams` to 51 bytes, add `batch_size` and `stride`
2. Update `PackedVdmaTransferBuffer` layout to use `buffer_type + addr_or_fd + size`
3. Update `PackedVdmaLaunchTransferParams` to 153 bytes, 8 buffers, remove output fields
4. Update `PackedVdmaInterruptsWaitParams` to 301 bytes, 3 bytes per irq_data

### Step 2: Fix IOCTL Commands (HIGH PRIORITY)

1. Change `ioctlVdmaLaunchTransfer` from `IoWR` to `IoR`
2. Change `ioctlWriteActionList` from `IoWR` to `IoW`
3. Update size constants to match new struct sizes

### Step 3: Update Function Signatures (MEDIUM PRIORITY)

1. Add `batchSize` and `stride` to `DescListProgram`
2. Change `VdmaLaunchTransfer` to return only `error`
3. Update all call sites

### Step 4: Update Size Constants (MEDIUM PRIORITY)

```go
const (
    SizeOfPackedDescListProgramParams      = 51
    SizeOfPackedVdmaLaunchTransferParams   = 153
    SizeOfPackedVdmaInterruptsWaitParams   = 301
    MaxBuffersPerSingleTransfer            = 8
)
```

### Step 5: Add Tests (LOW PRIORITY)

1. Add tests to verify struct sizes match expected values
2. Add tests to verify field offsets match expected layout
3. Consider using `go test` with cgo to compare against C sizeof()

---

## Verification

After implementing fixes, verify:

1. `sizeof(PackedDescListProgramParams) == 51`
2. `sizeof(PackedVdmaLaunchTransferParams) == 153`
3. `sizeof(PackedVdmaInterruptsWaitParams) == 301`
4. `sizeof(PackedVdmaTransferBuffer) == 16`

Test by running a simple inference and checking:
- No IOCTL errors (EINVAL, EFAULT)
- Interrupts are received properly
- Output data is correct

---

## Files to Modify

| File | Changes |
|------|---------|
| `pkg/driver/packed.go` | Fix all packed structure layouts and sizes |
| `pkg/driver/ioctl.go` | Fix IOCTL commands, function signatures |
| `pkg/driver/constants.go` | Update `MaxBuffersPerSingleTransfer` to 8 |
| `pkg/driver/types.go` | Update unpacked types for reference |

---

## Risk Assessment

**HIGH RISK**: These are critical fixes. Without them, the driver will not function correctly.

The current code is sending malformed data to the kernel, which can cause:
- Kernel rejecting IOCTLs with EINVAL
- Kernel reading garbage data leading to undefined behavior
- Data corruption in DMA transfers
- System instability

---

## Timeline

This is a critical fix that should be implemented immediately. The changes are mechanical
(just updating struct layouts and sizes) but require careful verification.

Estimated effort: 2-4 hours for implementation + testing.
