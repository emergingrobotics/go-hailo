package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/anthropics/purple-hailo/pkg/control"
	"github.com/anthropics/purple-hailo/pkg/hef"
)

func main() {
	hefPath := flag.String("hef", "", "Path to HEF file")
	flag.Parse()

	if *hefPath == "" {
		log.Fatal("Usage: debug_actions -hef <path>")
	}

	h, err := hef.Parse(*hefPath)
	if err != nil {
		log.Fatalf("Failed to parse HEF: %v", err)
	}

	ng, err := h.GetDefaultNetworkGroup()
	if err != nil {
		log.Fatalf("Failed to get network group: %v", err)
	}

	// Count action types in preliminary config
	if ng.PreliminaryConfig != nil {
		fmt.Println("=== Preliminary Config Actions ===")
		actionCounts := make(map[hef.ActionType]int)
		for _, op := range ng.PreliminaryConfig.Operations {
			for _, action := range op.Actions {
				actionCounts[action.Type]++
			}
		}
		for t, count := range actionCounts {
			fmt.Printf("  ActionType(%d): %d\n", t, count)
		}
	}

	// Count action types in each context
	for i, ctx := range ng.Contexts {
		fmt.Printf("\n=== Context %d Actions ===\n", i)
		actionCounts := make(map[hef.ActionType]int)
		for _, op := range ctx.Operations {
			for _, action := range op.Actions {
				actionCounts[action.Type]++
			}
		}
		for t, count := range actionCounts {
			fmt.Printf("  ActionType(%d): %d\n", t, count)
		}
	}

	// Show first few serialized actions from preliminary context
	if ng.PreliminaryConfig != nil && len(ng.PreliminaryConfig.Operations) > 0 {
		fmt.Println("\n=== Preliminary Serialized Actions (first 10) ===")
		timestamp := uint32(0xFFFFFFFF)
		count := 0
		for _, op := range ng.PreliminaryConfig.Operations {
			for _, action := range op.Actions {
				if count >= 10 {
					break
				}
				bytes, err := control.ConvertHefActionToFirmware(&action, timestamp)
				if err != nil {
					fmt.Printf("  %d. ActionType(%d) -> Error: %v\n", count, action.Type, err)
				} else if bytes == nil {
					fmt.Printf("  %d. ActionType(%d) -> nil (skipped)\n", count, action.Type)
				} else {
					fmt.Printf("  %d. ActionType(%d) -> % 02x\n", count, action.Type, bytes)
				}
				timestamp--
				count++
			}
		}
	}

	// Show first few serialized actions from context 0
	if len(ng.Contexts) > 0 && len(ng.Contexts[0].Operations) > 0 {
		fmt.Println("\n=== Context 0 Serialized Actions (first 10) ===")
		timestamp := uint32(0xFFFFFFFF)
		count := 0
		for _, op := range ng.Contexts[0].Operations {
			for _, action := range op.Actions {
				if count >= 10 {
					break
				}
				bytes, err := control.ConvertHefActionToFirmware(&action, timestamp)
				if err != nil {
					fmt.Printf("  %d. ActionType(%d) -> Error: %v\n", count, action.Type, err)
				} else if bytes == nil {
					fmt.Printf("  %d. ActionType(%d) -> nil (skipped)\n", count, action.Type)
				} else {
					fmt.Printf("  %d. ActionType(%d) -> % 02x\n", count, action.Type, bytes)
				}
				timestamp--
				count++
			}
		}
	}
}
