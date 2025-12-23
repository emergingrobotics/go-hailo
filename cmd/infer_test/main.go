package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/anthropics/purple-hailo/pkg/control"
	"github.com/anthropics/purple-hailo/pkg/device"
	"github.com/anthropics/purple-hailo/pkg/driver"
	"github.com/anthropics/purple-hailo/pkg/hef"
	"github.com/anthropics/purple-hailo/pkg/stream"
)

func main() {
	// Parse command line flags
	hefPath := flag.String("hef", "", "Path to HEF file")
	devicePath := flag.String("device", "/dev/hailo0", "Device path")
	verbose := flag.Bool("v", false, "Verbose output")
	flag.Parse()

	if *hefPath == "" {
		fmt.Println("Usage: infer_test -hef <path-to-hef> [-device /dev/hailo0] [-v]")
		os.Exit(1)
	}

	// Step 1: Parse HEF file
	fmt.Println("=== Step 1: Parsing HEF ===")
	h, err := hef.Parse(*hefPath)
	if err != nil {
		log.Fatalf("Failed to parse HEF: %v", err)
	}
	fmt.Printf("HEF Version: %d\n", h.Version)
	fmt.Printf("Device Architecture: %s\n", h.DeviceArch)
	fmt.Printf("Network Groups: %d\n", len(h.NetworkGroups))

	ng, err := h.GetDefaultNetworkGroup()
	if err != nil {
		log.Fatalf("Failed to get default network group: %v", err)
	}
	fmt.Printf("\nNetwork Group: %s\n", ng.Name)
	fmt.Printf("Input Streams: %d\n", len(ng.InputStreams))
	for i, s := range ng.InputStreams {
		fmt.Printf("  [%d] %s: %dx%dx%d (frame=%d bytes)\n",
			i, s.Name, s.Shape.Height, s.Shape.Width, s.Shape.Features,
			s.Shape.Height*s.Shape.Width*s.Shape.Features)
	}
	fmt.Printf("Output Streams: %d\n", len(ng.OutputStreams))
	for i, s := range ng.OutputStreams {
		fmt.Printf("  [%d] %s: %dx%dx%d (frame=%d bytes)\n",
			i, s.Name, s.Shape.Height, s.Shape.Width, s.Shape.Features,
			s.Shape.Height*s.Shape.Width*s.Shape.Features)
	}
	fmt.Printf("Contexts: %d\n", len(ng.Contexts))
	fmt.Printf("Bottleneck FPS: %.2f\n", ng.BottleneckFps)

	// Step 2: Open device
	fmt.Println("\n=== Step 2: Opening Device ===")
	dev, err := device.Open(*devicePath)
	if err != nil {
		log.Fatalf("Failed to open device: %v", err)
	}
	defer dev.Close()
	fmt.Printf("Device opened: %s\n", dev.Path())
	fmt.Printf("Driver version: %s\n", dev.DriverVersion())
	fmt.Printf("Firmware loaded: %v\n", dev.IsFirmwareLoaded())

	// Step 3: Test firmware communication
	fmt.Println("\n=== Step 3: Testing Firmware Communication ===")
	err = control.Identify(dev.DeviceFile(), 1)
	if err != nil {
		log.Printf("IDENTIFY (APP CPU) failed: %v", err)
	} else {
		fmt.Println("IDENTIFY (APP CPU): SUCCESS")
	}

	err = control.IdentifyCore(dev.DeviceFile(), 2)
	if err != nil {
		log.Printf("CORE_IDENTIFY failed: %v", err)
	} else {
		fmt.Println("CORE_IDENTIFY: SUCCESS")
	}

	// Step 4: Configure network group
	fmt.Println("\n=== Step 4: Configuring Network Group ===")
	cng, err := dev.ConfigureNetworkGroup(h, "")
	if err != nil {
		log.Fatalf("Failed to configure network group: %v", err)
	}
	fmt.Printf("Network group configured: %s\n", cng.Name())
	fmt.Printf("State: %d\n", cng.State())

	// Step 5: Activate network group
	fmt.Println("\n=== Step 5: Activating Network Group ===")
	ang, err := cng.Activate()
	if err != nil {
		log.Fatalf("Failed to activate network group: %v", err)
	}
	defer ang.Deactivate()
	fmt.Println("Network group activated!")

	// Step 6: Setup VDMA channels
	fmt.Println("\n=== Step 6: Setting Up VDMA Channels ===")

	// Get device properties for page size
	props, err := dev.DeviceFile().QueryDeviceProperties()
	if err != nil {
		log.Fatalf("Failed to query device properties: %v", err)
	}
	fmt.Printf("Page size: %d\n", props.DescMaxPageSize)

	// Create channels for inputs and outputs
	// Use engine 0, channel 0 for first input, channel 1 for first output
	inputChannel := stream.NewVdmaChannel(dev.DeviceFile(), 0, 0)
	outputChannel := stream.NewVdmaChannel(dev.DeviceFile(), 0, 1)

	// Enable channels
	fmt.Println("Enabling VDMA channels...")
	if err := inputChannel.Enable(false); err != nil {
		log.Fatalf("Failed to enable input channel: %v", err)
	}
	defer inputChannel.Disable()

	if err := outputChannel.Enable(false); err != nil {
		log.Fatalf("Failed to enable output channel: %v", err)
	}
	defer outputChannel.Disable()

	fmt.Println("VDMA channels enabled!")

	// Step 7: Allocate buffers
	fmt.Println("\n=== Step 7: Allocating Buffers ===")

	// Get first input/output stream info
	userInputs := cng.GetUserInputs()
	userOutputs := cng.GetUserOutputs()

	if len(userInputs) == 0 || len(userOutputs) == 0 {
		log.Fatalf("No user inputs or outputs found")
	}

	inputInfo := userInputs[0]
	outputInfo := userOutputs[0]

	fmt.Printf("Input: %s (frame size: %d)\n", inputInfo.Name, inputInfo.FrameSize)
	fmt.Printf("Output: %s (frame size: %d)\n", outputInfo.Name, outputInfo.FrameSize)

	// Allocate input buffer
	inputBuffer, err := stream.AllocateBuffer(dev.DeviceFile(), inputInfo.FrameSize, driver.DmaToDevice)
	if err != nil {
		log.Fatalf("Failed to allocate input buffer: %v", err)
	}
	defer inputBuffer.Close()
	fmt.Printf("Input buffer allocated: %d bytes, handle=%d\n", inputBuffer.Size(), inputBuffer.Handle())

	// Allocate output buffer
	outputBuffer, err := stream.AllocateBuffer(dev.DeviceFile(), outputInfo.FrameSize, driver.DmaFromDevice)
	if err != nil {
		log.Fatalf("Failed to allocate output buffer: %v", err)
	}
	defer outputBuffer.Close()
	fmt.Printf("Output buffer allocated: %d bytes, handle=%d\n", outputBuffer.Size(), outputBuffer.Handle())

	// Step 8: Create descriptor lists
	fmt.Println("\n=== Step 8: Creating Descriptor Lists ===")

	inputDescCount := stream.CalculateDescCount(inputInfo.FrameSize, props.DescMaxPageSize)
	outputDescCount := stream.CalculateDescCount(outputInfo.FrameSize, props.DescMaxPageSize)

	inputDescList, err := stream.CreateDescriptorList(dev.DeviceFile(), inputDescCount, props.DescMaxPageSize, false)
	if err != nil {
		log.Fatalf("Failed to create input descriptor list: %v", err)
	}
	defer inputDescList.Release()
	fmt.Printf("Input descriptor list: %d descriptors\n", inputDescList.DescCount())

	outputDescList, err := stream.CreateDescriptorList(dev.DeviceFile(), outputDescCount, props.DescMaxPageSize, false)
	if err != nil {
		log.Fatalf("Failed to create output descriptor list: %v", err)
	}
	defer outputDescList.Release()
	fmt.Printf("Output descriptor list: %d descriptors\n", outputDescList.DescCount())

	// Step 9: Prepare test data
	fmt.Println("\n=== Step 9: Preparing Test Data ===")

	// Fill input with test pattern
	inputData := inputBuffer.Data()
	for i := range inputData {
		inputData[i] = byte(i % 256)
	}
	fmt.Printf("Filled input buffer with test pattern (%d bytes)\n", len(inputData))

	// Sync input buffer to device
	if err := inputBuffer.SyncForDevice(); err != nil {
		log.Fatalf("Failed to sync input buffer: %v", err)
	}

	// Step 10: Program descriptors and launch transfer
	fmt.Println("\n=== Step 10: Programming Descriptors ===")

	// Program input descriptor list
	if err := inputDescList.Program(inputBuffer, inputChannel.ChannelIndex(), 0, true, driver.InterruptsDomainDevice); err != nil {
		log.Fatalf("Failed to program input descriptor list: %v", err)
	}
	fmt.Println("Input descriptors programmed")

	// Program output descriptor list
	if err := outputDescList.Program(outputBuffer, outputChannel.ChannelIndex(), 0, true, driver.InterruptsDomainHost); err != nil {
		log.Fatalf("Failed to program output descriptor list: %v", err)
	}
	fmt.Println("Output descriptors programmed")

	// Step 11: Launch transfers
	fmt.Println("\n=== Step 11: Launching Transfers ===")

	// Launch output transfer first (receive side ready)
	if err := outputChannel.LaunchTransfer(outputDescList, outputBuffer, 0, true, driver.InterruptsDomainNone, driver.InterruptsDomainHost); err != nil {
		log.Printf("Warning: Failed to launch output transfer: %v", err)
	} else {
		fmt.Println("Output transfer launched")
	}

	// Launch input transfer
	if err := inputChannel.LaunchTransfer(inputDescList, inputBuffer, 0, true, driver.InterruptsDomainNone, driver.InterruptsDomainDevice); err != nil {
		log.Printf("Warning: Failed to launch input transfer: %v", err)
	} else {
		fmt.Println("Input transfer launched")
	}

	// Step 12: Wait for completion
	fmt.Println("\n=== Step 12: Waiting for Completion ===")

	if *verbose {
		fmt.Println("Waiting for output interrupt...")
	}

	// Wait for output with timeout
	start := time.Now()
	err = outputChannel.WaitForInterruptWithTimeout(5 * time.Second)
	elapsed := time.Since(start)

	if err != nil {
		fmt.Printf("Output transfer completed with error: %v (took %v)\n", err, elapsed)
	} else {
		fmt.Printf("Output transfer completed successfully (took %v)\n", elapsed)

		// Sync output buffer from device
		if err := outputBuffer.SyncForCPU(); err != nil {
			log.Printf("Warning: Failed to sync output buffer: %v", err)
		}

		// Check output
		outputData := outputBuffer.Data()
		nonZero := 0
		for _, b := range outputData {
			if b != 0 {
				nonZero++
			}
		}
		fmt.Printf("Output data: %d non-zero bytes out of %d\n", nonZero, len(outputData))
	}

	fmt.Println("\n=== Test Complete ===")
}
