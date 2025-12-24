package device

import (
	"fmt"
	"sync"

	"github.com/anthropics/purple-hailo/pkg/control"
	"github.com/anthropics/purple-hailo/pkg/hef"
)

// NetworkGroupState represents the state of a network group
type NetworkGroupState int

const (
	StateUninitialized NetworkGroupState = iota
	StateConfigured
	StateActivated
	StateDeactivated
)

// StreamInfo contains stream information
type StreamInfo struct {
	Name      string
	FrameSize uint64
	Height    uint32
	Width     uint32
	Channels  uint32
	QuantInfo hef.QuantInfo
}

// ConfiguredNetworkGroup represents a configured network group
type ConfiguredNetworkGroup struct {
	device  *Device
	hef     *hef.Hef
	info    *hef.NetworkGroupInfo
	state   NetworkGroupState
	inputs  []StreamInfo
	outputs []StreamInfo
	mu      sync.RWMutex

	// Control protocol state
	networkGroupIndex uint8  // Index of this network group (0 for first/default)
	controlSequence   uint32 // Incrementing sequence number for control messages
}

// Name returns the network group name
func (ng *ConfiguredNetworkGroup) Name() string {
	return ng.info.Name
}

// State returns the current state
func (ng *ConfiguredNetworkGroup) State() NetworkGroupState {
	ng.mu.RLock()
	defer ng.mu.RUnlock()
	return ng.state
}

// InputStreamInfos returns input stream information
func (ng *ConfiguredNetworkGroup) InputStreamInfos() []StreamInfo {
	return ng.inputs
}

// OutputStreamInfos returns output stream information
func (ng *ConfiguredNetworkGroup) OutputStreamInfos() []StreamInfo {
	return ng.outputs
}

// IsMultiContext returns whether this is a multi-context network
func (ng *ConfiguredNetworkGroup) IsMultiContext() bool {
	return ng.info.IsMultiContext
}

// HasNmsOutput returns whether any output is an NMS layer
func (ng *ConfiguredNetworkGroup) HasNmsOutput() bool {
	return ng.info.HasNmsOutput()
}

// GetNmsInfo returns NMS configuration if present
func (ng *ConfiguredNetworkGroup) GetNmsInfo() *hef.VStreamInfo {
	return ng.info.GetNmsInfo()
}

// BottleneckFps returns the bottleneck FPS from model
func (ng *ConfiguredNetworkGroup) BottleneckFps() float64 {
	return ng.info.BottleneckFps
}

// GetUserInputs returns only user-facing input streams
func (ng *ConfiguredNetworkGroup) GetUserInputs() []StreamInfo {
	userInputs := ng.info.GetUserInputs()
	result := make([]StreamInfo, len(userInputs))
	for i, s := range userInputs {
		result[i] = StreamInfo{
			Name:      s.Name,
			FrameSize: uint64(s.Shape.Height * s.Shape.Width * s.Shape.Features),
			Height:    s.Shape.Height,
			Width:     s.Shape.Width,
			Channels:  s.Shape.Features,
			QuantInfo: s.QuantInfo,
		}
	}
	return result
}

// GetUserOutputs returns only user-facing output streams
func (ng *ConfiguredNetworkGroup) GetUserOutputs() []StreamInfo {
	userOutputs := ng.info.GetUserOutputs()
	result := make([]StreamInfo, len(userOutputs))
	for i, s := range userOutputs {
		result[i] = StreamInfo{
			Name:      s.Name,
			FrameSize: uint64(s.Shape.Height * s.Shape.Width * s.Shape.Features),
			Height:    s.Shape.Height,
			Width:     s.Shape.Width,
			Channels:  s.Shape.Features,
			QuantInfo: s.QuantInfo,
		}
	}
	return result
}

// Activate activates the network group for inference
func (ng *ConfiguredNetworkGroup) Activate() (*ActivatedNetworkGroup, error) {
	ng.mu.Lock()
	defer ng.mu.Unlock()

	if ng.state != StateConfigured && ng.state != StateDeactivated {
		return nil, ErrInvalidState
	}

	// Only call firmware if we have a real device (not a mock)
	if ng.device != nil && ng.device.DeviceFile() != nil {
		// Step 0: Clear any previously configured apps
		ng.controlSequence++
		fmt.Printf("[activate] Clearing configured apps (sequence %d)\n", ng.controlSequence)
		if err := control.ClearConfiguredApps(ng.device.DeviceFile(), ng.controlSequence); err != nil {
			fmt.Printf("[activate] Warning: clear_configured_apps failed: %v (continuing anyway)\n", err)
		}

		// Step 1: Send network group header
		ng.controlSequence++
		fmt.Printf("[activate] Sending network group header (sequence %d)\n", ng.controlSequence)

		// Count dynamic contexts from HEF
		dynamicContextsCount := uint16(len(ng.info.Contexts))
		if dynamicContextsCount == 0 {
			dynamicContextsCount = 1 // At least one context
		}

		// Create application header with HEF metadata (v4.20.0 format)
		appHeader := control.CreateDefaultApplicationHeader(dynamicContextsCount)

		err := control.SetNetworkGroupHeader(
			ng.device.DeviceFile(),
			ng.controlSequence,
			appHeader,
		)
		if err != nil {
			return nil, fmt.Errorf("set_network_group_header failed: %w", err)
		}
		fmt.Printf("[activate] Network group header sent successfully\n")

		// Step 2: Send context info for each context type
		// Build action lists from HEF context configurations
		fmt.Printf("[activate] Sending context info...\n")

		// Send ACTIVATION context - typically empty for most models
		// ACTIVATION context runs during activation (not inference)
		activationData := control.BuildEmptyActionList()
		if err := control.SendContextInfoChunks(ng.device.DeviceFile(), &ng.controlSequence,
			control.ContextTypeActivation, activationData); err != nil {
			return nil, fmt.Errorf("send activation context failed: %w", err)
		}
		fmt.Printf("[activate] Activation context sent (%d bytes)\n", len(activationData))

		// Send BATCH_SWITCHING context - typically empty
		batchSwitchingData := control.BuildEmptyActionList()
		if err := control.SendContextInfoChunks(ng.device.DeviceFile(), &ng.controlSequence,
			control.ContextTypeBatchSwitching, batchSwitchingData); err != nil {
			return nil, fmt.Errorf("send batch_switching context failed: %w", err)
		}
		fmt.Printf("[activate] Batch switching context sent (%d bytes)\n", len(batchSwitchingData))

		// Send PRELIMINARY context from HEF
		preliminaryData := []byte{}
		if ng.info.PreliminaryConfig != nil && len(ng.info.PreliminaryConfig.Operations) > 0 {
			var err error
			preliminaryData, err = control.BuildContextActionList(ng.info.PreliminaryConfig.Operations)
			if err != nil {
				fmt.Printf("[activate] Warning: failed to build preliminary action list: %v\n", err)
				preliminaryData = control.BuildEmptyActionList()
			}
		} else {
			preliminaryData = control.BuildEmptyActionList()
		}
		if err := control.SendContextInfoChunks(ng.device.DeviceFile(), &ng.controlSequence,
			control.ContextTypePreliminary, preliminaryData); err != nil {
			return nil, fmt.Errorf("send preliminary context failed: %w", err)
		}
		fmt.Printf("[activate] Preliminary context sent (%d bytes)\n", len(preliminaryData))

		// Send DYNAMIC contexts from HEF (one per HEF context)
		for i := 0; i < int(dynamicContextsCount); i++ {
			var dynamicData []byte
			if i < len(ng.info.Contexts) && len(ng.info.Contexts[i].Operations) > 0 {
				var err error
				dynamicData, err = control.BuildContextActionList(ng.info.Contexts[i].Operations)
				if err != nil {
					fmt.Printf("[activate] Warning: failed to build dynamic context %d action list: %v\n", i, err)
					dynamicData = control.BuildEmptyActionList()
				}
			} else {
				dynamicData = control.BuildEmptyActionList()
			}
			if err := control.SendContextInfoChunks(ng.device.DeviceFile(), &ng.controlSequence,
				control.ContextTypeDynamic, dynamicData); err != nil {
				return nil, fmt.Errorf("send dynamic context %d failed: %w", i, err)
			}
			fmt.Printf("[activate] Dynamic context %d sent (%d bytes)\n", i, len(dynamicData))
		}

		// Step 3: Enable core op
		ng.controlSequence++
		fmt.Printf("[activate] Enabling core op for network group %d (sequence %d)\n",
			ng.networkGroupIndex, ng.controlSequence)

		err = control.EnableCoreOp(
			ng.device.DeviceFile(),
			ng.controlSequence,
			ng.networkGroupIndex,
			0, // dynamic batch size (0 = use default)
			0, // batch count (0 = infinite)
		)
		if err != nil {
			return nil, fmt.Errorf("enable_core_op failed: %w", err)
		}
		fmt.Printf("[activate] Core op enabled successfully\n")
	} else {
		fmt.Printf("[activate] No device, skipping firmware calls\n")
	}

	ng.state = StateActivated
	return &ActivatedNetworkGroup{
		configured: ng,
	}, nil
}

// Close closes the configured network group
func (ng *ConfiguredNetworkGroup) Close() error {
	ng.mu.Lock()
	defer ng.mu.Unlock()

	if ng.state == StateActivated {
		return ErrStillActivated
	}

	ng.state = StateUninitialized
	return nil
}

// ActivatedNetworkGroup represents an activated network group ready for inference
type ActivatedNetworkGroup struct {
	configured  *ConfiguredNetworkGroup
	mu          sync.Mutex
	deactivated bool
}

// Deactivate deactivates the network group
func (ang *ActivatedNetworkGroup) Deactivate() error {
	ang.mu.Lock()
	defer ang.mu.Unlock()

	if ang.deactivated {
		return nil
	}

	// Only call firmware if we have a real device (not a mock)
	if ang.configured.device != nil && ang.configured.device.DeviceFile() != nil {
		ang.configured.controlSequence++
		fmt.Printf("[deactivate] Resetting context switch state machine (sequence %d)\n",
			ang.configured.controlSequence)

		err := control.ResetContextSwitchStateMachine(
			ang.configured.device.DeviceFile(),
			ang.configured.controlSequence,
		)
		if err != nil {
			// Log but don't fail - deactivation should still proceed
			fmt.Printf("[deactivate] Warning: reset_context_switch_state_machine failed: %v\n", err)
		} else {
			fmt.Printf("[deactivate] Context switch state machine reset successfully\n")
		}
	}

	ang.configured.mu.Lock()
	ang.configured.state = StateDeactivated
	ang.configured.mu.Unlock()

	ang.deactivated = true
	return nil
}

// IsActive returns whether the network group is active
func (ang *ActivatedNetworkGroup) IsActive() bool {
	ang.mu.Lock()
	defer ang.mu.Unlock()
	return !ang.deactivated
}

// ConfiguredNetworkGroup returns the underlying configured network group
func (ang *ActivatedNetworkGroup) ConfiguredNetworkGroup() *ConfiguredNetworkGroup {
	return ang.configured
}

// Device returns the device this network group is configured on
func (ang *ActivatedNetworkGroup) Device() *Device {
	return ang.configured.device
}

// Device returns the device this network group is configured on
func (ng *ConfiguredNetworkGroup) Device() *Device {
	return ng.device
}
