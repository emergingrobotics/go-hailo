# Hailo Go Wrapper

A Go wrapper around the official HailoRT C++ SDK for running neural network inference on Hailo-8 AI accelerators.

## Prerequisites

- Raspberry Pi 5 with Hailo-8 M.2 AI Accelerator
- HailoRT SDK installed (via `hailo-all` package)
- Go 1.21 or later
- CMake 3.14 or later
- C++17 compatible compiler (g++)

## Installation

### 1. Install HailoRT SDK

```bash
sudo apt update
sudo apt install hailo-all
```

### 2. Build the Project

```bash
make
```

This builds both the C++ shared library and the Go binary.

### 3. (Optional) Install System-Wide

```bash
sudo make install
```

## Usage

### Basic Usage

```bash
./detect -model models/yolov5s.hef -image photo.jpg
```

### Output

```
Loading model: models/yolov5s.hef
Model loaded in 1.2s
Input: 640x640x3 (1228800 bytes)
Loading image: photo.jpg
Running inference...
Inference completed in 15ms

People detected: 3

All detections (5):
  1: person (92.3%) at [100.5, 50.2, 250.8, 400.1]
  2: person (88.7%) at [300.2, 60.5, 450.3, 410.2]
  3: person (75.1%) at [500.0, 80.3, 600.5, 390.8]
  4: car (95.2%) at [10.0, 200.0, 150.5, 350.2]
  5: dog (82.4%) at [400.0, 300.0, 500.0, 400.0]
```

## Go Package API

```go
import "github.com/anthropics/purple-hailo/new/src/go/hailo"

// Create inference engine
inference, err := hailo.NewInference("model.hef")
if err != nil {
    log.Fatal(err)
}
defer inference.Close()

// Get input requirements
info := inference.GetInputInfo()
fmt.Printf("Input: %dx%dx%d\n", info.Width, info.Height, info.Channels)

// Load and preprocess image to RGB bytes
imageData := loadImage("photo.jpg", info.Width, info.Height)

// Detect people
count, err := inference.DetectPeople(imageData)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("People: %d\n", count)

// Or get all detections
detections, err := inference.Detect(imageData)
for _, det := range detections {
    fmt.Printf("Class %d: %.1f%%\n", det.ClassID, det.Confidence*100)
}
```

## Models

Place HEF model files in the `models/` directory. Recommended models:

- **yolov5s.hef** - YOLOv5 small, good balance of speed and accuracy
- **yolov8n.hef** - YOLOv8 nano, fastest inference

Models can be obtained from:
- [Hailo Model Zoo](https://github.com/hailo-ai/hailo_model_zoo)
- Compile your own using the Hailo Dataflow Compiler

## Project Structure

```
new/
├── docs/
│   └── DESIGN.md           # Architecture documentation
├── src/
│   ├── cpp/
│   │   ├── CMakeLists.txt  # C++ build config
│   │   ├── hailo_inference.hpp
│   │   ├── hailo_inference.cpp
│   │   ├── hailo_c_api.h   # C API for cgo
│   │   └── hailo_c_api.cpp
│   └── go/
│       └── hailo/
│           └── hailo.go    # Go package
├── cmd/
│   └── detect/
│       └── main.go         # CLI tool
├── models/                  # Place HEF files here
├── go.mod
├── Makefile
└── README.md
```

## Troubleshooting

### "Failed to create VDevice"

- Check that the Hailo device is detected: `ls /dev/hailo*`
- Ensure HailoRT is installed: `dpkg -l | grep hailort`
- Check driver is loaded: `lsmod | grep hailo`

### "libhailo_wrapper.so: cannot open shared object file"

Run with library path:
```bash
LD_LIBRARY_PATH=src/cpp/build:$LD_LIBRARY_PATH ./detect ...
```

Or install system-wide: `sudo make install`

### Build errors related to HailoRT

Ensure HailoRT development files are installed:
```bash
sudo apt install libhailort-dev
```
