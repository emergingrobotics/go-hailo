package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/anthropics/purple-hailo/pkg/control"
	"github.com/anthropics/purple-hailo/pkg/driver"
)

func main() {
	devicePath := flag.String("device", "/dev/hailo0", "Device path")
	flag.Parse()

	device, err := driver.OpenDevice(*devicePath)
	if err != nil {
		log.Fatalf("Failed to open device: %v", err)
	}
	defer device.Close()

	var seq uint32 = 1

	// First try NN Core reset (recovers stuck core)
	fmt.Println("Sending NN Core reset...")
	err = control.ResetNNCore(device, seq)
	if err != nil {
		fmt.Printf("ResetNNCore failed: %v\n", err)
	} else {
		fmt.Println("ResetNNCore: OK")
	}

	time.Sleep(2 * time.Second)

	// Then try soft reset
	seq++
	fmt.Println("Sending Soft reset...")
	err = control.SoftReset(device, seq)
	if err != nil {
		fmt.Printf("SoftReset failed: %v\n", err)
	} else {
		fmt.Println("SoftReset: OK")
	}

	time.Sleep(2 * time.Second)

	// Check if APP CPU responds
	seq++
	err = control.Identify(device, seq)
	if err != nil {
		fmt.Printf("Identify (APP CPU): FAILED (%v)\n", err)
	} else {
		fmt.Println("Identify (APP CPU): OK")
	}

	// Check if Core CPU responds
	seq++
	err = control.IdentifyCore(device, seq)
	if err != nil {
		fmt.Printf("IdentifyCore: FAILED (%v)\n", err)
	} else {
		fmt.Println("IdentifyCore: OK")
	}
}
