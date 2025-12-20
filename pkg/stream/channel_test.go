//go:build unit

package stream

import (
	"testing"

	"github.com/anthropics/purple-hailo/pkg/driver"
)

func TestVdmaChannelIndices(t *testing.T) {
	ch := &VdmaChannel{
		engineIndex:  0,
		channelIndex: 5,
	}

	if ch.EngineIndex() != 0 {
		t.Errorf("EngineIndex() = %d, expected 0", ch.EngineIndex())
	}

	if ch.ChannelIndex() != 5 {
		t.Errorf("ChannelIndex() = %d, expected 5", ch.ChannelIndex())
	}
}

func TestVdmaChannelInitialState(t *testing.T) {
	ch := &VdmaChannel{
		engineIndex:  0,
		channelIndex: 0,
		enabled:      false,
	}

	if ch.IsEnabled() {
		t.Error("New channel should not be enabled")
	}
}

func TestChannelSetAddChannel(t *testing.T) {
	cs := &ChannelSet{
		channels: make([]*VdmaChannel, 0),
	}

	ch1 := cs.AddChannel(0, 0)
	if ch1 == nil {
		t.Fatal("AddChannel returned nil")
	}

	if cs.Count() != 1 {
		t.Errorf("Count() = %d after adding 1 channel", cs.Count())
	}

	ch2 := cs.AddChannel(0, 1)
	if ch2 == nil {
		t.Fatal("AddChannel returned nil")
	}

	if cs.Count() != 2 {
		t.Errorf("Count() = %d after adding 2 channels", cs.Count())
	}
}

func TestChannelSetChannels(t *testing.T) {
	cs := &ChannelSet{
		channels: make([]*VdmaChannel, 0),
	}

	cs.AddChannel(0, 0)
	cs.AddChannel(0, 1)
	cs.AddChannel(1, 0)

	channels := cs.Channels()
	if len(channels) != 3 {
		t.Errorf("Channels() returned %d channels, expected 3", len(channels))
	}

	// Verify indices
	if channels[0].ChannelIndex() != 0 {
		t.Error("First channel has wrong index")
	}
	if channels[1].ChannelIndex() != 1 {
		t.Error("Second channel has wrong index")
	}
	if channels[2].EngineIndex() != 1 {
		t.Error("Third channel has wrong engine")
	}
}

func TestNewVdmaChannel(t *testing.T) {
	// Mocking driver.DeviceFile is hard, so test without it
	ch := NewVdmaChannel(nil, 2, 15)

	if ch.EngineIndex() != 2 {
		t.Errorf("EngineIndex() = %d, expected 2", ch.EngineIndex())
	}

	if ch.ChannelIndex() != 15 {
		t.Errorf("ChannelIndex() = %d, expected 15", ch.ChannelIndex())
	}

	if ch.IsEnabled() {
		t.Error("New channel should not be enabled")
	}
}

func TestChannelInputOutputSplit(t *testing.T) {
	// Input channels: 0-15 on engine 0
	// Output channels: 16-31 on engine 0
	
	inputCh := NewVdmaChannel(nil, 0, 5)
	if inputCh.ChannelIndex() >= 16 {
		t.Error("Input channel should be < 16")
	}

	outputCh := NewVdmaChannel(nil, 0, 20)
	if outputCh.ChannelIndex() < 16 {
		t.Error("Output channel should be >= 16")
	}

	// Verify expected output channel base
	expectedOutputBase := uint8(driver.VdmaDestChannelsStart)
	if expectedOutputBase != 16 {
		t.Errorf("VdmaDestChannelsStart = %d, expected 16", expectedOutputBase)
	}
}

func TestNewChannelSet(t *testing.T) {
	cs := NewChannelSet(nil)

	if cs == nil {
		t.Fatal("NewChannelSet returned nil")
	}

	if cs.Count() != 0 {
		t.Errorf("New ChannelSet should have 0 channels, got %d", cs.Count())
	}
}
