# Person Detector Example Plan

## Overview

This example demonstrates how to use the purple-hailo Go library to perform person detection using a YOLOX model on the Hailo-8 TPU.

## Prerequisites

1. **Hardware**: Raspberry Pi 5 with Hailo-8 M.2 AI Accelerator
2. **Driver**: Hailo driver installed and `/dev/hailo0` accessible
3. **Model**: YOLOX model compiled for Hailo-8 (`.hef` file)
   - Recommended: `yolox_s_leaky_hailo8.hef` from Hailo Model Zoo

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Person Detector                         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│   ┌───────────┐    ┌───────────┐    ┌──────────────────┐   │
│   │  Image    │───▶│ Preprocess│───▶│  Hailo-8 TPU    │   │
│   │  Input    │    │  (Resize/ │    │  (YOLOX Model)  │   │
│   │           │    │  Normalize)│    │                  │   │
│   └───────────┘    └───────────┘    └────────┬─────────┘   │
│                                               │             │
│   ┌───────────┐    ┌───────────┐    ┌────────▼─────────┐   │
│   │  Output   │◀───│ Postproc  │◀───│  Raw Output     │   │
│   │(Detections)│    │  (NMS/    │    │  (Quantized)    │   │
│   │           │    │  Filter)  │    │                  │   │
│   └───────────┘    └───────────┘    └──────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Implementation Steps

### Step 1: Device Discovery and Initialization

```go
// Scan for available Hailo devices
devices, err := driver.ScanDevices()
if err != nil {
    log.Fatal(err)
}

// Open the first device
dev, err := driver.OpenDevice(devices[0])
if err != nil {
    log.Fatal(err)
}
defer dev.Close()
```

### Step 2: Load the HEF Model

```go
// Parse the HEF file
hefData, err := os.ReadFile("yolox_s_leaky_hailo8.hef")
if err != nil {
    log.Fatal(err)
}

hef, err := hef.Parse(hefData)
if err != nil {
    log.Fatal(err)
}

// Get input/output stream information
networkGroup := hef.NetworkGroups[0]
inputStream := networkGroup.InputStreams[0]
outputStreams := networkGroup.OutputStreams
```

### Step 3: Image Preprocessing

The YOLOX model typically expects:
- Input size: 640x640 (or 416x416 depending on model variant)
- Format: RGB, NHWC layout
- Data type: uint8 (0-255)

```go
func preprocessImage(img image.Image, targetWidth, targetHeight int) []byte {
    // Resize image maintaining aspect ratio with letterboxing
    resized := resize.Resize(uint(targetWidth), uint(targetHeight), img, resize.Bilinear)

    // Convert to RGB bytes in NHWC format
    buf := make([]byte, targetWidth*targetHeight*3)
    for y := 0; y < targetHeight; y++ {
        for x := 0; x < targetWidth; x++ {
            r, g, b, _ := resized.At(x, y).RGBA()
            offset := (y*targetWidth + x) * 3
            buf[offset] = uint8(r >> 8)
            buf[offset+1] = uint8(g >> 8)
            buf[offset+2] = uint8(b >> 8)
        }
    }
    return buf
}
```

### Step 4: Run Inference

```go
// Create inference session
session, err := infer.NewSession(dev, hef)
if err != nil {
    log.Fatal(err)
}
defer session.Close()

// Run inference
outputs, err := session.Infer(map[string][]byte{
    inputStream.Name: preprocessedImage,
})
if err != nil {
    log.Fatal(err)
}
```

### Step 5: Postprocess Detections

YOLOX outputs require:
1. Dequantization (convert from int8 to float32)
2. Decoding bounding boxes (applying anchor offsets)
3. Non-Maximum Suppression (NMS)
4. Filtering by confidence threshold

```go
// Parse NMS output (if using NMS on device)
detections := transform.ParseNmsByClass(outputData, 80) // COCO has 80 classes

// Filter for person class (class_id = 0 in COCO)
personDetections := make([]transform.Detection, 0)
for _, d := range detections {
    if d.ClassId == 0 && d.Score >= 0.5 { // person class with 50% confidence
        personDetections = append(personDetections, d)
    }
}

// Apply additional NMS if needed
personDetections = transform.ApplyNms(personDetections, 0.45)
```

### Step 6: Output Results

```go
type Detection struct {
    BBox       BoundingBox
    Confidence float32
    Label      string
}

type BoundingBox struct {
    X1, Y1, X2, Y2 float32 // Normalized coordinates [0, 1]
}

// Print detections
for i, det := range personDetections {
    fmt.Printf("Person %d: confidence=%.2f, bbox=[%.2f, %.2f, %.2f, %.2f]\n",
        i+1, det.Score, det.BBox.XMin, det.BBox.YMin, det.BBox.XMax, det.BBox.YMax)
}
```

## COCO Class Labels

For reference, the COCO dataset class ordering (relevant subset):
- 0: person
- 1: bicycle
- 2: car
- ... (80 classes total)

## Command Line Interface

```
Usage: person-detector [options] <image_path>

Options:
  -model string     Path to YOLOX HEF model (default: yolox_s_leaky_hailo8.hef)
  -device string    Hailo device path (default: /dev/hailo0)
  -threshold float  Detection confidence threshold (default: 0.5)
  -output string    Output image path with drawn boxes (optional)
  -json             Output detections as JSON

Examples:
  person-detector image.jpg
  person-detector -threshold 0.7 -output result.jpg image.jpg
  person-detector -json image.jpg | jq .
```

## Expected Output

```
$ person-detector sample.jpg
Loading model: yolox_s_leaky_hailo8.hef
Using device: /dev/hailo0
Processing: sample.jpg (1920x1080)
Inference time: 12.3ms

Detected 3 persons:
  [1] confidence=0.92, bbox=[0.12, 0.15, 0.35, 0.85]
  [2] confidence=0.87, bbox=[0.45, 0.20, 0.62, 0.90]
  [3] confidence=0.71, bbox=[0.70, 0.25, 0.88, 0.80]
```

## File Structure

```
examples/
└── person-detector/
    ├── main.go           # Entry point and CLI handling
    ├── preprocess.go     # Image preprocessing utilities
    ├── postprocess.go    # Detection postprocessing (NMS, filtering)
    └── draw.go           # Optional: Draw bounding boxes on image
```

## Dependencies

The example uses only the purple-hailo library and standard library:

```go
import (
    "github.com/anthropics/purple-hailo/pkg/driver"
    "github.com/anthropics/purple-hailo/pkg/hef"
    "github.com/anthropics/purple-hailo/pkg/infer"
    "github.com/anthropics/purple-hailo/pkg/transform"
)
```

For image handling, the standard library `image` package is used:
```go
import (
    "image"
    "image/jpeg"
    "image/png"
)
```

## Error Handling

The example should handle these common error cases:
1. No Hailo device found
2. Invalid or missing model file
3. Invalid image file
4. Inference timeout
5. Device busy/locked

## Performance Considerations

1. **Batch Processing**: For multiple images, reuse the session and model
2. **Memory Management**: Pre-allocate input/output buffers
3. **Async Inference**: Use async inference for pipeline processing
4. **Model Selection**: Use appropriate YOLOX variant for speed vs accuracy tradeoff:
   - YOLOX-Nano: Fastest, lower accuracy
   - YOLOX-S: Good balance
   - YOLOX-M/L: Higher accuracy, slower

## Testing

Run with a sample image:
```bash
make build-examples
./bin/person-detector testdata/people.jpg
```

## Future Enhancements

1. Video/webcam input support
2. Multi-class detection mode
3. Tracking across frames
4. REST API server mode
5. Performance benchmarking tools
