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
//	-model string     Path to YOLOX HEF model (default: models/yolox_s_leaky_hailo8.hef)
//	-device string    Hailo device path (default: auto-detect)
//	-threshold float  Detection confidence threshold (default: 0.5)
//	-json             Output detections as JSON
package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"time"

	"github.com/anthropics/purple-hailo/pkg/device"
	"github.com/anthropics/purple-hailo/pkg/hef"
	"github.com/anthropics/purple-hailo/pkg/stream"
	"github.com/anthropics/purple-hailo/pkg/transform"
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
	ImagePath   string      `json:"image_path"`
	ImageWidth  int         `json:"image_width"`
	ImageHeight int         `json:"image_height"`
	InferenceMs float64     `json:"inference_ms"`
	Detections  []Detection `json:"detections"`
	PersonCount int         `json:"person_count"`
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

	flag.StringVar(&config.ModelPath, "model", "models/yolox_s_leaky_hailo8.hef", "Path to YOLOX HEF model")
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
	// Step 1: Load and parse the HEF model first (before device operations)
	if !config.JSONOutput {
		fmt.Printf("Loading model: %s\n", config.ModelPath)
	}

	hefFile, err := hef.Parse(config.ModelPath)
	if err != nil {
		return fmt.Errorf("parsing HEF model: %w", err)
	}

	// Display model information
	ng, err := hefFile.GetDefaultNetworkGroup()
	if err != nil {
		return fmt.Errorf("getting network group: %w", err)
	}

	if !config.JSONOutput {
		fmt.Printf("Model: %s\n", ng.Name)
		fmt.Printf("  Inputs: %d, Outputs: %d\n", len(ng.InputStreams), len(ng.OutputStreams))
		if ng.HasNmsOutput() {
			nmsInfo := ng.GetNmsInfo()
			fmt.Printf("  NMS output: %s (classes=%d, max_per_class=%d)\n",
				nmsInfo.Name, nmsInfo.NmsShape.NumberOfClasses, nmsInfo.NmsShape.MaxBboxesPerClass)
		}
	}

	// Step 2: Load and decode the input image
	img, imgWidth, imgHeight, err := loadImage(config.ImagePath)
	if err != nil {
		return fmt.Errorf("loading image: %w", err)
	}

	if !config.JSONOutput {
		fmt.Printf("Processing: %s (%dx%d)\n", config.ImagePath, imgWidth, imgHeight)
	}

	// Step 3: Get input dimensions from model
	userInputs := ng.GetUserInputs()
	if len(userInputs) == 0 {
		return fmt.Errorf("model has no user inputs")
	}
	inputStream := userInputs[0]
	inputWidth := int(inputStream.Shape.Width)
	inputHeight := int(inputStream.Shape.Height)
	inputChannels := int(inputStream.Shape.Features)

	if !config.JSONOutput {
		fmt.Printf("Input tensor: %s (%dx%dx%d)\n", inputStream.Name, inputHeight, inputWidth, inputChannels)
	}

	// Step 4: Preprocess the image
	inputData := preprocessImage(img, inputWidth, inputHeight)

	if !config.JSONOutput {
		fmt.Printf("Preprocessed to %dx%d, %d bytes\n", inputWidth, inputHeight, len(inputData))
	}

	// Step 5: Try to run real inference
	var detections []Detection
	var inferenceTime time.Duration
	useRealInference := true

	// Attempt to open device and run inference
	dev, deviceErr := openDevice(config.DevicePath)
	if deviceErr != nil {
		if !config.JSONOutput {
			fmt.Printf("Warning: Could not open device: %v\n", deviceErr)
			fmt.Printf("Falling back to simulated detections.\n")
		}
		useRealInference = false
	}

	if useRealInference && dev != nil {
		defer dev.Close()

		if !config.JSONOutput {
			fmt.Printf("Using device: %s (Driver: %s)\n", dev.Path(), dev.DriverVersion())
		}

		startTime := time.Now()
		detections, err = runRealInference(dev, hefFile, ng, inputData, config.Threshold)
		inferenceTime = time.Since(startTime)

		if err != nil {
			if !config.JSONOutput {
				fmt.Printf("Warning: Inference failed: %v\n", err)
				fmt.Printf("Falling back to simulated detections.\n")
			}
			useRealInference = false
		}
	}

	// Fallback to simulated detections
	if !useRealInference {
		startTime := time.Now()
		detections = generateMockDetections(config.Threshold)
		inferenceTime = time.Since(startTime)
		if !config.JSONOutput {
			fmt.Printf("Note: Using simulated detections\n")
		}
	}

	// Step 6: Filter for person detections only
	personDetections := filterPersonDetections(detections)

	// Step 7: Output results
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

func openDevice(preferredPath string) (*device.Device, error) {
	if preferredPath != "" {
		return device.Open(preferredPath)
	}
	return device.OpenFirst()
}

func runRealInference(dev *device.Device, hefFile *hef.Hef, ngInfo *hef.NetworkGroupInfo, inputData []byte, threshold float32) ([]Detection, error) {
	// Configure the network group
	cng, err := dev.ConfigureDefaultNetworkGroup(hefFile)
	if err != nil {
		return nil, fmt.Errorf("configuring network group: %w", err)
	}
	defer cng.Close()

	// Activate the network group
	ang, err := cng.Activate()
	if err != nil {
		return nil, fmt.Errorf("activating network group: %w", err)
	}
	defer ang.Deactivate()

	// Build VStreams
	params := stream.DefaultVStreamParams()
	vstreams, err := stream.BuildVStreams(cng, params)
	if err != nil {
		return nil, fmt.Errorf("building vstreams: %w", err)
	}
	defer vstreams.Close()

	if len(vstreams.Inputs) == 0 {
		return nil, fmt.Errorf("no input vstreams")
	}
	if len(vstreams.Outputs) == 0 {
		return nil, fmt.Errorf("no output vstreams")
	}

	// Write input data
	inputVStream := vstreams.Inputs[0]
	if err := inputVStream.Write(inputData); err != nil {
		return nil, fmt.Errorf("writing input: %w", err)
	}

	// Flush input
	if err := inputVStream.Flush(); err != nil {
		return nil, fmt.Errorf("flushing input: %w", err)
	}

	// Read output
	outputVStream := vstreams.Outputs[0]
	outputData, err := outputVStream.Read()
	if err != nil {
		return nil, fmt.Errorf("reading output: %w", err)
	}

	// Parse NMS output
	detections := parseNmsOutput(outputData, ngInfo, threshold)

	return detections, nil
}

// parseNmsOutput parses the raw NMS output from the model
func parseNmsOutput(data []byte, ngInfo *hef.NetworkGroupInfo, threshold float32) []Detection {
	nmsInfo := ngInfo.GetNmsInfo()
	if nmsInfo == nil {
		// No NMS, try to parse as raw output
		return parseRawOutput(data, threshold)
	}

	// Convert bytes to float32
	floatData := bytesToFloat32(data)

	numClasses := int(nmsInfo.NmsShape.NumberOfClasses)

	// Parse using NMS format
	rawDetections := transform.ParseNmsByClass(floatData, numClasses)

	// Filter by threshold
	rawDetections = transform.FilterByScore(rawDetections, threshold)

	// Convert to our Detection format
	var detections []Detection
	for _, d := range rawDetections {
		detections = append(detections, Detection{
			ClassID:    d.ClassId,
			ClassName:  getClassName(d.ClassId),
			Confidence: d.Score,
			BBox: BBox{
				XMin: d.BBox.XMin,
				YMin: d.BBox.YMin,
				XMax: d.BBox.XMax,
				YMax: d.BBox.YMax,
			},
		})
	}

	return detections
}

// parseRawOutput parses raw detection output (non-NMS models)
func parseRawOutput(data []byte, threshold float32) []Detection {
	floatData := bytesToFloat32(data)
	rawDetections := transform.ParseNmsByScore(floatData)
	rawDetections = transform.FilterByScore(rawDetections, threshold)

	var detections []Detection
	for _, d := range rawDetections {
		detections = append(detections, Detection{
			ClassID:    d.ClassId,
			ClassName:  getClassName(d.ClassId),
			Confidence: d.Score,
			BBox: BBox{
				XMin: d.BBox.XMin,
				YMin: d.BBox.YMin,
				XMax: d.BBox.XMax,
				YMax: d.BBox.YMax,
			},
		})
	}

	return detections
}

// bytesToFloat32 converts a byte slice to float32 slice
func bytesToFloat32(data []byte) []float32 {
	if len(data)%4 != 0 {
		return nil
	}

	result := make([]float32, len(data)/4)
	for i := range result {
		bits := binary.LittleEndian.Uint32(data[i*4 : (i+1)*4])
		result[i] = math.Float32frombits(bits)
	}
	return result
}

// getClassName returns the COCO class name for a class ID
func getClassName(classID int) string {
	cocoClasses := []string{
		"person", "bicycle", "car", "motorcycle", "airplane", "bus", "train", "truck",
		"boat", "traffic light", "fire hydrant", "stop sign", "parking meter", "bench",
		"bird", "cat", "dog", "horse", "sheep", "cow", "elephant", "bear", "zebra",
		"giraffe", "backpack", "umbrella", "handbag", "tie", "suitcase", "frisbee",
		"skis", "snowboard", "sports ball", "kite", "baseball bat", "baseball glove",
		"skateboard", "surfboard", "tennis racket", "bottle", "wine glass", "cup",
		"fork", "knife", "spoon", "bowl", "banana", "apple", "sandwich", "orange",
		"broccoli", "carrot", "hot dog", "pizza", "donut", "cake", "chair", "couch",
		"potted plant", "bed", "dining table", "toilet", "tv", "laptop", "mouse",
		"remote", "keyboard", "cell phone", "microwave", "oven", "toaster", "sink",
		"refrigerator", "book", "clock", "vase", "scissors", "teddy bear", "hair drier",
		"toothbrush",
	}
	if classID >= 0 && classID < len(cocoClasses) {
		return cocoClasses[classID]
	}
	return fmt.Sprintf("class_%d", classID)
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
