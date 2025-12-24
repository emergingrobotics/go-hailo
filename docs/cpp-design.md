# C++ Design for Hailo Inference

## Executive Summary

After extensive analysis of the HailoRT C++ SDK and the Go port effort, **the Go port is not worth continuing**. This document explains why and provides a C++ design that accomplishes the same goals with significantly less effort.

## Analysis: Is the Go Port Worth the Effort?

### The Scale of HailoRT SDK

| Metric | Value |
|--------|-------|
| Total source files | 353 (.cpp + .hpp) |
| Pipeline/VStream code | 10,500+ lines |
| VDMA subsystem code | 1,500+ lines |
| Core Op infrastructure | 100+ KB |
| Pipeline element types | 20+ classes |
| HEF parsing alone | 4,340+ lines |

### What the Go Port Achieved

- HEF protobuf parsing ✓
- Basic control protocol (identify, reset) ✓
- Network group header configuration ✓
- Action list serialization ✓
- Context info chunk transmission ✓

### What the Go Port Cannot Easily Achieve

1. **DMA Memory Management**
   - Kernel driver integration requires pinned memory
   - Go's GC can move memory, breaking DMA
   - Would need extensive cgo with manual memory management
   - Defeats the purpose of using Go

2. **Config Buffer DMA (CCW Data)**
   - CCW writes go through DMA config channels, not action lists
   - Requires descriptor list setup, memory mapping
   - 15,746 bytes of C++ just for config_buffer.cpp

3. **Boundary Channel Setup**
   - Input/output streams need VDMA channels configured
   - Descriptor lists, transfer requests, interrupt handling
   - Deeply integrated with kernel driver

4. **Pipeline Execution Model**
   - 20+ element types for data transformation
   - Async execution with atomic status propagation
   - Per-element statistics and latency tracking
   - This alone is 10,500+ lines of carefully tuned C++

5. **Resource Management**
   - 53KB resource_manager.cpp coordinates everything
   - DDR cache, intermediate buffers, action lists
   - Context switching between network groups

### Why the Go Port Fails

The fundamental issue: **Inference is not just control protocol messages.**

The Go port focused on:
```
Parse HEF → Send control messages → Enable → Inference works!
```

But reality is:
```
Parse HEF
   ↓
Allocate DMA-able memory (kernel integration)
   ↓
Set up config buffers with CCW data (via DMA)
   ↓
Create boundary channels for input/output
   ↓
Map user memory for DMA transfers
   ↓
Send action lists (context switch data)
   ↓
Send network group header
   ↓
Enable core op
   ↓
Launch DMA transfers for inference
   ↓
Handle completion interrupts
   ↓
Process output through pipeline
```

The Go port got through steps 1, 6, 7, 8 but skipped 2-5, which is why EnableCoreOp times out - the firmware is waiting for config buffer data that was never sent via DMA.

### Cost-Benefit Analysis

| Approach | Effort | Maintenance | Performance | Reliability |
|----------|--------|-------------|-------------|-------------|
| Continue Go port | 6-12 months | High (SDK updates) | Lower (GC, cgo) | Uncertain |
| C++ with SDK | 1-2 weeks | Low (SDK maintained) | Optimal | Production-ready |
| Go + cgo wrapper | 2-4 weeks | Medium | Good | Good |

**Recommendation: Use C++ with the official SDK.**

---

## C++ Design

### Goals

1. Load a HEF model file
2. Run inference on input images
3. Get detection/classification results
4. Be simple and maintainable

### Architecture Options

The SDK provides three API levels:

#### Option 1: InferModel API (Highest Level) - Recommended

```cpp
// Simplest possible inference
auto vdevice = VDevice::create();
auto infer_model = vdevice->create_infer_model("model.hef");
auto configured_model = infer_model->configure();
auto bindings = configured_model->create_bindings();

bindings.input()->set_buffer(input_data);
bindings.output()->set_buffer(output_data);

configured_model->run(bindings);
// Results in output_data
```

**Pros:**
- ~20 lines of code
- Handles all complexity internally
- Best for most use cases

**Cons:**
- Less control over pipeline
- May not expose all features

#### Option 2: VStream API (Medium Level)

```cpp
auto device = Device::create_pcie();
auto hef = Hef::create("model.hef");
auto network_group = device->configure(hef);
auto activated = network_group->activate();

auto input_params = network_group->make_input_vstream_params();
auto output_params = network_group->make_output_vstream_params();

auto input_vstreams = VStreamsBuilder::create_input_vstreams(*network_group, input_params);
auto output_vstreams = VStreamsBuilder::create_output_vstreams(*network_group, output_params);

input_vstreams[0].write(input_buffer);
output_vstreams[0].read(output_buffer);
```

**Pros:**
- More control over streams
- Access to stream metadata
- Can handle multi-input/output models

**Cons:**
- More boilerplate
- Manual stream management

#### Option 3: Low-Level Streams (Lowest Level)

```cpp
// Direct InputStream/OutputStream access
// Used by the SDK internally
// Not recommended for applications
```

### Recommended Design: Simple Inference Application

```
┌─────────────────────────────────────────────────────────────┐
│                    hailo_inference                          │
├─────────────────────────────────────────────────────────────┤
│  main.cpp          - Entry point, argument parsing          │
│  inference.cpp/hpp - InferenceEngine class                  │
│  image_utils.cpp   - Image loading/preprocessing            │
│  detection.cpp     - Detection result parsing               │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    HailoRT SDK                              │
│  libhailort.so - All the complexity handled for us          │
└─────────────────────────────────────────────────────────────┘
```

### Implementation

#### File: inference.hpp

```cpp
#pragma once

#include <hailo/hailort.hpp>
#include <string>
#include <vector>
#include <memory>

namespace hailo_app {

struct Detection {
    float x_min, y_min, x_max, y_max;
    float confidence;
    int class_id;
};

class InferenceEngine {
public:
    // Create from HEF file path
    static std::unique_ptr<InferenceEngine> create(const std::string& hef_path);

    // Run inference on raw input data
    // Returns detections for object detection models
    std::vector<Detection> infer(const uint8_t* input_data, size_t input_size);

    // Get input requirements
    size_t get_input_size() const;
    int get_input_width() const;
    int get_input_height() const;
    int get_input_channels() const;

    ~InferenceEngine();

private:
    InferenceEngine();

    std::unique_ptr<hailort::VDevice> m_vdevice;
    std::shared_ptr<hailort::ConfiguredInferModel> m_configured_model;
    hailort::ConfiguredInferModel::Bindings m_bindings;

    // Cached model info
    size_t m_input_size;
    int m_width, m_height, m_channels;
};

} // namespace hailo_app
```

#### File: inference.cpp

```cpp
#include "inference.hpp"
#include <stdexcept>

namespace hailo_app {

InferenceEngine::InferenceEngine() = default;
InferenceEngine::~InferenceEngine() = default;

std::unique_ptr<InferenceEngine> InferenceEngine::create(const std::string& hef_path) {
    auto engine = std::unique_ptr<InferenceEngine>(new InferenceEngine());

    // Create virtual device (handles PCIe device discovery)
    auto vdevice_exp = hailort::VDevice::create();
    if (!vdevice_exp) {
        throw std::runtime_error("Failed to create VDevice: " +
            std::to_string(vdevice_exp.status()));
    }
    engine->m_vdevice = vdevice_exp.release();

    // Load HEF and create infer model
    auto infer_model_exp = engine->m_vdevice->create_infer_model(hef_path);
    if (!infer_model_exp) {
        throw std::runtime_error("Failed to create InferModel: " +
            std::to_string(infer_model_exp.status()));
    }
    auto infer_model = infer_model_exp.release();

    // Configure the model
    auto configured_exp = infer_model->configure();
    if (!configured_exp) {
        throw std::runtime_error("Failed to configure model: " +
            std::to_string(configured_exp.status()));
    }
    engine->m_configured_model = configured_exp.release();

    // Create bindings for input/output
    auto bindings_exp = engine->m_configured_model->create_bindings();
    if (!bindings_exp) {
        throw std::runtime_error("Failed to create bindings: " +
            std::to_string(bindings_exp.status()));
    }
    engine->m_bindings = bindings_exp.release();

    // Cache input information
    const auto& input_stream = engine->m_configured_model->input();
    auto shape = input_stream.shape();
    engine->m_height = shape.height;
    engine->m_width = shape.width;
    engine->m_channels = shape.features;
    engine->m_input_size = input_stream.get_frame_size();

    return engine;
}

std::vector<Detection> InferenceEngine::infer(const uint8_t* input_data, size_t input_size) {
    if (input_size != m_input_size) {
        throw std::runtime_error("Input size mismatch: expected " +
            std::to_string(m_input_size) + ", got " + std::to_string(input_size));
    }

    // Set input buffer
    auto status = m_bindings.input()->set_buffer(
        hailort::MemoryView(const_cast<uint8_t*>(input_data), input_size));
    if (status != HAILO_SUCCESS) {
        throw std::runtime_error("Failed to set input buffer");
    }

    // Allocate output buffer
    auto output_size = m_configured_model->output().get_frame_size();
    std::vector<uint8_t> output_buffer(output_size);

    status = m_bindings.output()->set_buffer(
        hailort::MemoryView(output_buffer.data(), output_size));
    if (status != HAILO_SUCCESS) {
        throw std::runtime_error("Failed to set output buffer");
    }

    // Run inference
    status = m_configured_model->run(m_bindings);
    if (status != HAILO_SUCCESS) {
        throw std::runtime_error("Inference failed");
    }

    // Parse detections from output
    // This depends on the model's output format
    return parse_detections(output_buffer);
}

size_t InferenceEngine::get_input_size() const { return m_input_size; }
int InferenceEngine::get_input_width() const { return m_width; }
int InferenceEngine::get_input_height() const { return m_height; }
int InferenceEngine::get_input_channels() const { return m_channels; }

} // namespace hailo_app
```

#### File: main.cpp

```cpp
#include "inference.hpp"
#include <iostream>
#include <fstream>
#include <chrono>

// Simple image loading (in practice, use OpenCV or stb_image)
std::vector<uint8_t> load_raw_image(const std::string& path, size_t expected_size) {
    std::ifstream file(path, std::ios::binary);
    if (!file) {
        throw std::runtime_error("Cannot open image file: " + path);
    }

    std::vector<uint8_t> data(expected_size);
    file.read(reinterpret_cast<char*>(data.data()), expected_size);

    if (file.gcount() != static_cast<std::streamsize>(expected_size)) {
        throw std::runtime_error("Image size mismatch");
    }

    return data;
}

int main(int argc, char* argv[]) {
    if (argc < 3) {
        std::cerr << "Usage: " << argv[0] << " <model.hef> <image.raw>" << std::endl;
        return 1;
    }

    try {
        // Create inference engine
        std::cout << "Loading model: " << argv[1] << std::endl;
        auto engine = hailo_app::InferenceEngine::create(argv[1]);

        std::cout << "Input: " << engine->get_input_width() << "x"
                  << engine->get_input_height() << "x"
                  << engine->get_input_channels() << std::endl;

        // Load image
        auto image_data = load_raw_image(argv[2], engine->get_input_size());

        // Run inference with timing
        auto start = std::chrono::high_resolution_clock::now();
        auto detections = engine->infer(image_data.data(), image_data.size());
        auto end = std::chrono::high_resolution_clock::now();

        auto duration = std::chrono::duration_cast<std::chrono::milliseconds>(end - start);
        std::cout << "Inference time: " << duration.count() << " ms" << std::endl;

        // Print detections
        std::cout << "Detections: " << detections.size() << std::endl;
        for (const auto& det : detections) {
            std::cout << "  Class " << det.class_id
                      << " @ [" << det.x_min << "," << det.y_min
                      << "," << det.x_max << "," << det.y_max << "]"
                      << " conf=" << det.confidence << std::endl;
        }

    } catch (const std::exception& e) {
        std::cerr << "Error: " << e.what() << std::endl;
        return 1;
    }

    return 0;
}
```

#### File: CMakeLists.txt

```cmake
cmake_minimum_required(VERSION 3.14)
project(hailo_inference)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Find HailoRT
find_package(HailoRT REQUIRED)

add_executable(hailo_inference
    main.cpp
    inference.cpp
)

target_link_libraries(hailo_inference
    HailoRT::libhailort
)

# Optional: Link OpenCV for image processing
# find_package(OpenCV REQUIRED)
# target_link_libraries(hailo_inference ${OpenCV_LIBS})
```

### Building

```bash
# Ensure HailoRT SDK is installed
# On Raspberry Pi with hailo-all package:
mkdir build && cd build
cmake ..
make

# Run
./hailo_inference /path/to/model.hef /path/to/image.raw
```

### Integration with Go (If Needed)

If Go integration is still desired, create a thin C wrapper:

```c
// hailo_simple.h
#ifndef HAILO_SIMPLE_H
#define HAILO_SIMPLE_H

#ifdef __cplusplus
extern "C" {
#endif

typedef struct HailoInference HailoInference;

HailoInference* hailo_create(const char* hef_path);
void hailo_destroy(HailoInference* h);

int hailo_get_input_size(HailoInference* h);
int hailo_infer(HailoInference* h, const uint8_t* input, int input_size,
                float* detections, int max_detections);

#ifdef __cplusplus
}
#endif
#endif
```

Then use cgo to call this wrapper from Go:

```go
// #cgo LDFLAGS: -lhailo_simple -lhailort
// #include "hailo_simple.h"
import "C"

func NewInference(hefPath string) *Inference {
    cPath := C.CString(hefPath)
    defer C.free(unsafe.Pointer(cPath))
    return &Inference{handle: C.hailo_create(cPath)}
}
```

This approach:
- Uses the battle-tested HailoRT SDK
- Minimal custom code to maintain
- Gets updates automatically with SDK updates
- Full performance of the C++ implementation
- Clean Go interface if needed

---

## Conclusion

The Go port effort taught us valuable lessons about the Hailo architecture:

1. **The HEF format and control protocol** are well understood
2. **The DMA/VDMA subsystem** is the real complexity
3. **Config buffer management** is critical for inference
4. **The SDK is highly optimized** for the hardware

Rather than spending 6-12 months replicating 353 files of C++, we should:

1. **Use the official C++ SDK** - it works, it's maintained, it's optimized
2. **Write simple application code** - ~200 lines vs 10,000+
3. **Add Go bindings if needed** - thin cgo wrapper around C API

The pragmatic path forward is C++ with the official SDK.
