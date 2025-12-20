# Comprehensive Analysis: Hailo-8 M.2 + Raspberry Pi 5 Software Stack

## Executive Summary

This document provides a complete analysis of the Hailo software stack for the Hailo-8 M.2 accelerator on Raspberry Pi 5. The goal is to understand all components that would need to be reimplemented in Go, excluding Linux kernel drivers which will be retained.

---

## 1. Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           User Applications                                  │
│  (Python/C++ apps, GStreamer pipelines, hailo-rpi5-examples)                │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                          hailo-apps-infra                                    │
│  (Python infrastructure, GStreamer helpers, pipeline builders)              │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              TAPPAS                                          │
│  (GStreamer elements: hailonet, hailofilter, hailooverlay, etc.)            │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         pyHailoRT (Python)                                   │
│  (Python bindings via pybind11 wrapping C++ libhailort)                     │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         libhailort (C/C++)                                   │
│  Core runtime library providing:                                             │
│  - Device discovery & management                                             │
│  - HEF model loading & parsing                                               │
│  - VStreams (virtual streams) for inference                                  │
│  - Buffer management & DMA                                                   │
│  - Quantization/dequantization                                               │
│  - Network group scheduling                                                  │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼ (ioctl)
┌─────────────────────────────────────────────────────────────────────────────┐
│                     Linux Kernel Driver (RETAIN)                             │
│  - PCIe driver for Hailo-8                                                   │
│  - VDMA channel management                                                   │
│  - DMA buffer mapping                                                        │
│  - Interrupt handling                                                        │
│  - Firmware loading                                                          │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Hailo-8 Hardware                                     │
│  - 26 TOPS Neural Network Accelerator                                        │
│  - PCIe 3.0 x4 interface                                                     │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Repository Analysis

### 2.1 hailort (Core Runtime)

**Location:** `repos/hailort/hailort/`

#### Directory Structure
```
hailort/
├── libhailort/           # Core C/C++ library
│   ├── include/hailo/    # Public C/C++ headers (API surface)
│   ├── src/              # Implementation
│   │   ├── hef/          # HEF file parsing
│   │   ├── vdma/         # VDMA operations
│   │   │   ├── driver/   # Driver interface (ioctl wrappers)
│   │   │   ├── channel/  # VDMA channel management
│   │   │   └── memory/   # DMA memory management
│   │   ├── device_common/# Device abstraction
│   │   ├── network_group/# Network group management
│   │   ├── stream_common/# Stream abstractions
│   │   ├── transform/    # Data transformations
│   │   ├── vdevice/      # Virtual device (multi-device)
│   │   │   └── scheduler/# Network scheduling
│   │   ├── core_op/      # Core operations
│   │   ├── net_flow/     # Inference pipeline
│   │   └── os/           # OS abstractions (Linux/Windows)
│   ├── bindings/
│   │   ├── python/       # pyHailoRT (pybind11)
│   │   └── gstreamer/    # GStreamer elements
│   └── examples/         # C/C++ examples
├── hailortcli/           # Command-line tool
├── common/               # Shared utilities
└── hrpc/                 # RPC infrastructure (for service mode)
```

#### Key Source Files to Understand

| File/Directory | Purpose |
|---------------|---------|
| `src/hef/` | HEF file format parsing |
| `src/vdma/driver/os/posix/linux/` | Linux ioctl wrapper |
| `src/device_common/` | Device abstraction layer |
| `src/stream_common/` | Input/output stream handling |
| `src/transform/` | Data format transformations |
| `src/vdevice/` | Virtual device management |
| `src/network_group/` | Network group lifecycle |

---

### 2.2 hailort-drivers (Kernel Driver)

**Location:** `repos/hailort-drivers/`

**Note:** This will be RETAINED, not ported to Go.

#### Structure
```
hailort-drivers/
├── common/               # Shared definitions
│   └── hailo_ioctl_common.h  # CRITICAL: IOCTL definitions
├── linux/
│   ├── pcie/             # PCIe driver
│   └── vdma/             # VDMA subsystem
```

---

## 3. C API Surface (libhailort)

### 3.1 Core Types (from hailort.h)

```c
// Opaque handles - these are pointers to internal structures
typedef struct _hailo_device *hailo_device;
typedef struct _hailo_vdevice *hailo_vdevice;
typedef struct _hailo_hef *hailo_hef;
typedef struct _hailo_input_stream *hailo_input_stream;
typedef struct _hailo_output_stream *hailo_output_stream;
typedef struct _hailo_configured_network_group *hailo_configured_network_group;
typedef struct _hailo_activated_network_group *hailo_activated_network_group;
typedef struct _hailo_input_vstream *hailo_input_vstream;
typedef struct _hailo_output_vstream *hailo_output_vstream;
```

### 3.2 Key API Functions

#### Device Management
```c
hailo_status hailo_scan_devices(params, device_ids, device_ids_length);
hailo_status hailo_create_device_by_id(device_id, device);
hailo_status hailo_release_device(device);
hailo_status hailo_identify(device, device_identity);
hailo_status hailo_get_chip_temperature(device, temp_info);
hailo_status hailo_reset_device(device, mode);
```

#### Virtual Device (Multi-Device Support)
```c
hailo_status hailo_init_vdevice_params(params);
hailo_status hailo_create_vdevice(params, vdevice);
hailo_status hailo_release_vdevice(vdevice);
hailo_status hailo_configure_vdevice(vdevice, hef, params, network_groups, count);
```

#### HEF Model Loading
```c
hailo_status hailo_create_hef_file(hef, file_name);
hailo_status hailo_create_hef_buffer(hef, buffer, size);
hailo_status hailo_release_hef(hef);
hailo_status hailo_hef_get_all_vstream_infos(hef, name, vstream_infos, count);
hailo_status hailo_get_network_groups_infos(hef, infos, count);
```

#### Network Group Configuration
```c
hailo_status hailo_configure_device(device, hef, params, network_groups, count);
hailo_status hailo_activate_network_group(network_group, params, activated);
hailo_status hailo_deactivate_network_group(activated_network_group);
hailo_status hailo_shutdown_network_group(network_group);
```

#### VStreams (Virtual Streams)
```c
hailo_status hailo_create_input_vstreams(network_group, params, count, input_vstreams);
hailo_status hailo_create_output_vstreams(network_group, params, count, output_vstreams);
hailo_status hailo_vstream_write_raw_buffer(input_vstream, buffer, size);
hailo_status hailo_vstream_read_raw_buffer(output_vstream, buffer, size);
hailo_status hailo_release_input_vstream(input_vstream);
hailo_status hailo_release_output_vstream(output_vstream);
```

#### DMA Buffer Management
```c
hailo_status hailo_device_dma_map_buffer(device, address, size, direction);
hailo_status hailo_device_dma_unmap_buffer(device, address, size, direction);
hailo_status hailo_device_dma_map_dmabuf(device, dmabuf_fd, size, direction);
```

#### High-Level Inference
```c
hailo_status hailo_infer(network_group, input_buffers, output_buffers, count);
```

### 3.3 Data Types

#### Format Types
```c
typedef enum {
    HAILO_FORMAT_TYPE_AUTO = 0,
    HAILO_FORMAT_TYPE_UINT8 = 1,
    HAILO_FORMAT_TYPE_UINT16 = 2,
    HAILO_FORMAT_TYPE_FLOAT32 = 3,
} hailo_format_type_t;
```

#### Format Orders (Memory Layout)
```c
typedef enum {
    HAILO_FORMAT_ORDER_NHWC = 1,   // TensorFlow default
    HAILO_FORMAT_ORDER_NCHW = 11,  // PyTorch default
    HAILO_FORMAT_ORDER_NV12 = 13,  // YUV format
    // ... many others
} hailo_format_order_t;
```

#### Quantization Info
```c
typedef struct {
    float32_t qp_zp;        // zero point
    float32_t qp_scale;     // scale factor
    float32_t limvals_min;  // min limit
    float32_t limvals_max;  // max limit
} hailo_quant_info_t;
```

---

## 4. Driver Interface (IOCTL)

### 4.1 Device Files
- `/dev/hailo0` - First Hailo device
- `/dev/hailo1`, etc. for additional devices

### 4.2 IOCTL Command Categories

#### General IOCTLs (`HAILO_GENERAL_IOCTL_MAGIC = 'g'`)
```c
HAILO_QUERY_DEVICE_PROPERTIES  // Get device properties
HAILO_QUERY_DRIVER_INFO        // Get driver version
```

#### VDMA IOCTLs (`HAILO_VDMA_IOCTL_MAGIC = 'v'`)
```c
HAILO_VDMA_ENABLE_CHANNELS      // Enable VDMA channels
HAILO_VDMA_DISABLE_CHANNELS     // Disable VDMA channels
HAILO_VDMA_INTERRUPTS_WAIT      // Wait for interrupts
HAILO_VDMA_BUFFER_MAP           // Map user buffer for DMA
HAILO_VDMA_BUFFER_UNMAP         // Unmap DMA buffer
HAILO_VDMA_BUFFER_SYNC          // Sync buffer (CPU ↔ device)
HAILO_DESC_LIST_CREATE          // Create descriptor list
HAILO_DESC_LIST_RELEASE         // Release descriptor list
HAILO_DESC_LIST_PROGRAM         // Program descriptors
HAILO_VDMA_LAUNCH_TRANSFER      // Launch DMA transfer
HAILO_VDMA_CONTINUOUS_BUFFER_ALLOC  // Allocate continuous buffer
HAILO_VDMA_CONTINUOUS_BUFFER_FREE   // Free continuous buffer
```

#### NNC (Neural Network Core) IOCTLs (`HAILO_NNC_IOCTL_MAGIC = 'n'`)
```c
HAILO_FW_CONTROL           // Firmware control messages
HAILO_READ_NOTIFICATION    // Read device-to-host notifications
HAILO_READ_LOG             // Read firmware logs
HAILO_RESET_NN_CORE        // Reset neural network core
HAILO_WRITE_ACTION_LIST    // Write action list to device
```

### 4.3 Key Data Structures for IOCTL

```c
// Buffer mapping
struct hailo_vdma_buffer_map_params {
    uintptr_t user_address;                         // in
    size_t size;                                    // in
    enum hailo_dma_data_direction data_direction;   // in
    enum hailo_dma_buffer_type buffer_type;         // in
    size_t mapped_handle;                           // out
};

// Launch DMA transfer
struct hailo_vdma_launch_transfer_params {
    uint8_t engine_index;                           // in
    uint8_t channel_index;                          // in
    uintptr_t desc_handle;                          // in
    uint32_t starting_desc;                         // in
    bool should_bind;                               // in
    uint8_t buffers_count;                          // in
    struct hailo_vdma_transfer_buffer buffers[];    // in
};

// Firmware control
struct hailo_fw_control {
    uint8_t expected_md5[16];
    uint32_t buffer_len;
    uint8_t buffer[1500];
    uint32_t timeout_ms;
    enum hailo_cpu_id cpu_id;
};

// Device properties
struct hailo_device_properties {
    uint16_t desc_max_page_size;
    enum hailo_board_type board_type;
    enum hailo_allocation_mode allocation_mode;
    enum hailo_dma_type dma_type;
    size_t dma_engines_count;
    bool is_fw_loaded;
};
```

---

## 5. Inference Data Flow

### 5.1 Initialization Sequence

```
1. Open device (/dev/hailo0)
   └── ioctl(HAILO_QUERY_DEVICE_PROPERTIES)

2. Load HEF file
   └── Parse HEF binary format
   └── Extract network topology, weights, quantization params

3. Configure device with HEF
   └── ioctl(HAILO_FW_CONTROL) - Send configuration to firmware
   └── Create network group

4. Create VStreams
   └── Allocate input/output buffers
   └── ioctl(HAILO_VDMA_BUFFER_MAP) - Map buffers for DMA
   └── ioctl(HAILO_DESC_LIST_CREATE) - Create descriptor lists

5. Activate network group
   └── ioctl(HAILO_VDMA_ENABLE_CHANNELS)
```

### 5.2 Inference Loop

```
For each frame:
  1. Prepare input data
     └── Apply transformations (resize, normalize, quantize)
     └── Convert to device format (NHWC, UINT8, etc.)

  2. Write to input vstream
     └── ioctl(HAILO_VDMA_BUFFER_SYNC, SYNC_FOR_DEVICE)
     └── ioctl(HAILO_VDMA_LAUNCH_TRANSFER) - Start DMA

  3. Wait for completion
     └── ioctl(HAILO_VDMA_INTERRUPTS_WAIT)

  4. Read from output vstream
     └── ioctl(HAILO_VDMA_BUFFER_SYNC, SYNC_FOR_CPU)
     └── Dequantize output data
     └── Apply post-processing (NMS, etc.)
```

### 5.3 Cleanup Sequence

```
1. Deactivate network group
   └── ioctl(HAILO_VDMA_DISABLE_CHANNELS)

2. Release VStreams
   └── ioctl(HAILO_VDMA_BUFFER_UNMAP)
   └── ioctl(HAILO_DESC_LIST_RELEASE)

3. Release HEF

4. Close device
```

---

## 6. Python Bindings (pyHailoRT)

### 6.1 Binding Architecture

```
Python API (pyhailort.py)
         │
         ▼
  _pyhailort.cpython-*.so  (pybind11 compiled module)
         │
         ▼
    libhailort.so (C++ library)
```

### 6.2 Key Python Classes

```python
# From hailo_platform.pyhailort.pyhailort

class Device:
    """Represents a physical Hailo device"""
    @staticmethod
    def scan() -> List[str]
    def __init__(device_id: str)
    def identify() -> DeviceIdentity
    def configure(hef: HEF) -> ConfiguredNetworkGroup

class VDevice:
    """Virtual device managing multiple physical devices"""
    def __init__(params: VDeviceParams)
    def configure(hef: HEF) -> ConfiguredNetworkGroup

class HEF:
    """Compiled Hailo Executable Format model"""
    def __init__(hef_path: str)
    def get_network_group_names() -> List[str]
    def get_input_vstream_infos() -> List[VStreamInfo]
    def get_output_vstream_infos() -> List[VStreamInfo]

class ConfiguredNetworkGroup:
    """Network group configured on device"""
    def activate() -> ActivatedNetworkGroup
    def create_input_vstreams() -> List[InputVStream]
    def create_output_vstreams() -> List[OutputVStream]

class InputVStream:
    """Input virtual stream for sending data"""
    def send(data: np.ndarray)
    def get_info() -> VStreamInfo

class OutputVStream:
    """Output virtual stream for receiving results"""
    def recv() -> np.ndarray
    def get_info() -> VStreamInfo

class InferVStreams:
    """High-level inference wrapper"""
    def __init__(network_group, input_params, output_params)
    def infer(input_data: Dict[str, np.ndarray]) -> Dict[str, np.ndarray]
```

### 6.3 Exception Hierarchy

```python
HailoRTException (base)
├── UdpRecvError
├── InvalidProtocolVersionException
├── HailoRTFirmwareControlFailedException
├── HailoRTInvalidFrameException
├── HailoRTTimeout
├── HailoRTStreamAborted
├── HailoRTInvalidOperationException
├── HailoRTInvalidHEFException
├── HailoRTHEFNotCompatibleWithDevice
├── HailoRTDriverOperationFailedException
└── HailoCommunicationClosedException
```

---

## 7. GStreamer Integration

### 7.1 Hailo GStreamer Elements

| Element | Purpose |
|---------|---------|
| `hailonet` | Core inference element |
| `hailofilter` | Apply filters on metadata |
| `hailooverlay` | Draw detection overlays |
| `hailoaggregator` | Aggregate multi-stream results |
| `hailoroundrobin` | Round-robin stream multiplexing |
| `hailostreamrouter` | Route streams |
| `hailotileaggregator` | Aggregate tiled inference |
| `hailotilecropper` | Crop tiles for inference |
| `hailotracker` | Object tracking |

### 7.2 Typical GStreamer Pipeline

```
v4l2src → videoconvert → hailonet → hailooverlay → autovideosink
```

### 7.3 hailo-apps-infra Structure

```
hailo_apps/hailo_app_python/
├── apps/
│   ├── detection/           # Object detection pipelines
│   ├── pose_estimation/     # Pose estimation
│   ├── instance_segmentation/
│   ├── depth/               # Depth estimation
│   └── face_recognition/
├── core/
│   ├── common/
│   │   ├── buffer_utils.py  # Buffer manipulation
│   │   ├── camera_utils.py  # Camera handling
│   │   └── config_utils.py  # Configuration
│   └── gstreamer/
│       └── gstreamer_app.py # GStreamer app base class
```

---

## 8. HEF File Format

### 8.1 Overview

HEF (Hailo Executable Format) is the compiled model format containing:
- Network topology
- Quantized weights
- Quantization parameters
- Input/output specifications
- Execution schedule

### 8.2 Key Information Extracted from HEF

```c
// Per-stream information
typedef struct {
    char name[HAILO_MAX_STREAM_NAME_SIZE];
    hailo_stream_direction_t direction;
    hailo_format_t format;
    hailo_3d_image_shape_t shape;      // {height, width, features}
    hailo_3d_image_shape_t hw_shape;   // padded hardware shape
    size_t hw_frame_size;
    hailo_quant_info_t quant_info;
} hailo_stream_info_t;

// Per-vstream information
typedef struct {
    char name[HAILO_MAX_STREAM_NAME_SIZE];
    char network_name[HAILO_MAX_NETWORK_NAME_SIZE];
    hailo_stream_direction_t direction;
    hailo_format_t format;
    hailo_3d_image_shape_t shape;
    hailo_quant_info_t quant_info;
    // NMS-specific fields if applicable
    hailo_nms_shape_t nms_shape;
} hailo_vstream_info_t;
```

---

## 9. Dependencies Analysis

### 9.1 Python Dependencies (pyHailoRT)

| Package | Version | Purpose | Go Alternative |
|---------|---------|---------|----------------|
| numpy | <2 | Array operations | Native slices, gonum |
| argcomplete | * | CLI completion | cobra |
| contextlib2 | * | Context managers | Native defer |
| future | * | Py2/3 compat | N/A |
| netaddr | * | Network addresses | net package |
| netifaces | * | Network interfaces | net package |

### 9.2 Python Dependencies (hailo-rpi5-examples)

| Package | Purpose | Go Alternative |
|---------|---------|----------------|
| opencv-python | Image processing | gocv |
| setproctitle | Process naming | Native |
| python-dotenv | Env files | godotenv |
| pytest | Testing | testing package |
| gi (PyGObject) | GStreamer bindings | go-gst |

### 9.3 System Dependencies

| Dependency | Purpose |
|------------|---------|
| GStreamer 1.0 | Media pipelines |
| GLib | GStreamer dependency |
| OpenCV | Image processing |
| V4L2 | Video capture |
| libdrm | Display output |

### 9.4 C++ Dependencies (libhailort)

| Library | Purpose |
|---------|---------|
| pybind11 | Python bindings |
| spdlog | Logging |
| nlohmann_json | JSON parsing |
| protobuf | Serialization (for service) |
| gRPC | RPC (for service mode) |

---

## 10. Critical Implementation Details

### 10.1 Quantization

Input transformation:
```
quantized_value = (float_value / qp_scale) + qp_zp
```

Output transformation (dequantization):
```
float_value = (quantized_value - qp_zp) * qp_scale
```

### 10.2 Buffer Alignment

- Descriptors: 16 bytes each
- DMA buffers: Page-aligned (4096 bytes)
- Width padding: 8 bytes (for certain formats)

### 10.3 VDMA Channels

- Up to 32 channels per engine
- 16 channels per direction (H2D / D2H)
- Up to 3 VDMA engines
- Max 128 ongoing transfers

### 10.4 NMS (Non-Maximum Suppression)

Hailo devices can perform NMS on-chip. Output formats:
- `HAILO_FORMAT_ORDER_HAILO_NMS_BY_CLASS` - Grouped by class
- `HAILO_FORMAT_ORDER_HAILO_NMS_BY_SCORE` - Sorted by confidence

Detection structure:
```c
typedef struct {
    float32_t y_min, x_min, y_max, x_max;  // Normalized bbox
    float32_t score;
    uint16_t class_id;
} hailo_detection_t;
```

---

## 11. Components to Implement in Go

### 11.1 Must Implement

| Component | Complexity | Notes |
|-----------|------------|-------|
| Driver interface (ioctl wrapper) | Medium | Direct syscall to /dev/hailo* |
| HEF parser | High | Binary format parsing |
| Device management | Medium | Open, configure, close |
| VStream abstraction | High | Buffer management, DMA |
| Quantization/dequantization | Low | Simple math |
| Network group management | Medium | Lifecycle management |
| Inference pipeline | High | Orchestrate all components |

### 11.2 Optional (Can Use Alternatives)

| Component | Alternative |
|-----------|-------------|
| GStreamer elements | Use go-gst or skip GStreamer |
| Image preprocessing | gocv |
| CLI tools | cobra |

### 11.3 Will Retain (No Port Needed)

| Component | Reason |
|-----------|--------|
| Linux kernel driver | Works as-is |
| Hailo firmware | On device |

---

## 12. Go Implementation Strategy

### 12.1 Package Structure (Proposed)

```
hailo/
├── driver/           # Low-level ioctl interface
│   ├── ioctl.go      # IOCTL definitions
│   ├── device.go     # Device file operations
│   └── vdma.go       # VDMA operations
├── hef/              # HEF file parsing
│   ├── parser.go     # Binary parser
│   └── types.go      # HEF structures
├── device/           # Device abstraction
│   ├── hailo.go      # Device interface
│   └── vdevice.go    # Virtual device
├── stream/           # VStream implementation
│   ├── input.go      # Input streams
│   └── output.go     # Output streams
├── transform/        # Data transformations
│   ├── quantize.go   # Quantization
│   └── format.go     # Format conversions
├── infer/            # Inference API
│   ├── model.go      # Model loading
│   └── session.go    # Inference session
└── cmd/              # CLI tools
    └── hailort/      # hailortcli equivalent
```

### 12.2 Key Go Libraries to Use

| Purpose | Library |
|---------|---------|
| IOCTL | golang.org/x/sys/unix |
| Image processing | gocv.io/x/gocv |
| GStreamer | github.com/go-gst/go-gst |
| CLI | github.com/spf13/cobra |
| Logging | log/slog (stdlib) |

---

## 13. Testing Strategy

### 13.1 Unit Tests
- HEF parsing
- Quantization math
- Format conversions

### 13.2 Integration Tests
- Device open/close
- Buffer mapping
- Simple inference

### 13.3 End-to-End Tests
- Full inference pipeline
- Multi-model scheduling
- Performance benchmarks

---

## 14. Appendix: Version Information

| Component | Version |
|-----------|---------|
| HailoRT | 4.23.0 |
| Driver | 4.23.0 |
| Model Zoo | 2.17 |
| TAPPAS | 5.1.0 |

---

*Document generated: 2024-12-20*
*Analysis based on hailo8 branch of repositories*
