package stream

import (
	"fmt"
	"time"

	"github.com/anthropics/purple-hailo/pkg/device"
	"github.com/anthropics/purple-hailo/pkg/driver"
	"github.com/anthropics/purple-hailo/pkg/hef"
)

// VStreamParams holds parameters for VStream creation
type VStreamParams struct {
	FormatType hef.FormatType
	Timeout    time.Duration
	QueueDepth int
	BatchSize  uint32
}

// DefaultVStreamParams returns default VStream parameters
func DefaultVStreamParams() VStreamParams {
	return VStreamParams{
		FormatType: hef.FormatTypeAuto,
		Timeout:    5 * time.Second,
		QueueDepth: 2,
		BatchSize:  1,
	}
}

// VStreamSet holds input and output VStreams
type VStreamSet struct {
	Inputs  []*InputVStream
	Outputs []*OutputVStream
	device  *driver.DeviceFile
	channels *ChannelSet
}

// Close closes all VStreams
func (vs *VStreamSet) Close() error {
	var lastErr error

	for _, input := range vs.Inputs {
		if err := input.Close(); err != nil {
			lastErr = err
		}
	}

	for _, output := range vs.Outputs {
		if err := output.Close(); err != nil {
			lastErr = err
		}
	}

	if vs.channels != nil {
		if err := vs.channels.DisableAll(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// InputByName returns the input VStream with the given name
func (vs *VStreamSet) InputByName(name string) *InputVStream {
	for _, input := range vs.Inputs {
		if input.info.Name == name {
			return input
		}
	}
	return nil
}

// OutputByName returns the output VStream with the given name
func (vs *VStreamSet) OutputByName(name string) *OutputVStream {
	for _, output := range vs.Outputs {
		if output.info.Name == name {
			return output
		}
	}
	return nil
}

// BuildVStreams creates VStreams from a configured network group
func BuildVStreams(ng *device.ConfiguredNetworkGroup, params VStreamParams) (*VStreamSet, error) {
	dev := ng.Device().DeviceFile()

	inputInfos := ng.InputStreamInfos()
	outputInfos := ng.OutputStreamInfos()

	channels := NewChannelSet(dev)

	inputs := make([]*InputVStream, len(inputInfos))
	outputs := make([]*OutputVStream, len(outputInfos))

	// Create input VStreams
	// Input channels use engine 0, channels 0-15
	for i, info := range inputInfos {
		channelIdx := uint8(i % 16)
		channel := channels.AddChannel(0, channelIdx)

		vsInfo := VStreamInfo{
			Name:      info.Name,
			FrameSize: info.FrameSize,
			Height:    info.Height,
			Width:     info.Width,
			Channels:  info.Channels,
			QuantInfo: info.QuantInfo,
		}

		input, err := NewInputVStream(InputVStreamConfig{
			Info:       vsInfo,
			Device:     dev,
			Channel:    channel,
			Timeout:    params.Timeout,
			BatchSize:  params.BatchSize,
			QueueDepth: params.QueueDepth,
		})
		if err != nil {
			// Clean up already created streams
			for j := 0; j < i; j++ {
				inputs[j].Close()
			}
			return nil, fmt.Errorf("failed to create input VStream %s: %w", info.Name, err)
		}

		inputs[i] = input
	}

	// Create output VStreams
	// Output channels use engine 0, channels 16-31
	for i, info := range outputInfos {
		channelIdx := uint8(16 + (i % 16))
		channel := channels.AddChannel(0, channelIdx)

		vsInfo := VStreamInfo{
			Name:      info.Name,
			FrameSize: info.FrameSize,
			Height:    info.Height,
			Width:     info.Width,
			Channels:  info.Channels,
			QuantInfo: info.QuantInfo,
		}

		output, err := NewOutputVStream(OutputVStreamConfig{
			Info:       vsInfo,
			Device:     dev,
			Channel:    channel,
			Timeout:    params.Timeout,
			BatchSize:  params.BatchSize,
			QueueDepth: params.QueueDepth,
		})
		if err != nil {
			// Clean up
			for _, in := range inputs {
				if in != nil {
					in.Close()
				}
			}
			for j := 0; j < i; j++ {
				outputs[j].Close()
			}
			return nil, fmt.Errorf("failed to create output VStream %s: %w", info.Name, err)
		}

		outputs[i] = output
	}

	// Enable all channels
	if err := channels.EnableAll(false); err != nil {
		// Clean up
		for _, in := range inputs {
			in.Close()
		}
		for _, out := range outputs {
			out.Close()
		}
		return nil, fmt.Errorf("failed to enable channels: %w", err)
	}

	return &VStreamSet{
		Inputs:   inputs,
		Outputs:  outputs,
		device:   dev,
		channels: channels,
	}, nil
}

// BuildInputVStreams creates only input VStreams
func BuildInputVStreams(ng *device.ConfiguredNetworkGroup, params VStreamParams) ([]*InputVStream, error) {
	dev := ng.Device().DeviceFile()
	inputInfos := ng.InputStreamInfos()

	inputs := make([]*InputVStream, len(inputInfos))
	channels := NewChannelSet(dev)

	for i, info := range inputInfos {
		channelIdx := uint8(i % 16)
		channel := channels.AddChannel(0, channelIdx)

		vsInfo := VStreamInfo{
			Name:      info.Name,
			FrameSize: info.FrameSize,
			Height:    info.Height,
			Width:     info.Width,
			Channels:  info.Channels,
			QuantInfo: info.QuantInfo,
		}

		input, err := NewInputVStream(InputVStreamConfig{
			Info:       vsInfo,
			Device:     dev,
			Channel:    channel,
			Timeout:    params.Timeout,
			BatchSize:  params.BatchSize,
			QueueDepth: params.QueueDepth,
		})
		if err != nil {
			for j := 0; j < i; j++ {
				inputs[j].Close()
			}
			return nil, fmt.Errorf("failed to create input VStream %s: %w", info.Name, err)
		}

		inputs[i] = input
	}

	if err := channels.EnableAll(false); err != nil {
		for _, in := range inputs {
			in.Close()
		}
		return nil, fmt.Errorf("failed to enable channels: %w", err)
	}

	return inputs, nil
}

// BuildOutputVStreams creates only output VStreams
func BuildOutputVStreams(ng *device.ConfiguredNetworkGroup, params VStreamParams) ([]*OutputVStream, error) {
	dev := ng.Device().DeviceFile()
	outputInfos := ng.OutputStreamInfos()

	outputs := make([]*OutputVStream, len(outputInfos))
	channels := NewChannelSet(dev)

	for i, info := range outputInfos {
		channelIdx := uint8(16 + (i % 16))
		channel := channels.AddChannel(0, channelIdx)

		vsInfo := VStreamInfo{
			Name:      info.Name,
			FrameSize: info.FrameSize,
			Height:    info.Height,
			Width:     info.Width,
			Channels:  info.Channels,
			QuantInfo: info.QuantInfo,
		}

		output, err := NewOutputVStream(OutputVStreamConfig{
			Info:       vsInfo,
			Device:     dev,
			Channel:    channel,
			Timeout:    params.Timeout,
			BatchSize:  params.BatchSize,
			QueueDepth: params.QueueDepth,
		})
		if err != nil {
			for j := 0; j < i; j++ {
				outputs[j].Close()
			}
			return nil, fmt.Errorf("failed to create output VStream %s: %w", info.Name, err)
		}

		outputs[i] = output
	}

	if err := channels.EnableAll(false); err != nil {
		for _, out := range outputs {
			out.Close()
		}
		return nil, fmt.Errorf("failed to enable channels: %w", err)
	}

	return outputs, nil
}
