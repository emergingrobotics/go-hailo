package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/anthropics/purple-hailo/pkg/control"
	"github.com/anthropics/purple-hailo/pkg/driver"
	"github.com/anthropics/purple-hailo/pkg/hef"
)

func main() {
	hefPath := flag.String("hef", "", "Path to HEF file")
	devicePath := flag.String("device", "/dev/hailo0", "Device path")
	flag.Parse()

	if *hefPath == "" {
		fmt.Println("Usage: activation_test -hef <path-to-hef> [-device /dev/hailo0]")
		os.Exit(1)
	}

	// Parse HEF file
	fmt.Println("=== Parsing HEF ===")
	h, err := hef.Parse(*hefPath)
	if err != nil {
		log.Fatalf("Failed to parse HEF: %v", err)
	}
	fmt.Printf("HEF: %s\n", *hefPath)
	fmt.Printf("Architecture: %s\n", h.DeviceArch)

	ng, err := h.GetDefaultNetworkGroup()
	if err != nil {
		log.Fatalf("Failed to get network group: %v", err)
	}
	fmt.Printf("Network Group: %s\n", ng.Name)
	fmt.Printf("Contexts: %d\n", len(ng.Contexts))

	// Show action statistics
	fmt.Println("\n=== HEF Context Actions ===")
	for i, ctx := range ng.Contexts {
		totalActions := 0
		for _, op := range ctx.Operations {
			totalActions += len(op.Actions)
		}
		fmt.Printf("Context %d: %d operations, %d total actions\n", i, len(ctx.Operations), totalActions)
	}

	// Open device
	fmt.Println("\n=== Opening Device ===")
	device, err := driver.OpenDevice(*devicePath)
	if err != nil {
		log.Fatalf("Failed to open device: %v", err)
	}
	defer device.Close()
	fmt.Printf("Device: %s\n", *devicePath)

	// Test firmware communication
	fmt.Println("\n=== Testing Firmware Communication ===")
	var seq uint32 = 0

	seq++
	if err := control.Identify(device, seq); err != nil {
		log.Printf("IDENTIFY failed: %v", err)
	} else {
		fmt.Println("IDENTIFY (APP CPU): OK")
	}

	seq++
	if err := control.IdentifyCore(device, seq); err != nil {
		log.Printf("CORE_IDENTIFY failed: %v", err)
	} else {
		fmt.Println("CORE_IDENTIFY: OK")
	}

	// Clear configured apps
	fmt.Println("\n=== Clearing Configured Apps ===")
	seq++
	if err := control.ClearConfiguredApps(device, seq); err != nil {
		fmt.Printf("ClearConfiguredApps: FAILED (%v) - continuing anyway\n", err)
	} else {
		fmt.Println("ClearConfiguredApps: OK")
	}

	// Send network group header
	fmt.Println("\n=== Setting Network Group Header ===")
	dynamicContexts := uint16(len(ng.Contexts))
	if dynamicContexts == 0 {
		dynamicContexts = 1
	}
	appHeader := control.CreateDefaultApplicationHeader(dynamicContexts)
	fmt.Printf("Dynamic contexts: %d\n", dynamicContexts)

	seq++
	if err := control.SetNetworkGroupHeader(device, seq, appHeader); err != nil {
		log.Fatalf("SetNetworkGroupHeader failed: %v", err)
	}
	fmt.Println("SetNetworkGroupHeader: OK")

	// Test context info with empty data (this will fail but shows the error)
	fmt.Println("\n=== Testing Context Info (empty - expected to fail) ===")
	seq++
	chunk := &control.ContextInfoChunk{
		IsFirstChunk: true,
		IsLastChunk:  true,
		ContextType:  control.ContextTypeActivation,
		Data:         []byte{},
	}
	if err := control.SetContextInfo(device, seq, chunk); err != nil {
		fmt.Printf("SetContextInfo (empty activation): FAILED (expected)\n")
		fmt.Printf("  Error: %v\n", err)
		fmt.Println("\n  NOTE: Empty contexts are not accepted by firmware.")
		fmt.Println("  Full inference requires serializing action lists from HEF.")
	} else {
		fmt.Println("SetContextInfo: OK (unexpected!)")
	}

	// Summary
	fmt.Println("\n=== Summary ===")
	fmt.Println("Working:")
	fmt.Println("  - Device opening and querying")
	fmt.Println("  - IDENTIFY command (APP CPU)")
	fmt.Println("  - CORE_IDENTIFY command (Core CPU)")
	fmt.Println("  - ClearConfiguredApps")
	fmt.Println("  - SetNetworkGroupHeader (v4.20.0 format)")
	fmt.Println("")
	fmt.Println("Needs Implementation:")
	fmt.Println("  - SetContextInfo with action list data from HEF")
	fmt.Println("  - Action list serialization (CONTEXT_SWITCH_DEFS__ format)")
	fmt.Println("  - EnableCoreOp (requires valid context info first)")
	fmt.Println("  - VDMA channel setup for data transfer")
	fmt.Println("")
	fmt.Println("The HEF contains the action data, but it needs to be serialized")
	fmt.Println("into the binary format expected by the firmware.")
}
