# Hailo Go Wrapper Design

## Overview

This project provides a Go wrapper around the official HailoRT C++ SDK, enabling Go applications to run neural network inference on Hailo-8 AI accelerators.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Go Application                             │
│                   (cmd/detect/main.go)                          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Go Package                                 │
│                   (src/go/hailo/hailo.go)                       │
│                                                                 │
│   type Inference struct { ... }                                 │
│   func NewInference(hefPath string) (*Inference, error)         │
│   func (i *Inference) DetectPeople(imageData []byte) (int, error)│
│   func (i *Inference) Close()                                   │
└─────────────────────────────────────────────────────────────────┘
                              │ cgo
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      C API Wrapper                              │
│                   (src/cpp/hailo_c_api.h)                       │
│                                                                 │
│   hailo_inference_t* hailo_create(const char* hef_path)         │
│   void hailo_destroy(hailo_inference_t* h)                      │
│   int hailo_detect_people(hailo_inference_t* h, ...)            │
│   hailo_input_info_t hailo_get_input_info(hailo_inference_t* h) │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    C++ Implementation                           │
│               (src/cpp/hailo_inference.cpp)                     │
│                                                                 │
│   class HailoInference {                                        │
│       VDevice, ConfiguredInferModel, Bindings                   │
│       detectPeople(input) -> count                              │
│   }                                                             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    HailoRT SDK                                  │
│                   (libhailort.so)                               │
│                                                                 │
│   - Device management                                           │
│   - HEF loading                                                 │
│   - DMA/VDMA handling                                           │
│   - Config buffers                                              │
│   - Inference execution                                         │
│   - Output post-processing                                      │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Hailo-8 Hardware                             │
│                   (/dev/hailo0)                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Directory Structure

```
new/
├── docs/
│   └── DESIGN.md           # This file
├── src/
│   ├── cpp/
│   │   ├── CMakeLists.txt  # Build configuration
│   │   ├── hailo_inference.hpp  # C++ class header
│   │   ├── hailo_inference.cpp  # C++ implementation
│   │   ├── hailo_c_api.h        # C API header (for cgo)
│   │   └── hailo_c_api.cpp      # C API implementation
│   └── go/
│       └── hailo/
│           └── hailo.go    # Go package with cgo bindings
├── cmd/
│   └── detect/
│       └── main.go         # Example CLI tool
├── models/                 # Place HEF files here
├── go.mod
├── Makefile
└── README.md
```

## Components

### 1. C++ Implementation (hailo_inference.cpp)

Uses the high-level HailoRT InferModel API:
- Creates VDevice (auto-discovers Hailo hardware)
- Loads HEF model file
- Configures inference bindings
- Runs inference and parses results
- Filters detections for "person" class

**Key Classes:**
- `HailoInference` - Main inference engine class
- Wraps `hailort::VDevice`, `hailort::ConfiguredInferModel`

### 2. C API Wrapper (hailo_c_api.h/cpp)

Provides a C-compatible interface for cgo:
- `hailo_create()` - Create inference engine
- `hailo_destroy()` - Clean up resources
- `hailo_get_input_info()` - Get model input requirements
- `hailo_detect_people()` - Run inference, return person count

### 3. Go Package (hailo/hailo.go)

Go wrapper using cgo:
- `Inference` struct - Wraps C pointer
- `NewInference()` - Constructor
- `DetectPeople()` - Main inference method
- `Close()` - Cleanup
- `GetInputInfo()` - Get image requirements

### 4. CLI Tool (cmd/detect/main.go)

Example application:
- Takes image path and model path as arguments
- Loads image (JPEG/PNG support via Go stdlib)
- Resizes to model input size
- Runs inference
- Prints number of people detected

## Data Flow

```
1. User provides: image.jpg + model.hef

2. Go CLI loads image:
   - Decode JPEG/PNG
   - Resize to model dimensions (e.g., 640x640)
   - Convert to RGB bytes

3. Go calls C wrapper:
   - hailo_detect_people(handle, rgb_data, width, height)

4. C++ runs inference:
   - Set input buffer
   - Run model
   - Parse YOLO output format
   - Filter for class_id == 0 (person)
   - Apply NMS if needed
   - Return count

5. Result returned to Go → printed to user
```

## Model Requirements

This design expects a YOLO-style object detection model:
- Input: RGB image (typically 640x640x3)
- Output: Detections with [x, y, w, h, confidence, class_scores...]
- Class 0 should be "person" (COCO dataset convention)

Recommended models:
- yolov5s (fast, good accuracy)
- yolov8n (newer, efficient)

## Build Process

```bash
# 1. Build C++ library
cd src/cpp
mkdir build && cd build
cmake ..
make

# 2. Install library
sudo make install
# Or set LD_LIBRARY_PATH

# 3. Build Go binary
cd ../../..
go build -o detect ./cmd/detect

# 4. Run
./detect -model models/yolov5s.hef -image test.jpg
```

## Error Handling

- C++ exceptions caught and converted to error codes
- C API returns negative values on error
- Go wrapper converts to Go errors
- Detailed error messages available via `hailo_get_last_error()`

## Thread Safety

- Each `Inference` instance is NOT thread-safe
- Create separate instances for concurrent inference
- Or use a mutex in the calling code

## Performance Considerations

- Model loading is slow (~1-2 seconds) - do once at startup
- Inference is fast (~10-50ms depending on model)
- Image resize in Go may be bottleneck - consider doing in C++ if needed
- Memory is pinned in C++ for DMA efficiency

## Future Enhancements

1. Batch inference support
2. Async inference with callbacks
3. Multiple model support
4. Direct camera input (bypassing Go image decode)
5. GPU-accelerated image preprocessing
