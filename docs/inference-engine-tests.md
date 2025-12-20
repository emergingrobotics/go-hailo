# Inference Engine Test Plan

This document provides a comprehensive test plan for each phase of the inference engine implementation. All tests are designed to be thorough, covering unit tests, edge cases, error conditions, benchmarks, and integration tests.

## Test Organization

```
/go-hailo/
├── pkg/
│   ├── driver/
│   │   ├── ioctl_test.go        # IOCTL code construction tests
│   │   ├── types_test.go        # Struct layout/size tests
│   │   ├── constants_test.go    # Constants validation
│   │   ├── errors_test.go       # Error handling tests
│   │   └── device_test.go       # Device operations (requires hardware)
│   ├── hef/
│   │   ├── header_test.go       # Header parsing tests
│   │   ├── parser_test.go       # Full parser tests
│   │   ├── checksum_test.go     # Hash validation tests
│   │   └── real_hef_test.go     # Tests with real HEF files
│   ├── device/
│   │   ├── scanner_test.go      # Device discovery tests
│   │   ├── device_test.go       # Device management tests
│   │   └── network_group_test.go # Network configuration tests
│   ├── stream/
│   │   ├── buffer_test.go       # Buffer allocation tests
│   │   ├── vstream_test.go      # VStream tests
│   │   └── dma_test.go          # DMA transfer tests
│   ├── transform/
│   │   ├── quantize_test.go     # Quantization tests
│   │   ├── format_test.go       # Format conversion tests
│   │   └── nms_test.go          # NMS parsing tests
│   └── infer/
│       ├── model_test.go        # Model API tests
│       ├── session_test.go      # Session tests
│       └── bindings_test.go     # Buffer binding tests
├── integration/
│   ├── inference_integration_test.go  # End-to-end tests
│   └── benchmark_test.go              # Performance benchmarks
└── testutil/
    ├── helpers.go               # Test utilities
    └── fakes.go                 # Mock implementations
```

## Build Tags

Tests use build tags for categorization:
- `unit` - Pure unit tests, no hardware required
- `integration` - End-to-end tests requiring Hailo hardware
- `benchmark` - Performance benchmarks

Run specific test categories:
```bash
go test -tags=unit ./...           # Unit tests only
go test -tags=integration ./...    # Integration tests (needs hardware)
go test -tags=benchmark -bench=. ./... # Benchmarks
```

---

## Phase 1: HEF Parser Tests

### 1.1 Header Parsing Tests (`pkg/hef/header_test.go`)

| Test Name | Description | Type |
|-----------|-------------|------|
| `TestValidMagicNumber` | Verify correct magic number parsing | Unit |
| `TestInvalidMagicNumber` | Reject invalid magic with proper error | Unit |
| `TestVersionZeroHeader` | Parse V0 header with MD5 hash | Unit |
| `TestVersionOneHeader` | Parse V1 header with CRC32 | Unit |
| `TestVersionTwoHeader` | Parse V2 header with XXH3 | Unit |
| `TestVersionThreeHeader` | Parse V3 header with additional info | Unit |
| `TestUnknownVersionRejected` | Reject unsupported versions | Unit |
| `TestTruncatedHeaderRejected` | Reject headers smaller than minimum | Unit |
| `TestHeaderSizeByVersion` | Verify correct header sizes per version | Unit |
| `TestHefMagicValue` | Verify magic bytes are "FEH\x01" | Unit |
| `TestVersionSpecificParserRejectsWrongVersion` | V2 parser rejects V0 data | Unit |
| `TestMinimumValidHeader` | Parse 12-byte minimum header | Unit |

### 1.2 Full Parser Tests (`pkg/hef/parser_test.go`)

| Test Name | Description | Type |
|-----------|-------------|------|
| `TestParseFromFilePath` | Load and parse from file | Unit |
| `TestParseFromBuffer` | Parse from in-memory buffer | Unit |
| `TestParseEmptyFileFails` | Reject empty files | Unit |
| `TestExtractNetworkGroupNames` | Extract network group list | Unit |
| `TestExtractInputStreamInfo` | Parse input stream metadata | Unit |
| `TestExtractOutputStreamInfo` | Parse output stream metadata | Unit |
| `TestExtractQuantizationInfo` | Parse quantization parameters | Unit |
| `TestParseMultiNetworkHef` | Handle multiple network groups | Unit |
| `TestDeviceArchitectureExtraction` | Extract target device type | Unit |
| `TestVStreamInfoWithNms` | Parse NMS layer information | Unit |
| `TestStreamInfoHwShape` | Verify hardware shape padding | Unit |

### 1.3 Checksum Tests (`pkg/hef/checksum_test.go`)

| Test Name | Description | Type |
|-----------|-------------|------|
| `TestMd5ValidationV0` | Validate MD5 for V0 files | Unit |
| `TestMd5ValidationV0Invalid` | Reject corrupted V0 files | Unit |
| `TestXxh3ValidationV2` | Validate XXH3 for V2 files | Unit |
| `TestXxh3ValidationV3` | Validate XXH3 for V3 files | Unit |
| `TestXxh3InvalidFails` | Reject corrupted V2/V3 files | Unit |
| `TestMd5HashConstantSize` | Verify MD5 is 16 bytes | Unit |
| `TestXxh3HashIs64Bit` | Verify XXH3 is 64 bits | Unit |
| `TestChecksumDataBoundaries` | Verify checksum covers correct regions | Unit |

### 1.4 Real HEF File Tests (`pkg/hef/real_hef_test.go`)

| Test Name | Description | Type |
|-----------|-------------|------|
| `TestParseYoloxHef` | Parse YOLOX model from models/ | Unit |
| `TestCompareWithHailortcli` | Verify output matches hailortcli | Unit |
| `TestMultipleHefFiles` | Parse all HEF files in test directory | Unit |
| `TestLargeHefFile` | Handle large model files | Unit |
| `BenchmarkHefParsing` | Benchmark HEF parsing speed | Bench |

---

## Phase 2: Device Management Tests

### 2.1 Device Scanner Tests (`pkg/device/scanner_test.go`)

| Test Name | Description | Type |
|-----------|-------------|------|
| `TestScanFindsDevicesInMockSysfs` | Find devices via mock sysfs | Unit |
| `TestScanEmptyWhenNoDevices` | Return empty list when no devices | Unit |
| `TestScanByBoardType` | Filter devices by board type | Unit |
| `TestDeviceIdFormat` | Verify PCIe address format | Unit |
| `TestDevicePathValidation` | Validate /dev/hailo* paths | Unit |
| `TestScanRealDevices` | Scan actual hardware | Integration |

### 2.2 Device Operations Tests (`pkg/device/device_test.go`)

| Test Name | Description | Type |
|-----------|-------------|------|
| `TestDeviceOpen` | Open device file successfully | Integration |
| `TestDeviceOpenNonexistent` | Handle missing device | Unit |
| `TestDeviceClose` | Close device cleanly | Integration |
| `TestDeviceDoubleClose` | Handle double close safely | Unit |
| `TestDeviceQueryProperties` | Query device properties | Integration |
| `TestDeviceQueryDriverInfo` | Query driver version | Integration |
| `TestDeviceIdentify` | Get device identification | Integration |
| `TestDeviceReset` | Reset neural network core | Integration |

### 2.3 Network Group Tests (`pkg/device/network_group_test.go`)

| Test Name | Description | Type |
|-----------|-------------|------|
| `TestConfigureNetworkGroup` | Configure with HEF | Integration |
| `TestActivateNetworkGroup` | Activate configured network | Integration |
| `TestDeactivateNetworkGroup` | Deactivate network | Integration |
| `TestMultipleNetworkGroups` | Switch between networks | Integration |
| `TestNetworkGroupCleanup` | Verify resource cleanup | Integration |
| `TestNetworkGroupStreamInfo` | Get stream information | Unit |
| `TestNetworkGroupBatchSize` | Configure batch size | Unit |

---

## Phase 3: Stream and Buffer Tests

### 3.1 Buffer Allocation Tests (`pkg/stream/buffer_test.go`)

| Test Name | Description | Type |
|-----------|-------------|------|
| `TestAllocateAlignedBuffer` | Allocate page-aligned buffers | Unit |
| `TestBufferReferenceCount` | Reference counting works | Unit |
| `TestBufferPoolAcquireRelease` | Pool acquire/release cycle | Unit |
| `TestBufferPoolExhaustion` | Handle pool exhaustion with timeout | Unit |
| `TestBufferPoolConcurrent` | Concurrent pool access | Unit |
| `TestBufferPoolClose` | Clean pool shutdown | Unit |
| `TestPageAlignment` | Verify page boundary alignment | Unit |
| `TestBufferZeroInitialized` | New buffers are zeroed | Unit |
| `TestDmaBufferMapping` | Map buffer for DMA | Integration |
| `TestDmaBufferUnmapping` | Unmap DMA buffer | Integration |
| `TestDmaBufferSync` | Sync buffer CPU<->device | Integration |
| `BenchmarkBufferPoolThroughput` | Measure pool performance | Bench |

### 3.2 Descriptor List Tests (`pkg/stream/descriptor_test.go`)

| Test Name | Description | Type |
|-----------|-------------|------|
| `TestDescListCreate` | Create descriptor list | Integration |
| `TestDescListRelease` | Release descriptor list | Integration |
| `TestDescListProgram` | Program descriptors | Integration |
| `TestCircularDescList` | Circular buffer mode | Integration |
| `TestLinearDescList` | Linear buffer mode | Integration |
| `TestDescListAlignment` | Verify 16-byte alignment | Unit |

### 3.3 VStream Tests (`pkg/stream/vstream_test.go`)

| Test Name | Description | Type |
|-----------|-------------|------|
| `TestInputVStreamWrite` | Write data to input stream | Integration |
| `TestOutputVStreamRead` | Read data from output stream | Integration |
| `TestVStreamFlush` | Flush pending data | Integration |
| `TestVStreamTimeout` | Handle timeout on read/write | Integration |
| `TestVStreamFrameSize` | Calculate frame size | Unit |
| `TestVStreamBufferReuse` | Buffer reuse between frames | Integration |
| `TestVStreamConcurrentAccess` | Multiple goroutines | Integration |

### 3.4 DMA Transfer Tests (`pkg/stream/dma_test.go`)

| Test Name | Description | Type |
|-----------|-------------|------|
| `TestVdmaChannelEnable` | Enable VDMA channel | Integration |
| `TestVdmaChannelDisable` | Disable VDMA channel | Integration |
| `TestVdmaLaunchTransfer` | Launch DMA transfer | Integration |
| `TestVdmaWaitInterrupt` | Wait for completion interrupt | Integration |
| `TestVdmaTransferH2D` | Host-to-device transfer | Integration |
| `TestVdmaTransferD2H` | Device-to-host transfer | Integration |
| `TestVdmaTransferBidirectional` | Simultaneous H2D and D2H | Integration |
| `BenchmarkDmaThroughput` | Measure DMA bandwidth | Bench |

---

## Phase 4: Data Transformation Tests

### 4.1 Quantization Tests (`pkg/transform/quantize_test.go`)

| Test Name | Description | Type |
|-----------|-------------|------|
| `TestQuantizeFloat32ToUint8` | Basic quantization | Unit |
| `TestDequantizeUint8ToFloat32` | Basic dequantization | Unit |
| `TestQuantizeRoundTrip` | Quantize then dequantize | Unit |
| `TestQuantizeClipping` | Values clipped to [0, 255] | Unit |
| `TestQuantizeZeroPoint` | Different zero point values | Unit |
| `TestQuantizeBatchData` | Batch quantization | Unit |
| `TestDequantizeBatchData` | Batch dequantization | Unit |
| `TestQuantizePreservesOrder` | Ordering preserved | Unit |
| `TestQuantizeUint16` | 16-bit quantization | Unit |
| `TestQuantizePerChannel` | Per-channel quantization | Unit |
| `TestQuantizeNaN` | Handle NaN inputs | Unit |
| `TestQuantizeInf` | Handle Inf inputs | Unit |
| `BenchmarkQuantize` | Single value performance | Bench |
| `BenchmarkQuantizeBatch1000` | Batch performance | Bench |
| `BenchmarkQuantizeBatch224x224x3` | Image-sized batch | Bench |

### 4.2 Format Conversion Tests (`pkg/transform/format_test.go`)

| Test Name | Description | Type |
|-----------|-------------|------|
| `TestNHWCtoNCHW` | NHWC to NCHW conversion | Unit |
| `TestNCHWtoNHWC` | NCHW to NHWC conversion | Unit |
| `TestFormatConversionRoundTrip` | Round-trip preserves data | Unit |
| `TestRGB888toRGBA` | Add alpha channel | Unit |
| `TestRGBAtoRGB888` | Remove alpha channel | Unit |
| `TestBGRtoRGB` | BGR to RGB swap | Unit |
| `TestPaddingApplication` | Apply zero padding | Unit |
| `TestPaddingRemoval` | Remove padding | Unit |
| `TestPaddingVariousSizes` | Different image sizes | Unit |
| `TestChannelPadding` | Pad channels to 8 | Unit |
| `BenchmarkNHWCtoNCHW` | Format conversion speed | Bench |
| `BenchmarkPadding224to256` | Padding speed | Bench |

### 4.3 NMS Parsing Tests (`pkg/transform/nms_test.go`)

| Test Name | Description | Type |
|-----------|-------------|------|
| `TestParseNmsByClass` | Parse BY_CLASS format | Unit |
| `TestParseNmsByScore` | Parse BY_SCORE format | Unit |
| `TestNmsScoreFiltering` | Filter by score threshold | Unit |
| `TestNmsIouFiltering` | Apply IOU-based NMS | Unit |
| `TestNmsMaxProposals` | Limit detections per class | Unit |
| `TestNmsEmptyOutput` | Handle no detections | Unit |
| `TestIouCalculation` | IoU computation accuracy | Unit |
| `TestIouNoOverlap` | Non-overlapping boxes | Unit |
| `TestIouFullOverlap` | Identical boxes | Unit |
| `TestIouPartialOverlap` | Partial overlap | Unit |
| `TestDetectionBBoxNormalization` | Coords in [0, 1] | Unit |
| `TestNmsWithMask` | Instance segmentation masks | Unit |
| `TestNmsMultiClass` | Multiple classes | Unit |
| `BenchmarkParseNmsByClass` | NMS parsing speed | Bench |
| `BenchmarkApplyNms` | NMS algorithm speed | Bench |

---

## Phase 5: High-Level Inference API Tests

### 5.1 Model Tests (`pkg/infer/model_test.go`)

| Test Name | Description | Type |
|-----------|-------------|------|
| `TestModelInputInfo` | Get input stream info | Unit |
| `TestModelOutputInfo` | Get output stream info | Unit |
| `TestModelNetworkGroups` | List network groups | Unit |
| `TestModelFrameSize` | Calculate frame sizes | Unit |
| `TestModelValidation` | Validate model structure | Unit |
| `TestModelClose` | Close model cleanly | Unit |
| `TestModelLoadFromHef` | Load from HEF file | Integration |
| `TestModelSetBatchSize` | Configure batch size | Unit |
| `TestModelSetFormatType` | Configure format type | Unit |

### 5.2 Session Tests (`pkg/infer/session_test.go`)

| Test Name | Description | Type |
|-----------|-------------|------|
| `TestSessionCreate` | Create inference session | Unit |
| `TestSessionInferSync` | Synchronous inference | Integration |
| `TestSessionInferAsync` | Asynchronous inference | Integration |
| `TestSessionClose` | Close session cleanly | Unit |
| `TestSessionTimeout` | Configure timeout | Unit |
| `TestSessionMultipleInfers` | Multiple sequential infers | Integration |
| `TestSessionClosedOperationsFail` | Ops fail after close | Unit |
| `TestSessionConcurrentInfers` | Concurrent inference calls | Integration |

### 5.3 Bindings Tests (`pkg/infer/bindings_test.go`)

| Test Name | Description | Type |
|-----------|-------------|------|
| `TestBindingsCreate` | Create empty bindings | Unit |
| `TestBindingsSetInputBuffer` | Bind input buffer | Unit |
| `TestBindingsSetOutputBuffer` | Bind output buffer | Unit |
| `TestBindingsMultipleInputs` | Multiple input tensors | Unit |
| `TestBindingsMultipleOutputs` | Multiple output tensors | Unit |
| `TestBindingsBufferSize` | Verify buffer sizes | Unit |
| `TestBindingsReuse` | Reuse bindings | Unit |
| `TestBindingsInvalidName` | Handle invalid stream names | Unit |

### 5.4 Batch Inference Tests (`pkg/infer/batch_test.go`)

| Test Name | Description | Type |
|-----------|-------------|------|
| `TestBatchInference` | Batch of inputs | Integration |
| `TestBatchSizeOne` | Single item batch | Integration |
| `TestBatchSizeLarge` | Large batch | Integration |
| `TestBatchAsyncCallback` | Async batch with callback | Integration |
| `BenchmarkBatchInference` | Batch inference speed | Bench |

---

## Phase 6: Integration and End-to-End Tests

### 6.1 Full Pipeline Tests (`integration/inference_integration_test.go`)

| Test Name | Description | Type |
|-----------|-------------|------|
| `TestFullInferencePipeline` | Complete flow from load to result | Integration |
| `TestClassificationEndToEnd` | ResNet-50 classification | Integration |
| `TestDetectionEndToEnd` | YOLO detection with NMS | Integration |
| `TestPoseEstimationEndToEnd` | Pose estimation model | Integration |
| `TestSegmentationEndToEnd` | Instance segmentation | Integration |
| `TestMultiModelEndToEnd` | Switch between models | Integration |
| `TestModelHotSwap` | Hot-swap models | Integration |

### 6.2 Stress Tests (`integration/stress_test.go`)

| Test Name | Description | Type |
|-----------|-------------|------|
| `TestStressTest` | Many sequential inferences | Integration |
| `TestConcurrentInference` | Parallel inference calls | Integration |
| `TestDeviceRecovery` | Recover from errors | Integration |
| `TestMemoryLeakDetection` | No memory leaks | Integration |
| `TestLongRunningStability` | Extended operation | Integration |

### 6.3 Benchmark Tests (`integration/benchmark_test.go`)

| Test Name | Description | Type |
|-----------|-------------|------|
| `BenchmarkSingleInference` | Single inference latency | Bench |
| `BenchmarkThroughput` | Sustained throughput | Bench |
| `BenchmarkLatencyP50` | 50th percentile latency | Bench |
| `BenchmarkLatencyP99` | 99th percentile latency | Bench |
| `BenchmarkMemoryUsage` | Memory consumption | Bench |
| `BenchmarkYoloV8n` | YOLOv8n specific | Bench |
| `BenchmarkResNet50` | ResNet-50 specific | Bench |

---

## Test Data Requirements

### Required HEF Files

Place test HEF files in `/go-hailo/testdata/`:
- `yolox_s.hef` - Object detection model
- `resnet50.hef` - Classification model (optional)
- `yolov8n_pose.hef` - Pose estimation (optional)
- `minimal.hef` - Minimal test model

### Test Image Data

Test utilities provide synthetic data:
- `testutil.MakeTestImage(w, h, c)` - Creates test image
- `testutil.MakeRandomBytes(size)` - Creates random data

---

## Error Condition Testing

Each phase should test these error conditions:

### Resource Errors
- [ ] Out of memory allocation
- [ ] File not found
- [ ] Permission denied
- [ ] Device busy

### Protocol Errors
- [ ] Invalid HEF format
- [ ] Checksum mismatch
- [ ] Unsupported version
- [ ] Corrupted data

### Hardware Errors
- [ ] Device disconnected
- [ ] Timeout waiting for interrupt
- [ ] DMA transfer failure
- [ ] Firmware communication failure

### API Errors
- [ ] Invalid parameters
- [ ] Wrong buffer sizes
- [ ] Operations after close
- [ ] Concurrent modification

---

## Test Coverage Targets

| Package | Line Coverage Target |
|---------|---------------------|
| `pkg/driver` | 80% |
| `pkg/hef` | 90% |
| `pkg/device` | 75% |
| `pkg/stream` | 80% |
| `pkg/transform` | 95% |
| `pkg/infer` | 85% |

Run coverage:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

---

## CI/CD Integration

### Unit Tests
```yaml
test-unit:
  run: go test -tags=unit -race -coverprofile=coverage.out ./...
  timeout: 5m
```

### Integration Tests (when hardware available)
```yaml
test-integration:
  run: go test -tags=integration -race ./...
  timeout: 15m
  requires: hailo-device
```

### Benchmarks
```yaml
benchmark:
  run: go test -tags=benchmark -bench=. -benchmem ./...
  timeout: 30m
```

---

## Test Implementation Status

### Phase 1: HEF Parser
- [x] header_test.go - Complete
- [x] parser_test.go - Complete
- [x] checksum_test.go - Complete
- [ ] real_hef_test.go - Needs implementation

### Phase 2: Device Management
- [x] scanner_test.go - Complete
- [ ] device_test.go - Needs hardware tests
- [ ] network_group_test.go - Needs implementation

### Phase 3: Stream/Buffer
- [x] buffer_test.go - Complete
- [ ] vstream_test.go - Needs implementation
- [ ] dma_test.go - Needs implementation

### Phase 4: Transformation
- [x] quantize_test.go - Complete
- [x] format_test.go - Complete
- [x] nms_test.go - Complete

### Phase 5: Inference API
- [x] model_test.go - Complete
- [ ] session_test.go - Needs hardware tests
- [ ] bindings_test.go - Needs implementation

### Phase 6: Integration
- [ ] inference_integration_test.go - Stub only
- [ ] benchmark_test.go - Needs implementation

---

*Last Updated: 2024-12-20*
