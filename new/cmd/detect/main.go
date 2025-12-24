// Command detect is a CLI tool that detects people in images using a Hailo-8 accelerator.
//
// Usage:
//
//	detect -model <path-to-hef> -image <path-to-image>
package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"time"

	"github.com/anthropics/purple-hailo/new/src/go/hailo"
	"golang.org/x/image/draw"
)

func main() {
	modelPath := flag.String("model", "", "Path to HEF model file")
	imagePath := flag.String("image", "", "Path to input image (JPEG or PNG)")
	flag.Parse()

	if *modelPath == "" || *imagePath == "" {
		fmt.Println("Usage: detect -model <path-to-hef> -image <path-to-image>")
		os.Exit(1)
	}

	// Create inference engine
	fmt.Printf("Loading model: %s\n", *modelPath)
	startLoad := time.Now()

	inference, err := hailo.NewInference(*modelPath)
	if err != nil {
		log.Fatalf("Failed to create inference: %v", err)
	}
	defer inference.Close()

	loadTime := time.Since(startLoad)
	fmt.Printf("Model loaded in %v\n", loadTime)

	// Get input requirements
	inputInfo := inference.GetInputInfo()
	fmt.Printf("Input: %dx%dx%d (%d bytes)\n",
		inputInfo.Width, inputInfo.Height, inputInfo.Channels, inputInfo.FrameSize)

	// Load and preprocess image
	fmt.Printf("Loading image: %s\n", *imagePath)
	imageData, err := loadAndPreprocess(*imagePath, inputInfo)
	if err != nil {
		log.Fatalf("Failed to load image: %v", err)
	}

	// Run inference
	fmt.Println("Running inference...")
	startInfer := time.Now()

	count, err := inference.DetectPeople(imageData)
	if err != nil {
		log.Fatalf("Inference failed: %v", err)
	}

	inferTime := time.Since(startInfer)
	fmt.Printf("Inference completed in %v\n", inferTime)

	// Print result
	fmt.Printf("\nPeople detected: %d\n", count)

	// Also get all detections for more detail
	detections, err := inference.Detect(imageData)
	if err == nil && len(detections) > 0 {
		fmt.Printf("\nAll detections (%d):\n", len(detections))
		for i, det := range detections {
			className := getClassName(det.ClassID)
			fmt.Printf("  %d: %s (%.1f%%) at [%.1f, %.1f, %.1f, %.1f]\n",
				i+1, className, det.Confidence*100,
				det.XMin, det.YMin, det.XMax, det.YMax)
		}
	}
}

// loadAndPreprocess loads an image and converts it to RGB format for inference.
func loadAndPreprocess(path string, info hailo.InputInfo) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("cannot decode image: %w", err)
	}

	// Resize to model input size
	resized := image.NewRGBA(image.Rect(0, 0, info.Width, info.Height))
	draw.BiLinear.Scale(resized, resized.Bounds(), img, img.Bounds(), draw.Over, nil)

	// Convert RGBA to RGB
	rgb := make([]byte, info.Width*info.Height*3)
	for y := 0; y < info.Height; y++ {
		for x := 0; x < info.Width; x++ {
			offset := (y*info.Width + x) * 4
			rgbOffset := (y*info.Width + x) * 3
			rgb[rgbOffset] = resized.Pix[offset]     // R
			rgb[rgbOffset+1] = resized.Pix[offset+1] // G
			rgb[rgbOffset+2] = resized.Pix[offset+2] // B
		}
	}

	return rgb, nil
}

// getClassName returns the COCO class name for a class ID.
func getClassName(classID int) string {
	classes := []string{
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
	if classID >= 0 && classID < len(classes) {
		return classes[classID]
	}
	return fmt.Sprintf("class_%d", classID)
}
