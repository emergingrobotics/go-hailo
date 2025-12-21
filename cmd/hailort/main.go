package main

import (
	"fmt"
	"os"

	"github.com/anthropics/purple-hailo/pkg/driver"
)

// Version information (set by ldflags)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GoVersion = "unknown"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "scan":
		scanDevices()
	case "info":
		if len(args) < 1 {
			fmt.Println("Usage: hailort info <device>")
			os.Exit(1)
		}
		deviceInfo(args[0])
	case "debug":
		printDebugInfo()
	case "version":
		printVersion()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Hailo Runtime CLI")
	fmt.Println()
	fmt.Println("Usage: hailort <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  scan              Scan for Hailo devices")
	fmt.Println("  info <device>     Show device information")
	fmt.Println("  debug             Print IOCTL debug information")
	fmt.Println("  version           Print version information")
	fmt.Println("  help              Show this help")
}

func printDebugInfo() {
	fmt.Println("IOCTL Debug Information")
	fmt.Println()
	fmt.Printf("Expected Driver Version: %d.%d.%d\n",
		driver.HailoDrvVerMajor, driver.HailoDrvVerMinor, driver.HailoDrvVerRevision)
	fmt.Println()
	fmt.Println("Struct Sizes:")
	fmt.Printf("  DeviceProperties:     %d bytes\n", driver.SizeOfDeviceProperties)
	fmt.Printf("  DriverInfo:           %d bytes\n", driver.SizeOfDriverInfo)
	fmt.Printf("  VdmaBufferMapParams:  %d bytes\n", driver.SizeOfVdmaBufferMapParams)
	fmt.Printf("  DescListCreateParams: %d bytes\n", driver.SizeOfDescListCreateParams)
	fmt.Println()
	fmt.Println("IOCTL Command Codes:")
	fmt.Printf("  QueryDeviceProperties: 0x%08x\n", driver.GetIoctlQueryDeviceProperties())
	fmt.Printf("  QueryDriverInfo:       0x%08x\n", driver.GetIoctlQueryDriverInfo())
}

func printVersion() {
	fmt.Printf("hailort version %s\n", Version)
	fmt.Printf("  Build time: %s\n", BuildTime)
	fmt.Printf("  Go version: %s\n", GoVersion)
}

func scanDevices() {
	devices, err := driver.ScanDevices()
	if err != nil {
		fmt.Printf("Error scanning devices: %v\n", err)
		os.Exit(1)
	}

	if len(devices) == 0 {
		fmt.Println("No Hailo devices found")
		return
	}

	fmt.Printf("Found %d Hailo device(s):\n", len(devices))
	for i, dev := range devices {
		fmt.Printf("  [%d] %s\n", i, dev)
	}
}

func deviceInfo(devicePath string) {
	dev, err := driver.OpenDevice(devicePath)
	if err != nil {
		fmt.Printf("Error opening device %s: %v\n", devicePath, err)
		os.Exit(1)
	}
	defer dev.Close()

	props, err := dev.QueryDeviceProperties()
	if err != nil {
		fmt.Printf("Error querying device properties: %v\n", err)
		os.Exit(1)
	}

	info, err := dev.QueryDriverInfo()
	if err != nil {
		fmt.Printf("Error querying driver info: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Device: %s\n", devicePath)
	fmt.Printf("  Board Type: %d\n", props.BoardType)
	fmt.Printf("  DMA Type: %d\n", props.DmaType)
	fmt.Printf("  DMA Engines: %d\n", props.DmaEnginesCount)
	fmt.Printf("  Max Page Size: %d\n", props.DescMaxPageSize)
	fmt.Printf("  Firmware Loaded: %v\n", props.IsFwLoaded)
	fmt.Printf("  Driver Version: %d.%d.%d\n", info.MajorVersion, info.MinorVersion, info.RevisionVersion)
}
