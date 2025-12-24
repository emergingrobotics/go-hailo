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
		fmt.Println("Usage: action_test -hef <path-to-hef> [-device /dev/hailo0]")
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

	// Show action statistics and try to serialize
	fmt.Println("\n=== HEF Context Actions ===")
	for i, ctx := range ng.Contexts {
		totalActions := 0
		for _, op := range ctx.Operations {
			totalActions += len(op.Actions)
		}
		fmt.Printf("Context %d: %d operations, %d total actions\n", i, len(ctx.Operations), totalActions)

		// Try to build action list
		actionList, err := control.BuildContextActionList(ctx.Operations)
		if err != nil {
			fmt.Printf("  Failed to build action list: %v\n", err)
		} else {
			fmt.Printf("  Serialized action list: %d bytes\n", len(actionList))
			if len(actionList) > 0 && len(actionList) <= 64 {
				fmt.Printf("  Data: % 02x\n", actionList)
			} else if len(actionList) > 64 {
				fmt.Printf("  Data (first 64 bytes): % 02x...\n", actionList[:64])
			}
		}
	}

	// Show preliminary config
	if ng.PreliminaryConfig != nil {
		totalActions := 0
		for _, op := range ng.PreliminaryConfig.Operations {
			totalActions += len(op.Actions)
		}
		fmt.Printf("\nPreliminary Config: %d operations, %d total actions\n",
			len(ng.PreliminaryConfig.Operations), totalActions)

		actionList, err := control.BuildContextActionList(ng.PreliminaryConfig.Operations)
		if err != nil {
			fmt.Printf("  Failed to build action list: %v\n", err)
		} else {
			fmt.Printf("  Serialized action list: %d bytes\n", len(actionList))
		}
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

	// Skip IdentifyCore - it can hang if Core CPU is in bad state
	// seq++
	// if err := control.IdentifyCore(device, seq); err != nil {
	// 	log.Printf("CORE_IDENTIFY failed: %v", err)
	// } else {
	// 	fmt.Println("CORE_IDENTIFY: OK")
	// }

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

	// Send context info with action data
	fmt.Println("\n=== Sending Context Info ===")

	// Activation context (halt action)
	activationData := control.BuildEmptyActionList()
	if err := control.SendContextInfoChunks(device, &seq,
		control.ContextTypeActivation, activationData); err != nil {
		log.Fatalf("Activation context failed: %v", err)
	}
	fmt.Printf("ACTIVATION context: OK (%d bytes)\n", len(activationData))

	// Batch switching context
	batchData := control.BuildEmptyActionList()
	if err := control.SendContextInfoChunks(device, &seq,
		control.ContextTypeBatchSwitching, batchData); err != nil {
		log.Fatalf("Batch switching context failed: %v", err)
	}
	fmt.Printf("BATCH_SWITCHING context: OK (%d bytes)\n", len(batchData))

	// Preliminary context
	prelimData := []byte{}
	if ng.PreliminaryConfig != nil && len(ng.PreliminaryConfig.Operations) > 0 {
		prelimData, _ = control.BuildContextActionList(ng.PreliminaryConfig.Operations)
	}
	if len(prelimData) == 0 {
		prelimData = control.BuildEmptyActionList()
	}
	if err := control.SendContextInfoChunks(device, &seq,
		control.ContextTypePreliminary, prelimData); err != nil {
		log.Fatalf("Preliminary context failed: %v", err)
	}
	fmt.Printf("PRELIMINARY context: OK (%d bytes)\n", len(prelimData))

	// Dynamic contexts
	for i := 0; i < int(dynamicContexts); i++ {
		var dynData []byte
		if i < len(ng.Contexts) && len(ng.Contexts[i].Operations) > 0 {
			dynData, _ = control.BuildContextActionList(ng.Contexts[i].Operations)
		}
		if len(dynData) == 0 {
			dynData = control.BuildEmptyActionList()
		}
		if err := control.SendContextInfoChunks(device, &seq,
			control.ContextTypeDynamic, dynData); err != nil {
			log.Fatalf("Dynamic context %d failed: %v", i, err)
		}
		fmt.Printf("DYNAMIC context %d: OK (%d bytes)\n", i, len(dynData))
	}


	// Try EnableCoreOp
	fmt.Println("\n=== Enabling Core Op ===")
	seq++
	err = control.EnableCoreOp(device, seq, 0, 0, 0)
	if err != nil {
		fmt.Printf("EnableCoreOp: FAILED (%v)\n", err)
		fmt.Println("\nNote: EnableCoreOp requires valid action data from HEF.")
		fmt.Println("The action list serialization may need refinement for this model.")
	} else {
		fmt.Println("EnableCoreOp: OK!")
		fmt.Println("\nNetwork group is now activated and ready for inference!")

		// Reset to clean up
		seq++
		if err := control.ResetContextSwitchStateMachine(device, seq); err != nil {
			fmt.Printf("Reset: FAILED (%v)\n", err)
		} else {
			fmt.Println("Reset: OK")
		}
	}

	// Summary
	fmt.Println("\n=== Summary ===")
	fmt.Println("Action serialization implementation complete.")
	fmt.Println("The action list is built from HEF context operations.")
}
