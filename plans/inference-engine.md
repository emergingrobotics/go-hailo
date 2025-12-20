# Inference Engine Implementation Plan

This document outlines the step-by-step implementation of a complete inference engine for Hailo-8 in Go. Each phase builds on the previous, with checkboxes for tracking progress during implementation.

## Overview

### Goal
Build a native Go inference engine that communicates directly with the Hailo-8 hardware via the Linux kernel driver, without depending on libhailort.

### Architecture Summary

```
┌─────────────────────────────────────────────────────────────────┐
│                      User Application                            │
├─────────────────────────────────────────────────────────────────┤
│  pkg/infer         High-level inference API (InferModel)         │
├─────────────────────────────────────────────────────────────────┤
│  pkg/stream        VStream abstraction (buffers, DMA)            │
├─────────────────────────────────────────────────────────────────┤
│  pkg/device        Device & NetworkGroup management              │
├─────────────────────────────────────────────────────────────────┤
│  pkg/transform     Quantization, format conversion, NMS          │
├─────────────────────────────────────────────────────────────────┤
│  pkg/hef           HEF parsing (protobuf + binary)               │
├─────────────────────────────────────────────────────────────────┤
│  pkg/driver        IOCTL interface (ALREADY IMPLEMENTED)         │
├─────────────────────────────────────────────────────────────────┤
│                      Linux Kernel Driver                         │
├─────────────────────────────────────────────────────────────────┤
│                      Hailo-8 Hardware                            │
└─────────────────────────────────────────────────────────────────┘
```

### Key Reference Files
- C++ async inference: `/external/hailo-repos/hailort/hailort/libhailort/examples/cpp/async_infer_basic_example/async_infer_basic_example.cpp`
- C++ VStream example: `/external/hailo-repos/hailort/hailort/libhailort/examples/cpp/vstreams_example/vstreams_example.cpp`
- Python wrapper: `/external/hailo-repos/Hailo-Application-Code-Examples/runtime/python/common/hailo_inference.py`
- InferModel API: `/external/hailo-repos/hailort/hailort/libhailort/include/hailo/infer_model.hpp`

---

## Phase 1: Complete HEF Parser

**Objective**: Fully parse HEF files to extract network topology, stream info, and quantization parameters.

**Reference**: `/external/hailo-repos/hailort/hailort/libhailort/src/hef/` directory

### 1.1 Protobuf Schema Generation
- [ ] Locate HEF protobuf definitions in hailort source
  - Path: `/external/hailo-repos/hailort/hailort/libhailort/src/hef/hef.proto` (or similar)
- [ ] Generate Go protobuf bindings using `protoc --go_out`
- [ ] Create `pkg/hef/proto/` directory for generated code
- [ ] Verify generated types match expected structures

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 1.2 Binary Format Parsing
- [ ] Implement full header parsing for all versions (V0, V1, V2, V3)
  - Current: `pkg/hef/types.go` has header structs defined
- [ ] Parse protobuf section (starts after header, size from `HefProtoSize`)
- [ ] Parse CCWs (Core Configuration Words) section
- [ ] Handle V3 additional_info section
- [ ] Implement padding/alignment handling

**Implementation details:**
```go
// Expected structure in pkg/hef/parser.go
type Parser struct {
    data     []byte
    header   interface{} // HefHeaderV0/V2/V3
    proto    *HefProto   // generated protobuf
    ccws     []byte      // raw CCW data
}

func ParseFile(path string) (*Hef, error)
func Parse(data []byte) (*Hef, error)
```

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 1.3 Network Topology Extraction
- [ ] Extract network group names and metadata
- [ ] Parse input stream information (name, shape, format, quant_info)
- [ ] Parse output stream information
- [ ] Extract VStream metadata (user-facing stream info)
- [ ] Handle NMS layer detection and shape parsing
- [ ] Extract quantization parameters (scale, zero_point, limits)

**Key data structures:**
```go
// From current pkg/hef/types.go - expand as needed
type StreamInfo struct {
    Name        string
    Direction   StreamDirection
    Format      Format
    Shape       ImageShape3D
    HwShape     ImageShape3D      // Hardware padded shape
    HwFrameSize uint64
    QuantInfo   QuantInfo
}
```

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 1.4 Checksum Validation
- [ ] Implement XXH3 hash validation for V2/V3 headers
- [ ] Implement MD5 validation for V0 headers
- [ ] Implement CRC32 validation for V1 headers
- [ ] Add option to skip validation for performance

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 1.5 Testing
- [ ] Unit tests with real HEF files from `/go-hailo/models/`
- [ ] Verify extracted info matches `hailortcli parse-hef` output
- [ ] Test all supported HEF versions
- [ ] Benchmark parsing performance

---

## Phase 2: Device Management Layer

**Objective**: Build high-level device abstraction over the driver layer.

**Reference**:
- `/external/hailo-repos/hailort/hailort/libhailort/src/device_common/device.cpp`
- `/external/hailo-repos/hailort/hailort/libhailort/src/vdevice/vdevice.cpp`

### 2.1 Device Abstraction
- [ ] Create `pkg/device/device.go` wrapping `driver.DeviceFile`
- [ ] Implement device identification and capability queries
- [ ] Add firmware version checking
- [ ] Implement device reset functionality
- [ ] Add proper resource cleanup (Close with deferred cleanup)

**Interface design:**
```go
// pkg/device/device.go
type Device struct {
    raw        *driver.DeviceFile
    properties *driver.DeviceProperties
    info       *driver.DriverInfo
}

func Open(path string) (*Device, error)
func Scan() ([]*Device, error)
func (d *Device) Properties() DeviceProperties
func (d *Device) Configure(hef *hef.Hef) (*ConfiguredNetworkGroup, error)
func (d *Device) Close() error
```

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 2.2 Network Group Configuration
- [ ] Implement firmware control message construction
- [ ] Send configuration commands to device
- [ ] Set up VDMA channels based on HEF stream info
- [ ] Handle multi-context networks
- [ ] Implement network group activation/deactivation

**Key firmware control messages:**
```
- Configure network group
- Set batch size
- Allocate resources
- Activate/deactivate
```

**Reference**: `/external/hailo-repos/hailort/hailort/libhailort/src/network_group/network_group.cpp`

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 2.3 ConfiguredNetworkGroup
- [ ] Create struct to hold configured network state
- [ ] Track allocated VDMA channels
- [ ] Maintain reference to parent device
- [ ] Implement stream info accessors
- [ ] Add lifecycle management (activate/deactivate)

**Interface design:**
```go
// pkg/device/network_group.go
type ConfiguredNetworkGroup struct {
    device      *Device
    hef         *hef.Hef
    groupInfo   *hef.NetworkGroupInfo
    activated   bool
    channels    []allocatedChannel
}

func (ng *ConfiguredNetworkGroup) Activate() (*ActivatedNetworkGroup, error)
func (ng *ConfiguredNetworkGroup) InputStreamInfos() []hef.StreamInfo
func (ng *ConfiguredNetworkGroup) OutputStreamInfos() []hef.StreamInfo
func (ng *ConfiguredNetworkGroup) Close() error
```

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 2.4 Testing
- [ ] Test device open/close cycles
- [ ] Test configuration with various HEF files
- [ ] Test activation/deactivation sequences
- [ ] Verify resource cleanup on errors

---

## Phase 3: Stream and Buffer Management

**Objective**: Implement VStreams for data transfer between host and device.

**Reference**:
- `/external/hailo-repos/hailort/hailort/libhailort/src/net_flow/pipeline/vstream.cpp`
- `/external/hailo-repos/hailort/hailort/libhailort/src/net_flow/pipeline/vstream_builder.cpp`

### 3.1 DMA Buffer Management
- [ ] Create `pkg/stream/buffer.go` for buffer abstraction
- [ ] Implement page-aligned memory allocation (mmap)
- [ ] Create buffer pool for reuse
- [ ] Implement buffer mapping via driver IOCTLs
- [ ] Handle buffer sync (CPU <-> device cache coherency)

**Buffer alignment requirements:**
```
- Page alignment: 4096 bytes
- Descriptor alignment: 16 bytes
- Width padding: 8 bytes for certain formats
```

**Interface design:**
```go
// pkg/stream/buffer.go
type Buffer struct {
    data         []byte
    size         uint64
    mappedHandle uint64
    direction    driver.DmaDataDirection
    device       *driver.DeviceFile
}

func AllocateBuffer(dev *driver.DeviceFile, size uint64, direction DmaDirection) (*Buffer, error)
func (b *Buffer) Data() []byte
func (b *Buffer) Sync(syncType BufferSyncType) error
func (b *Buffer) Close() error

type BufferPool struct {
    buffers []*Buffer
    free    chan *Buffer
}
```

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 3.2 Descriptor List Management
- [ ] Create descriptor list abstraction
- [ ] Implement descriptor list creation via IOCTL
- [ ] Program descriptors with buffer addresses
- [ ] Handle circular vs. linear descriptor lists
- [ ] Implement descriptor list cleanup

**Reference for descriptor programming:**
```go
// Use driver.DescListCreate and driver.DescListProgram
params := driver.DescListProgramParams{
    BufferHandle:   buffer.mappedHandle,
    BufferSize:     buffer.size,
    DescHandle:     descHandle,
    ChannelIndex:   channelIdx,
    ...
}
```

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 3.3 VStream Implementation
- [ ] Create `pkg/stream/vstream.go`
- [ ] Implement InputVStream for writing data to device
- [ ] Implement OutputVStream for reading results from device
- [ ] Handle data transformation pipeline (quantize on input, dequantize on output)
- [ ] Implement blocking and non-blocking I/O modes
- [ ] Add timeout support

**Interface design:**
```go
// pkg/stream/vstream.go
type InputVStream struct {
    info         hef.VStreamInfo
    buffer       *Buffer
    descList     *DescriptorList
    channel      *VdmaChannel
    transformer  *transform.InputTransformer
}

func (vs *InputVStream) Write(data []byte) error
func (vs *InputVStream) WriteAsync(data []byte) (*WriteJob, error)
func (vs *InputVStream) Flush() error
func (vs *InputVStream) FrameSize() uint64

type OutputVStream struct {
    info         hef.VStreamInfo
    buffer       *Buffer
    descList     *DescriptorList
    channel      *VdmaChannel
    transformer  *transform.OutputTransformer
}

func (vs *OutputVStream) Read() ([]byte, error)
func (vs *OutputVStream) ReadAsync() (*ReadJob, error)
func (vs *OutputVStream) FrameSize() uint64
```

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 3.4 VDMA Channel Management
- [ ] Create channel abstraction
- [ ] Implement channel enable/disable
- [ ] Launch DMA transfers
- [ ] Wait for transfer completion (interrupts)
- [ ] Handle transfer errors and recovery

**DMA transfer flow:**
```
1. Map buffer -> driver.VdmaBufferMap
2. Create desc list -> driver.DescListCreate
3. Program descriptors -> ioctl HAILO_DESC_LIST_PROGRAM
4. Enable channel -> driver.VdmaEnableChannels
5. Launch transfer -> ioctl HAILO_VDMA_LAUNCH_TRANSFER
6. Wait for completion -> driver.VdmaInterruptsWait
7. Sync buffer -> driver.VdmaBufferSync (for reads)
```

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 3.5 VStream Builder
- [ ] Create factory for building VStreams from ConfiguredNetworkGroup
- [ ] Handle format parameters (user format vs HW format)
- [ ] Set up transformation pipeline
- [ ] Configure buffer sizes based on batch size

**Interface:**
```go
// pkg/stream/builder.go
type VStreamParams struct {
    FormatType  hef.FormatType  // User-side format
    Timeout     time.Duration
    QueueSize   int
}

func BuildInputVStreams(ng *ConfiguredNetworkGroup, params VStreamParams) ([]InputVStream, error)
func BuildOutputVStreams(ng *ConfiguredNetworkGroup, params VStreamParams) ([]OutputVStream, error)
```

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 3.6 Testing
- [ ] Test buffer allocation and mapping
- [ ] Test descriptor list lifecycle
- [ ] Test single-frame transfers
- [ ] Test multi-frame batch transfers
- [ ] Test concurrent input/output streams
- [ ] Benchmark DMA throughput

---

## Phase 4: Data Transformation Pipeline

**Objective**: Implement quantization, format conversion, and post-processing.

**Reference**:
- `/external/hailo-repos/hailort/hailort/libhailort/src/net_flow/pipeline/filter_elements.cpp`

### 4.1 Quantization/Dequantization
- [ ] Create `pkg/transform/quantize.go`
- [ ] Implement float32 -> uint8 quantization (input)
- [ ] Implement uint8 -> float32 dequantization (output)
- [ ] Support uint16 format
- [ ] Handle per-channel vs. per-tensor quantization
- [ ] Optimize with SIMD if possible

**Quantization formulas:**
```
Quantize (input):   quantized = (float_value / scale) + zero_point
Dequantize (output): float_value = (quantized - zero_point) * scale
```

**Interface:**
```go
// pkg/transform/quantize.go
type Quantizer struct {
    scale     float32
    zeroPoint float32
    limMin    float32
    limMax    float32
}

func (q *Quantizer) Quantize(input []float32, output []uint8)
func (q *Quantizer) QuantizeF32ToU16(input []float32, output []uint16)

type Dequantizer struct {
    scale     float32
    zeroPoint float32
}

func (d *Dequantizer) Dequantize(input []uint8, output []float32)
func (d *Dequantizer) DequantizeU16(input []uint16, output []float32)
```

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 4.2 Format Conversion
- [ ] Create `pkg/transform/format.go`
- [ ] Implement NHWC <-> NCHW conversion
- [ ] Handle padding/unpadding to HW shape
- [ ] Support NV12 format (for camera inputs)
- [ ] Optimize memory layout transformations

**Shape handling:**
```
User shape:  [N, H, W, C] or [N, C, H, W]
HW shape:    [N, H, W, C_padded] (padded to 8-byte boundary)
```

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 4.3 NMS Post-Processing
- [ ] Create `pkg/transform/nms.go`
- [ ] Parse NMS output format (BY_CLASS or BY_SCORE)
- [ ] Extract bounding boxes, scores, and class IDs
- [ ] Handle instance segmentation masks (NMS_WITH_BYTE_MASK)
- [ ] Convert to user-friendly detection structures

**NMS output structure:**
```go
// Detection result from NMS
type Detection struct {
    BBox     BoundingBox  // x_min, y_min, x_max, y_max (normalized 0-1)
    Score    float32
    ClassID  int
    Mask     []byte       // Optional, for instance segmentation
}

type BoundingBox struct {
    XMin, YMin, XMax, YMax float32
}
```

**NMS data format in memory:**
```
For BY_CLASS:
  [num_detections_class0][det0][det1]...[num_detections_class1][det0]...

For BY_SCORE:
  [total_detections][det0][det1][det2]... (sorted by score descending)

Each detection:
  [y_min:uint16][x_min:uint16][y_max:uint16][x_max:uint16][score:uint16][class_id:uint16]
```

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 4.4 Input Transformer Pipeline
- [ ] Create composite transformer for inputs
- [ ] Chain: Format conversion -> Quantization -> Padding
- [ ] Handle batch dimension
- [ ] Support in-place transformation where possible

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 4.5 Output Transformer Pipeline
- [ ] Create composite transformer for outputs
- [ ] Chain: Unpadding -> Dequantization -> Format conversion
- [ ] Special handling for NMS outputs
- [ ] Support streaming output processing

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 4.6 Testing
- [ ] Test quantization accuracy against reference
- [ ] Test format conversions
- [ ] Test NMS parsing with known outputs
- [ ] Benchmark transformation performance
- [ ] Test with various tensor shapes

---

## Phase 5: High-Level Inference API

**Objective**: Provide a clean, user-friendly API for running inference.

**Reference**:
- `/external/hailo-repos/hailort/hailort/libhailort/include/hailo/infer_model.hpp`
- `/external/hailo-repos/Hailo-Application-Code-Examples/runtime/python/common/hailo_inference.py`

### 5.1 InferModel
- [ ] Create `pkg/infer/model.go`
- [ ] Implement model loading from HEF path
- [ ] Expose input/output stream information
- [ ] Allow format type configuration before configure()
- [ ] Support batch size configuration

**Interface:**
```go
// pkg/infer/model.go
type InferModel struct {
    device    *device.Device
    hef       *hef.Hef
    batchSize int
    inputs    []InferStream
    outputs   []InferStream
}

type InferStream struct {
    Name     string
    Shape    []int
    Format   Format
    FrameSize uint64
}

func NewInferModel(device *device.Device, hefPath string) (*InferModel, error)
func (m *InferModel) SetBatchSize(size int)
func (m *InferModel) Input(name string) *InferStream
func (m *InferModel) Output(name string) *InferStream
func (m *InferModel) Inputs() []InferStream
func (m *InferModel) Outputs() []InferStream
func (m *InferModel) Configure() (*ConfiguredInferModel, error)
```

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 5.2 ConfiguredInferModel
- [ ] Create configured model that is ready for inference
- [ ] Set up VStreams internally
- [ ] Manage inference lifecycle
- [ ] Provide bindings creation

**Interface:**
```go
// pkg/infer/configured.go
type ConfiguredInferModel struct {
    model        *InferModel
    networkGroup *device.ActivatedNetworkGroup
    inputStreams []stream.InputVStream
    outputStreams []stream.OutputVStream
}

func (c *ConfiguredInferModel) CreateBindings() (*Bindings, error)
func (c *ConfiguredInferModel) Run(bindings *Bindings, timeout time.Duration) error
func (c *ConfiguredInferModel) RunAsync(bindings *Bindings, callback func(AsyncInferResult)) (*AsyncInferJob, error)
func (c *ConfiguredInferModel) WaitForAsyncReady(timeout time.Duration) error
func (c *ConfiguredInferModel) Close() error
```

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 5.3 Bindings
- [ ] Create bindings to hold input/output buffers
- [ ] Support MemoryView (user-provided buffers)
- [ ] Support automatic buffer allocation
- [ ] Handle multi-input/multi-output models

**Interface:**
```go
// pkg/infer/bindings.go
type Bindings struct {
    inputs  map[string]*BoundBuffer
    outputs map[string]*BoundBuffer
}

type BoundBuffer struct {
    data   []byte
    size   uint64
}

func (b *Bindings) Input(name string) *BoundBuffer
func (b *Bindings) Output(name string) *BoundBuffer
func (b *Bindings) SetInputBuffer(name string, data []byte) error
func (b *Bindings) SetOutputBuffer(name string, data []byte) error
```

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 5.4 Async Inference
- [ ] Implement AsyncInferJob for async execution
- [ ] Support callback-based completion notification
- [ ] Implement job.Wait() for synchronous waiting
- [ ] Handle multiple concurrent inference jobs
- [ ] Implement queue management

**Interface:**
```go
// pkg/infer/async.go
type AsyncInferJob struct {
    done     chan struct{}
    err      error
    bindings *Bindings
}

func (j *AsyncInferJob) Wait(timeout time.Duration) error
func (j *AsyncInferJob) Detach()

type AsyncInferResult struct {
    Status   error
    Bindings *Bindings
}
```

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 5.5 Synchronous Inference Helper
- [ ] Create simple single-call inference function
- [ ] Handle all setup/teardown internally
- [ ] Useful for one-off inference or testing

**Interface:**
```go
// pkg/infer/simple.go
func Infer(hefPath string, input []byte) ([]byte, error)
func InferBatch(hefPath string, inputs [][]byte) ([][]byte, error)
```

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 5.6 Testing
- [ ] End-to-end inference test with real model
- [ ] Test async inference
- [ ] Test batch inference
- [ ] Verify output correctness against Python reference
- [ ] Benchmark inference throughput

---

## Phase 6: Integration and Optimization

**Objective**: Complete integration, testing, and performance optimization.

### 6.1 Integration Tests
- [ ] Create `integration/inference_test.go`
- [ ] Test complete flow: load HEF -> configure -> infer -> close
- [ ] Test with YOLO detection model
- [ ] Test with classification model
- [ ] Test with pose estimation model
- [ ] Compare outputs with Python/C++ reference implementation

**Test setup:**
```go
func TestEndToEndInference(t *testing.T) {
    device, _ := device.Open("/dev/hailo0")
    defer device.Close()

    model, _ := infer.NewInferModel(device, "models/yolov8n.hef")
    configured, _ := model.Configure()
    defer configured.Close()

    bindings, _ := configured.CreateBindings()
    bindings.SetInputBuffer("input", preprocessedImage)

    configured.Run(bindings, 5*time.Second)

    output := bindings.Output("output").Data()
    // Verify output
}
```

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 6.2 Example Applications
- [ ] Update person-detector example to use real inference
- [ ] Create classification example
- [ ] Create pose estimation example
- [ ] Add image preprocessing utilities

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 6.3 Performance Optimization
- [ ] Profile and identify bottlenecks
- [ ] Optimize buffer management (reduce allocations)
- [ ] Optimize quantization with SIMD (if applicable)
- [ ] Implement buffer pre-allocation
- [ ] Add pipeline parallelism (prepare next input while processing current)

**Target metrics:**
```
- Single inference latency: < 10ms for YOLOv8n
- Throughput: > 100 FPS for YOLOv8n
- Memory usage: Stable under sustained load
```

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 6.4 Error Handling and Recovery
- [ ] Implement comprehensive error types
- [ ] Add device recovery after errors
- [ ] Handle timeout scenarios gracefully
- [ ] Add logging and diagnostics

**Notes during implementation:**
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

### 6.5 Documentation
- [ ] Document public APIs with godoc comments
- [ ] Create usage examples in pkg READMEs
- [ ] Document error codes and troubleshooting
- [ ] Add architecture documentation

---

## Appendix A: Driver IOCTL Reference

The driver layer (`pkg/driver/`) is already implemented. Key IOCTLs used by inference:

| IOCTL | Purpose | Used In |
|-------|---------|---------|
| `QueryDeviceProperties` | Get device capabilities | Device init |
| `VdmaBufferMap` | Map user buffer for DMA | Buffer setup |
| `VdmaBufferUnmap` | Unmap buffer | Cleanup |
| `VdmaBufferSync` | Sync buffer CPU<->Device | After DMA |
| `DescListCreate` | Create descriptor list | Stream setup |
| `DescListRelease` | Release descriptor list | Cleanup |
| `VdmaEnableChannels` | Enable VDMA channels | Before transfer |
| `VdmaDisableChannels` | Disable channels | After transfer |
| `VdmaLaunchTransfer` | Start DMA transfer | Data transfer |
| `VdmaInterruptsWait` | Wait for completion | After launch |
| `FwControl` | Send firmware commands | Configuration |

## Appendix B: HEF File Structure

```
┌────────────────────────────────────┐
│ Header (32-56 bytes depending on V)│
│   - Magic: 0x46454801              │
│   - Version: 0-3                   │
│   - HefProtoSize                   │
│   - Hash (XXH3/MD5/CRC)            │
│   - CCWs size                      │
├────────────────────────────────────┤
│ Protobuf Section (HefProtoSize)    │
│   - Network topology               │
│   - Stream definitions             │
│   - Quantization params            │
│   - Layer configurations           │
├────────────────────────────────────┤
│ CCWs Section (Core Config Words)   │
│   - Compiled network weights       │
│   - Hardware instructions          │
├────────────────────────────────────┤
│ Additional Info (V3 only)          │
│   - Extended metadata              │
└────────────────────────────────────┘
```

## Appendix C: Inference Data Flow

```
1. LOAD MODEL
   HEF File -> Parse -> Extract Network Info, Quant Params

2. CONFIGURE
   Network Info -> FwControl -> Device Configuration
   Allocate VDMA Channels -> Enable Channels

3. CREATE STREAMS
   For each input:  Create Buffer -> Map -> Create DescList -> InputVStream
   For each output: Create Buffer -> Map -> Create DescList -> OutputVStream

4. INFERENCE LOOP
   ┌─────────────────────────────────────────────────────────────┐
   │ Input Processing:                                           │
   │   User Data (float32) -> Quantize -> HW Format -> Buffer    │
   │   Buffer -> Sync(ToDevice) -> Launch H2D Transfer           │
   ├─────────────────────────────────────────────────────────────┤
   │ Wait for completion (interrupt)                             │
   ├─────────────────────────────────────────────────────────────┤
   │ Output Processing:                                          │
   │   Wait D2H Transfer -> Sync(ForCPU) -> Buffer               │
   │   Buffer -> HW Format -> Dequantize -> User Data (float32)  │
   └─────────────────────────────────────────────────────────────┘

5. CLEANUP
   Disable Channels -> Release DescLists -> Unmap Buffers
   Deactivate Network -> Close Device
```

## Appendix D: Risk Areas and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| HEF protobuf schema not public | Can't parse HEF | Reverse engineer from binaries or use hailortcli |
| Firmware control protocol undocumented | Can't configure device | Trace reference implementation, analyze message format |
| DMA buffer alignment issues | Crashes, data corruption | Strict alignment, comprehensive testing |
| Interrupt timing issues | Hangs, timeouts | Implement robust timeout handling |
| Multi-context network complexity | Incorrect execution | Start with single-context, add multi-context later |

## Appendix E: Testing Checklist

### Unit Tests
- [ ] `pkg/hef/parser_test.go` - HEF parsing
- [ ] `pkg/device/device_test.go` - Device management
- [ ] `pkg/stream/buffer_test.go` - Buffer allocation
- [ ] `pkg/stream/vstream_test.go` - VStream operations
- [ ] `pkg/transform/quantize_test.go` - Quantization
- [ ] `pkg/transform/format_test.go` - Format conversion
- [ ] `pkg/transform/nms_test.go` - NMS parsing
- [ ] `pkg/infer/model_test.go` - InferModel API

### Integration Tests
- [ ] End-to-end single inference
- [ ] Batch inference
- [ ] Async inference
- [ ] Error recovery
- [ ] Resource cleanup
- [ ] Performance benchmarks

### Models to Test
- [ ] YOLOv8n (object detection)
- [ ] ResNet50 (classification)
- [ ] YOLOv8n-pose (pose estimation)
- [ ] YOLO11n-seg (instance segmentation)

---

## Implementation Order Summary

1. **Phase 1**: HEF Parser (foundation for everything)
2. **Phase 4.1-4.2**: Basic Quantization & Format (needed by Phase 3)
3. **Phase 2**: Device Management (device + network group)
4. **Phase 3**: Stream Management (buffers, VStreams)
5. **Phase 4.3-4.5**: Complete Transformations (NMS, pipelines)
6. **Phase 5**: High-Level API (InferModel)
7. **Phase 6**: Integration & Optimization

**Estimated LOC**: ~5000-8000 lines of Go code

---

*Last Updated: 2024-12-20*
*Status: Planning*
