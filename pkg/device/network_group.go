package device

import (
	"sync"

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

	// TODO: Actually configure the device via IOCTL
	// This would involve:
	// 1. Writing action lists for preliminary config
	// 2. Programming VDMA descriptors
	// 3. Enabling VDMA channels

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

	// TODO: Actually disable VDMA channels via IOCTL

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
