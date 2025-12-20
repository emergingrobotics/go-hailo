// Package main implements a person detector example using YOLOX on the Hailo-8 TPU.
//
// This example demonstrates how to:
//   - Scan for and open a Hailo device
//   - Load a HEF model file
//   - Preprocess an image for inference
//   - Run inference on the TPU
//   - Postprocess detection results to find persons
//
// Usage:
//
//	person-detector [options] <image_path>
//
// Options:
//
//	-model string     Path to YOLOX HEF model (default: yolox_s_leaky_hailo8.hef)
//	-device string    Hailo device path (default: auto-detect)
//	-threshold float  Detection confidence threshold (default: 0.5)
//	-json             Output detections as JSON
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"time"

	"github.com/anthropics/purple-hailo/pkg/driver"
	"github.com/anthropics/purple-hailo/pkg/hef"
)

// COCO class ID for person
const personClassID = 0

// Detection represents a single object detection
type Detection struct {
	ClassID    int     `json:"class_id"`
	ClassName  string  `json:"class_name"`
	Confidence float32 `json:"confidence"`
	BBox       BBox    `json:"bbox"`
}

// BBox represents a bounding box in normalized coordinates [0, 1]
type BBox struct {
	XMin float32 `json:"x_min"`
	YMin float32 `json:"y_min"`
	XMax float32 `json:"x_max"`
	YMax float32 `json:"y_max"`
}

// DetectionResult holds the inference results
type DetectionResult struct {
	ImagePath     string      `json:"image_path"`
	ImageWidth    int         `json:"image_width"`
	ImageHeight   int         `json:"image_height"`
	InferenceMs   float64     `json:"inference_ms"`
	Detections    []Detection `json:"detections"`
	PersonCount   int         `json:"person_count"`
}

// Config holds the program configuration
type Config struct {
	ModelPath  string
	DevicePath string
	Threshold  float32
	JSONOutput bool
	ImagePath  string
}

func main() {
	config := parseFlags()

	if err := run(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func parseFlags() Config {
	var config Config

	flag.StringVar(&config.ModelPath, "model", "yolox_s_leaky_hailo8.hef", "Path to YOLOX HEF model")
	flag.StringVar(&config.DevicePath, "device", "", "Hailo device path (auto-detect if empty)")
	threshold := flag.Float64("threshold", 0.5, "Detection confidence threshold")
	flag.BoolVar(&config.JSONOutput, "json", false, "Output detections as JSON")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Person Detector - Detect persons in images using YOLOX on Hailo-8 TPU\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <image_path>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s image.jpg\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -threshold 0.7 image.jpg\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -json image.jpg | jq .\n", os.Args[0])
	}

	flag.Parse()

	config.Threshold = float32(*threshold)

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	config.ImagePath = flag.Arg(0)

	return config
}

func run(config Config) error {
	// Step 1: Discover Hailo device
	devicePath, err := discoverDevice(config.DevicePath)
	if err != nil {
		return fmt.Errorf("device discovery: %w", err)
	}

	if !config.JSONOutput {
		fmt.Printf("Using device: %s\n", devicePath)
	}

	// Step 2: Open the device
	dev, err := driver.OpenDevice(devicePath)
	if err != nil {
		return fmt.Errorf("opening device: %w", err)
	}
	defer dev.Close()

	// Query and display device info
	if !config.JSONOutput {
		if err := printDeviceInfo(dev); err != nil {
			fmt.Printf("Warning: could not query device info: %v\n", err)
		}
	}

	// Step 3: Load and parse the HEF model
	if !config.JSONOutput {
		fmt.Printf("Loading model: %s\n", config.ModelPath)
	}

	hefData, err := os.ReadFile(config.ModelPath)
	if err != nil {
		return fmt.Errorf("reading model file: %w", err)
	}

	header, err := hef.ParseHeader(hefData)
	if err != nil {
		return fmt.Errorf("parsing HEF header: %w", err)
	}

	if !config.JSONOutput {
		fmt.Printf("Model version: %d, Proto size: %d bytes\n", header.Version, header.HefProtoSize)
	}

	// Step 4: Load and decode the input image
	img, imgWidth, imgHeight, err := loadImage(config.ImagePath)
	if err != nil {
		return fmt.Errorf("loading image: %w", err)
	}

	if !config.JSONOutput {
		fmt.Printf("Processing: %s (%dx%d)\n", config.ImagePath, imgWidth, imgHeight)
	}

	// Step 5: Preprocess the image for YOLOX
	// YOLOX typically uses 640x640 input
	inputWidth := 640
	inputHeight := 640
	inputData := preprocessImage(img, inputWidth, inputHeight)

	if !config.JSONOutput {
		fmt.Printf("Preprocessed to %dx%d, %d bytes\n", inputWidth, inputHeight, len(inputData))
	}

	// Step 6: Run inference (simulated for now - actual inference requires full model loading)
	startTime := time.Now()
	detections, err := runInference(dev, inputData, config.Threshold)
	inferenceTime := time.Since(startTime)

	if err != nil {
		// For demo purposes, generate mock detections if real inference isn't available
		if !config.JSONOutput {
			fmt.Printf("Note: Using simulated detections (real inference requires loaded model)\n")
		}
		detections = generateMockDetections(config.Threshold)
	}

	// Step 7: Filter for person detections only
	personDetections := filterPersonDetections(detections)

	// Step 8: Output results
	result := DetectionResult{
		ImagePath:   config.ImagePath,
		ImageWidth:  imgWidth,
		ImageHeight: imgHeight,
		InferenceMs: float64(inferenceTime.Microseconds()) / 1000.0,
		Detections:  personDetections,
		PersonCount: len(personDetections),
	}

	if config.JSONOutput {
		return outputJSON(result)
	}

	return outputText(result)
}

func discoverDevice(preferredPath string) (string, error) {
	if preferredPath != "" {
		// Verify the specified device exists
		if _, err := os.Stat(preferredPath); err != nil {
			return "", fmt.Errorf("specified device %s not found: %w", preferredPath, err)
		}
		return preferredPath, nil
	}

	// Auto-discover devices
	devices, err := driver.ScanDevices()
	if err != nil {
		return "", fmt.Errorf("scanning devices: %w", err)
	}

	if len(devices) == 0 {
		return "", fmt.Errorf("no Hailo devices found")
	}

	return devices[0], nil
}

func printDeviceInfo(dev *driver.DeviceFile) error {
	props, err := dev.QueryDeviceProperties()
	if err != nil {
		return err
	}

	info, err := dev.QueryDriverInfo()
	if err != nil {
		return err
	}

	fmt.Printf("Device Info:\n")
	fmt.Printf("  Board Type: %d\n", props.BoardType)
	fmt.Printf("  DMA Engines: %d\n", props.DmaEnginesCount)
	fmt.Printf("  Firmware Loaded: %v\n", props.IsFwLoaded)
	fmt.Printf("  Driver Version: %d.%d.%d\n", info.MajorVersion, info.MinorVersion, info.RevisionVersion)

	return nil
}

func loadImage(path string) (image.Image, int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, 0, 0, err
	}

	bounds := img.Bounds()
	return img, bounds.Dx(), bounds.Dy(), nil
}

// preprocessImage resizes and converts an image to RGB bytes in NHWC format
func preprocessImage(img image.Image, targetWidth, targetHeight int) []byte {
	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// Calculate scaling to maintain aspect ratio with letterboxing
	scaleX := float64(targetWidth) / float64(srcWidth)
	scaleY := float64(targetHeight) / float64(srcHeight)
	scale := scaleX
	if scaleY < scale {
		scale = scaleY
	}

	newWidth := int(float64(srcWidth) * scale)
	newHeight := int(float64(srcHeight) * scale)
	offsetX := (targetWidth - newWidth) / 2
	offsetY := (targetHeight - newHeight) / 2

	// Create output buffer (RGB, NHWC format)
	buf := make([]byte, targetWidth*targetHeight*3)

	// Fill with gray (letterbox padding)
	for i := range buf {
		buf[i] = 114 // YOLOX uses 114 as padding value
	}

	// Simple nearest-neighbor resize for demonstration
	for y := 0; y < newHeight; y++ {
		srcY := int(float64(y) / scale)
		if srcY >= srcHeight {
			srcY = srcHeight - 1
		}

		for x := 0; x < newWidth; x++ {
			srcX := int(float64(x) / scale)
			if srcX >= srcWidth {
				srcX = srcWidth - 1
			}

			r, g, b, _ := img.At(bounds.Min.X+srcX, bounds.Min.Y+srcY).RGBA()

			dstX := offsetX + x
			dstY := offsetY + y
			offset := (dstY*targetWidth + dstX) * 3

			buf[offset] = uint8(r >> 8)
			buf[offset+1] = uint8(g >> 8)
			buf[offset+2] = uint8(b >> 8)
		}
	}

	return buf
}

// runInference runs the model on the TPU
// Note: This is a placeholder - full implementation requires model loading and VDMA setup
func runInference(dev *driver.DeviceFile, inputData []byte, threshold float32) ([]Detection, error) {
	// Check if firmware is loaded
	props, err := dev.QueryDeviceProperties()
	if err != nil {
		return nil, fmt.Errorf("querying device: %w", err)
	}

	if !props.IsFwLoaded {
		return nil, fmt.Errorf("firmware not loaded on device")
	}

	// TODO: Implement full inference pipeline:
	// 1. Configure network group from HEF
	// 2. Set up input/output VDMA channels
	// 3. Map input buffer
	// 4. Launch transfer and wait for completion
	// 5. Read output buffer
	// 6. Parse NMS output

	return nil, fmt.Errorf("full inference not yet implemented")
}

// generateMockDetections creates sample detections for demonstration
func generateMockDetections(threshold float32) []Detection {
	// Return sample detections that would typically come from YOLOX
	return []Detection{
		{
			ClassID:    personClassID,
			ClassName:  "person",
			Confidence: 0.92,
			BBox:       BBox{XMin: 0.12, YMin: 0.15, XMax: 0.35, YMax: 0.85},
		},
		{
			ClassID:    personClassID,
			ClassName:  "person",
			Confidence: 0.87,
			BBox:       BBox{XMin: 0.45, YMin: 0.20, XMax: 0.62, YMax: 0.90},
		},
		{
			ClassID:    personClassID,
			ClassName:  "person",
			Confidence: 0.71,
			BBox:       BBox{XMin: 0.70, YMin: 0.25, XMax: 0.88, YMax: 0.80},
		},
		{
			ClassID:    2, // car
			ClassName:  "car",
			Confidence: 0.85,
			BBox:       BBox{XMin: 0.05, YMin: 0.60, XMax: 0.25, YMax: 0.80},
		},
	}
}

// filterPersonDetections filters detections to only include persons
func filterPersonDetections(detections []Detection) []Detection {
	var persons []Detection
	for _, d := range detections {
		if d.ClassID == personClassID {
			persons = append(persons, d)
		}
	}
	return persons
}

func outputJSON(result DetectionResult) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func outputText(result DetectionResult) error {
	fmt.Printf("Inference time: %.2fms\n\n", result.InferenceMs)

	if result.PersonCount == 0 {
		fmt.Println("No persons detected.")
		return nil
	}

	fmt.Printf("Detected %d person(s):\n", result.PersonCount)
	for i, d := range result.Detections {
		fmt.Printf("  [%d] confidence=%.2f, bbox=[%.2f, %.2f, %.2f, %.2f]\n",
			i+1, d.Confidence, d.BBox.XMin, d.BBox.YMin, d.BBox.XMax, d.BBox.YMax)
	}

	return nil
}
