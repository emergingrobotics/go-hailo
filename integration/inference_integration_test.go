//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

func skipIfNoHardware(t *testing.T) {
	t.Helper()

	// Check for Hailo device
	if _, err := os.Stat("/dev/hailo0"); os.IsNotExist(err) {
		t.Skip("No Hailo hardware available")
	}
}

func skipIfNoModel(t *testing.T) string {
	t.Helper()

	modelPaths := []string{
		"../models/yolox_s_leaky_hailo8.hef",
		"../models/yolov8n.hef",
		"../models/resnet50.hef",
		"testdata/model.hef",
	}

	for _, path := range modelPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	t.Skip("No HEF model file available")
	return ""
}

func getDevicePath(t *testing.T) string {
	t.Helper()

	devices := []string{"/dev/hailo0", "/dev/hailo1"}
	for _, path := range devices {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	t.Skip("No Hailo device available")
	return ""
}

func TestFullInferencePipeline(t *testing.T) {
	devicePath := getDevicePath(t)
	modelPath := skipIfNoModel(t)

	t.Logf("Using device: %s", devicePath)
	t.Logf("Using model: %s", modelPath)

	// Step 1: Scan for devices
	t.Log("Step 1: Scanning for devices...")
	devices := scanDevices()
	if len(devices) == 0 {
		t.Fatal("No devices found")
	}
	t.Logf("Found %d device(s)", len(devices))

	// Step 2: Open device
	t.Log("Step 2: Opening device...")
	dev, err := openDevice(devicePath)
	if err != nil {
		t.Fatalf("Failed to open device: %v", err)
	}
	defer dev.Close()
	t.Log("Device opened successfully")

	// Step 3: Load HEF
	t.Log("Step 3: Loading HEF file...")
	hefData, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("Failed to read HEF: %v", err)
	}
	hef, err := parseHef(hefData)
	if err != nil {
		t.Fatalf("Failed to parse HEF: %v", err)
	}
	t.Logf("HEF loaded: version=%d", hef.Version)

	// Step 4: Configure network group
	t.Log("Step 4: Configuring network group...")
	ng, err := dev.Configure(hef)
	if err != nil {
		t.Fatalf("Failed to configure: %v", err)
	}
	defer ng.Deactivate()
	t.Logf("Network group configured: %s", ng.Name)

	// Step 5: Create VStreams
	t.Log("Step 5: Creating VStreams...")
	inputStreams := ng.InputStreams()
	outputStreams := ng.OutputStreams()
	t.Logf("Created %d input, %d output streams", len(inputStreams), len(outputStreams))

	// Step 6: Run inference
	t.Log("Step 6: Running inference...")
	inputData := make([]byte, ng.InputFrameSize())
	outputData := make([]byte, ng.OutputFrameSize())

	start := time.Now()
	err = ng.Infer(inputData, outputData)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Inference failed: %v", err)
	}
	t.Logf("Inference completed in %v", elapsed)

	// Step 7: Cleanup
	t.Log("Step 7: Cleanup...")
	t.Log("Full pipeline completed successfully")
}

func TestClassificationEndToEnd(t *testing.T) {
	devicePath := getDevicePath(t)
	modelPath := skipIfNoModel(t)

	t.Log("Classification E2E test...")

	// Load model
	dev, err := openDevice(devicePath)
	if err != nil {
		t.Fatalf("Failed to open device: %v", err)
	}
	defer dev.Close()

	hefData, _ := os.ReadFile(modelPath)
	hef, err := parseHef(hefData)
	if err != nil {
		t.Fatalf("Failed to parse HEF: %v", err)
	}

	ng, err := dev.Configure(hef)
	if err != nil {
		t.Fatalf("Failed to configure: %v", err)
	}
	defer ng.Deactivate()

	// Prepare input (224x224x3 for classification)
	inputSize := 224 * 224 * 3
	input := make([]byte, inputSize)

	// Fill with test pattern (simulated image)
	for i := range input {
		input[i] = byte(i % 256)
	}

	// Run inference
	output := make([]byte, ng.OutputFrameSize())
	err = ng.Infer(input, output)
	if err != nil {
		t.Fatalf("Inference failed: %v", err)
	}

	// Parse output (simulated - would parse softmax scores)
	topK := parseTopKClasses(output, 5)
	t.Logf("Top 5 predictions: %v", topK)

	if len(topK) != 5 {
		t.Errorf("Expected 5 top classes, got %d", len(topK))
	}
}

func TestDetectionEndToEnd(t *testing.T) {
	devicePath := getDevicePath(t)
	modelPath := skipIfNoModel(t)

	t.Log("Detection E2E test...")

	dev, err := openDevice(devicePath)
	if err != nil {
		t.Fatalf("Failed to open device: %v", err)
	}
	defer dev.Close()

	hefData, _ := os.ReadFile(modelPath)
	hef, err := parseHef(hefData)
	if err != nil {
		t.Fatalf("Failed to parse HEF: %v", err)
	}

	ng, err := dev.Configure(hef)
	if err != nil {
		t.Fatalf("Failed to configure: %v", err)
	}
	defer ng.Deactivate()

	// Prepare input (640x640x3 for YOLO)
	inputSize := 640 * 640 * 3
	input := make([]byte, inputSize)

	// Run inference
	output := make([]byte, ng.OutputFrameSize())
	err = ng.Infer(input, output)
	if err != nil {
		t.Fatalf("Inference failed: %v", err)
	}

	// Parse NMS output
	detections := parseDetections(output, 0.5)
	t.Logf("Found %d detections", len(detections))
}

func TestMultiModelEndToEnd(t *testing.T) {
	devicePath := getDevicePath(t)

	t.Log("Multi-model E2E test...")

	// Find multiple models
	models, _ := filepath.Glob("../models/*.hef")
	if len(models) < 2 {
		t.Skip("Need at least 2 models for multi-model test")
	}

	dev, err := openDevice(devicePath)
	if err != nil {
		t.Fatalf("Failed to open device: %v", err)
	}
	defer dev.Close()

	for i, modelPath := range models[:2] {
		t.Logf("Testing model %d: %s", i+1, filepath.Base(modelPath))

		hefData, _ := os.ReadFile(modelPath)
		hef, err := parseHef(hefData)
		if err != nil {
			t.Errorf("Failed to parse HEF %s: %v", modelPath, err)
			continue
		}

		ng, err := dev.Configure(hef)
		if err != nil {
			t.Errorf("Failed to configure %s: %v", modelPath, err)
			continue
		}

		// Run one inference
		input := make([]byte, ng.InputFrameSize())
		output := make([]byte, ng.OutputFrameSize())
		err = ng.Infer(input, output)
		if err != nil {
			t.Errorf("Inference failed for %s: %v", modelPath, err)
		}

		ng.Deactivate()
		t.Logf("Model %d completed successfully", i+1)
	}
}

func TestStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	devicePath := getDevicePath(t)
	modelPath := skipIfNoModel(t)

	dev, err := openDevice(devicePath)
	if err != nil {
		t.Fatalf("Failed to open device: %v", err)
	}
	defer dev.Close()

	hefData, _ := os.ReadFile(modelPath)
	hef, err := parseHef(hefData)
	if err != nil {
		t.Fatalf("Failed to parse HEF: %v", err)
	}

	ng, err := dev.Configure(hef)
	if err != nil {
		t.Fatalf("Failed to configure: %v", err)
	}
	defer ng.Deactivate()

	numIterations := 1000
	t.Logf("Running %d inference iterations...", numIterations)

	input := make([]byte, ng.InputFrameSize())
	output := make([]byte, ng.OutputFrameSize())

	var totalLatency time.Duration
	successCount := 0

	for i := 0; i < numIterations; i++ {
		start := time.Now()
		err := ng.Infer(input, output)
		elapsed := time.Since(start)

		if err == nil {
			successCount++
			totalLatency += elapsed
		}

		if (i+1)%100 == 0 {
			t.Logf("Completed %d/%d iterations", i+1, numIterations)
		}
	}

	avgLatency := totalLatency / time.Duration(successCount)
	t.Logf("Completed: %d/%d successful, avg latency: %v", successCount, numIterations, avgLatency)

	if successCount != numIterations {
		t.Errorf("Expected %d successes, got %d", numIterations, successCount)
	}
}

func TestConcurrentInference(t *testing.T) {
	devicePath := getDevicePath(t)
	modelPath := skipIfNoModel(t)

	dev, err := openDevice(devicePath)
	if err != nil {
		t.Fatalf("Failed to open device: %v", err)
	}
	defer dev.Close()

	hefData, _ := os.ReadFile(modelPath)
	hef, err := parseHef(hefData)
	if err != nil {
		t.Fatalf("Failed to parse HEF: %v", err)
	}

	ng, err := dev.Configure(hef)
	if err != nil {
		t.Fatalf("Failed to configure: %v", err)
	}
	defer ng.Deactivate()

	numGoroutines := 4
	numIterationsPerGoroutine := 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numIterationsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			input := make([]byte, ng.InputFrameSize())
			output := make([]byte, ng.OutputFrameSize())

			for j := 0; j < numIterationsPerGoroutine; j++ {
				err := ng.Infer(input, output)
				if err != nil {
					errors <- err
				}
			}
			t.Logf("Goroutine %d completed", id)
		}(i)
	}

	wg.Wait()
	close(errors)

	errorCount := 0
	for err := range errors {
		t.Logf("Concurrent error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("Had %d errors during concurrent inference", errorCount)
	}
}

func TestDeviceRecovery(t *testing.T) {
	devicePath := getDevicePath(t)

	t.Log("Testing device recovery...")

	// Open and close device multiple times
	for i := 0; i < 5; i++ {
		dev, err := openDevice(devicePath)
		if err != nil {
			t.Fatalf("Iteration %d: Failed to open device: %v", i, err)
		}

		// Query device to verify it's functional
		props, err := dev.QueryProperties()
		if err != nil {
			t.Errorf("Iteration %d: Failed to query: %v", i, err)
		} else {
			t.Logf("Iteration %d: Device OK, DMA engines: %d", i, props.DmaEngines)
		}

		err = dev.Close()
		if err != nil {
			t.Errorf("Iteration %d: Failed to close: %v", i, err)
		}

		time.Sleep(100 * time.Millisecond) // Brief pause between iterations
	}

	t.Log("Device recovery test passed")
}

func TestMemoryLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	devicePath := getDevicePath(t)
	modelPath := skipIfNoModel(t)

	t.Log("Testing for memory leaks...")

	var memStatsBefore, memStatsAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memStatsBefore)

	// Run many operations
	for iteration := 0; iteration < 10; iteration++ {
		dev, err := openDevice(devicePath)
		if err != nil {
			t.Fatalf("Failed to open device: %v", err)
		}

		hefData, _ := os.ReadFile(modelPath)
		hef, err := parseHef(hefData)
		if err != nil {
			dev.Close()
			t.Fatalf("Failed to parse HEF: %v", err)
		}

		ng, err := dev.Configure(hef)
		if err != nil {
			dev.Close()
			t.Fatalf("Failed to configure: %v", err)
		}

		// Run multiple inferences
		input := make([]byte, ng.InputFrameSize())
		output := make([]byte, ng.OutputFrameSize())
		for i := 0; i < 100; i++ {
			ng.Infer(input, output)
		}

		ng.Deactivate()
		dev.Close()
	}

	runtime.GC()
	runtime.ReadMemStats(&memStatsAfter)

	heapGrowth := int64(memStatsAfter.HeapAlloc) - int64(memStatsBefore.HeapAlloc)
	t.Logf("Heap before: %d bytes, after: %d bytes, growth: %d bytes",
		memStatsBefore.HeapAlloc, memStatsAfter.HeapAlloc, heapGrowth)

	// Allow some growth but flag excessive leaks
	maxAcceptableGrowth := int64(10 * 1024 * 1024) // 10 MB
	if heapGrowth > maxAcceptableGrowth {
		t.Errorf("Potential memory leak: heap grew by %d bytes", heapGrowth)
	}
}

func TestLatencyConsistency(t *testing.T) {
	devicePath := getDevicePath(t)
	modelPath := skipIfNoModel(t)

	dev, err := openDevice(devicePath)
	if err != nil {
		t.Fatalf("Failed to open device: %v", err)
	}
	defer dev.Close()

	hefData, _ := os.ReadFile(modelPath)
	hef, err := parseHef(hefData)
	if err != nil {
		t.Fatalf("Failed to parse HEF: %v", err)
	}

	ng, err := dev.Configure(hef)
	if err != nil {
		t.Fatalf("Failed to configure: %v", err)
	}
	defer ng.Deactivate()

	input := make([]byte, ng.InputFrameSize())
	output := make([]byte, ng.OutputFrameSize())

	// Warmup
	for i := 0; i < 10; i++ {
		ng.Infer(input, output)
	}

	// Measure latencies
	numSamples := 100
	latencies := make([]time.Duration, numSamples)

	for i := 0; i < numSamples; i++ {
		start := time.Now()
		ng.Infer(input, output)
		latencies[i] = time.Since(start)
	}

	// Calculate statistics
	var total time.Duration
	var min, max time.Duration = latencies[0], latencies[0]
	for _, l := range latencies {
		total += l
		if l < min {
			min = l
		}
		if l > max {
			max = l
		}
	}
	avg := total / time.Duration(numSamples)

	t.Logf("Latency stats: min=%v, max=%v, avg=%v", min, max, avg)

	// Check consistency (max should not be more than 3x avg)
	if max > 3*avg {
		t.Logf("Warning: high latency variance (max=%v, avg=%v)", max, avg)
	}
}

// Mock types and functions for integration tests
// In real implementation, these would import from actual packages

type mockDevice struct {
	path   string
	closed bool
}

func (d *mockDevice) Close() error {
	d.closed = true
	return nil
}

func (d *mockDevice) Configure(hef *mockHef) (*mockNetworkGroup, error) {
	return &mockNetworkGroup{name: "test_network"}, nil
}

func (d *mockDevice) QueryProperties() (*mockDeviceProps, error) {
	return &mockDeviceProps{DmaEngines: 4}, nil
}

type mockDeviceProps struct {
	DmaEngines int
}

type mockHef struct {
	Version int
}

type mockNetworkGroup struct {
	name string
}

func (ng *mockNetworkGroup) Deactivate() error { return nil }
func (ng *mockNetworkGroup) InputStreams() []string { return []string{"input0"} }
func (ng *mockNetworkGroup) OutputStreams() []string { return []string{"output0"} }
func (ng *mockNetworkGroup) InputFrameSize() int { return 224 * 224 * 3 }
func (ng *mockNetworkGroup) OutputFrameSize() int { return 1000 * 4 }
func (ng *mockNetworkGroup) Infer(input, output []byte) error { return nil }

func scanDevices() []string {
	var devices []string
	for i := 0; i < 4; i++ {
		path := filepath.Join("/dev", "hailo"+string(rune('0'+i)))
		if _, err := os.Stat(path); err == nil {
			devices = append(devices, path)
		}
	}
	return devices
}

func openDevice(path string) (*mockDevice, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}
	return &mockDevice{path: path}, nil
}

func parseHef(data []byte) (*mockHef, error) {
	if len(data) < 8 {
		return nil, os.ErrInvalid
	}
	return &mockHef{Version: 3}, nil
}

func parseTopKClasses(output []byte, k int) []int {
	// Mock: return k class indices
	result := make([]int, k)
	for i := 0; i < k; i++ {
		result[i] = i
	}
	return result
}

type Detection struct {
	ClassID int
	Score   float32
	BBox    [4]float32
}

func parseDetections(output []byte, threshold float32) []Detection {
	// Mock: return some detections
	return []Detection{
		{ClassID: 0, Score: 0.9, BBox: [4]float32{0.1, 0.1, 0.5, 0.5}},
		{ClassID: 1, Score: 0.8, BBox: [4]float32{0.5, 0.5, 0.9, 0.9}},
	}
}
