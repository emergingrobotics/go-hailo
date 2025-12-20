package device

import (
	"fmt"
	"sync"

	"github.com/anthropics/purple-hailo/pkg/driver"
	"github.com/anthropics/purple-hailo/pkg/hef"
)

// Device represents an open Hailo device
type Device struct {
	df         *driver.DeviceFile
	properties *driver.DeviceProperties
	driverInfo *driver.DriverInfo
	mu         sync.RWMutex
	closed     bool
}

// Open opens a Hailo device by path
func Open(path string) (*Device, error) {
	df, err := driver.OpenDevice(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open device: %w", err)
	}

	// Query device properties
	props, err := df.QueryDeviceProperties()
	if err != nil {
		df.Close()
		return nil, fmt.Errorf("failed to query device properties: %w", err)
	}

	// Query driver info
	driverInfo, err := df.QueryDriverInfo()
	if err != nil {
		df.Close()
		return nil, fmt.Errorf("failed to query driver info: %w", err)
	}

	return &Device{
		df:         df,
		properties: props,
		driverInfo: driverInfo,
	}, nil
}

// OpenFirst opens the first available Hailo device
func OpenFirst() (*Device, error) {
	devices, err := Scan()
	if err != nil {
		return nil, fmt.Errorf("failed to scan devices: %w", err)
	}

	if len(devices) == 0 {
		return nil, ErrNoDevices
	}

	return Open(devices[0].Path)
}

// Close closes the device
func (d *Device) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return nil
	}

	d.closed = true
	return d.df.Close()
}

// Path returns the device path
func (d *Device) Path() string {
	return d.df.Path()
}

// BoardType returns the board type
func (d *Device) BoardType() driver.BoardType {
	return d.properties.BoardType
}

// IsFirmwareLoaded returns whether firmware is loaded
func (d *Device) IsFirmwareLoaded() bool {
	return d.properties.IsFwLoaded
}

// DriverVersion returns the driver version string
func (d *Device) DriverVersion() string {
	return fmt.Sprintf("%d.%d.%d",
		d.driverInfo.MajorVersion,
		d.driverInfo.MinorVersion,
		d.driverInfo.RevisionVersion)
}

// DeviceFile returns the underlying driver device file
func (d *Device) DeviceFile() *driver.DeviceFile {
	return d.df
}

// ConfigureNetworkGroup configures a network group from a HEF
func (d *Device) ConfigureNetworkGroup(hefFile *hef.Hef, groupName string) (*ConfiguredNetworkGroup, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.closed {
		return nil, ErrDeviceClosed
	}

	// Get network group info from HEF
	var ngInfo *hef.NetworkGroupInfo
	var err error
	if groupName == "" {
		ngInfo, err = hefFile.GetDefaultNetworkGroup()
	} else {
		ngInfo, err = hefFile.GetNetworkGroup(groupName)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get network group: %w", err)
	}

	// Create configured network group
	cng := &ConfiguredNetworkGroup{
		device:  d,
		hef:     hefFile,
		info:    ngInfo,
		state:   StateConfigured,
		inputs:  make([]StreamInfo, len(ngInfo.InputStreams)),
		outputs: make([]StreamInfo, len(ngInfo.OutputStreams)),
	}

	// Copy stream info
	for i, s := range ngInfo.InputStreams {
		cng.inputs[i] = StreamInfo{
			Name:      s.Name,
			FrameSize: uint64(s.Shape.Height * s.Shape.Width * s.Shape.Features),
			Height:    s.Shape.Height,
			Width:     s.Shape.Width,
			Channels:  s.Shape.Features,
			QuantInfo: s.QuantInfo,
		}
	}
	for i, s := range ngInfo.OutputStreams {
		cng.outputs[i] = StreamInfo{
			Name:      s.Name,
			FrameSize: uint64(s.Shape.Height * s.Shape.Width * s.Shape.Features),
			Height:    s.Shape.Height,
			Width:     s.Shape.Width,
			Channels:  s.Shape.Features,
			QuantInfo: s.QuantInfo,
		}
	}

	return cng, nil
}

// ConfigureDefaultNetworkGroup configures the default network group from a HEF
func (d *Device) ConfigureDefaultNetworkGroup(hefFile *hef.Hef) (*ConfiguredNetworkGroup, error) {
	return d.ConfigureNetworkGroup(hefFile, "")
}

// ResetNnCore resets the neural network core
func (d *Device) ResetNnCore() error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.closed {
		return ErrDeviceClosed
	}

	return d.df.ResetNnCore()
}
