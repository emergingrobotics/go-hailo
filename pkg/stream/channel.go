package stream

import (
	"fmt"
	"sync"

	"github.com/anthropics/purple-hailo/pkg/driver"
)

// VdmaChannel represents a VDMA channel for DMA transfers
type VdmaChannel struct {
	engineIndex  uint8
	channelIndex uint8
	device       *driver.DeviceFile
	enabled      bool
	mu           sync.Mutex
}

// NewVdmaChannel creates a new VDMA channel reference
func NewVdmaChannel(dev *driver.DeviceFile, engineIndex, channelIndex uint8) *VdmaChannel {
	return &VdmaChannel{
		engineIndex:  engineIndex,
		channelIndex: channelIndex,
		device:       dev,
		enabled:      false,
	}
}

// EngineIndex returns the engine index
func (c *VdmaChannel) EngineIndex() uint8 {
	return c.engineIndex
}

// ChannelIndex returns the channel index
func (c *VdmaChannel) ChannelIndex() uint8 {
	return c.channelIndex
}

// IsEnabled returns whether the channel is enabled
func (c *VdmaChannel) IsEnabled() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enabled
}

// Enable enables the VDMA channel
func (c *VdmaChannel) Enable(enableTimestamps bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.enabled {
		return nil
	}

	var bitmap [driver.MaxVdmaEngines]uint32
	bitmap[c.engineIndex] = 1 << c.channelIndex

	err := c.device.VdmaEnableChannels(bitmap, enableTimestamps)
	if err != nil {
		return fmt.Errorf("failed to enable channel: %w", err)
	}

	c.enabled = true
	return nil
}

// Disable disables the VDMA channel
func (c *VdmaChannel) Disable() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.enabled {
		return nil
	}

	var bitmap [driver.MaxVdmaEngines]uint32
	bitmap[c.engineIndex] = 1 << c.channelIndex

	err := c.device.VdmaDisableChannels(bitmap)
	if err != nil {
		return fmt.Errorf("failed to disable channel: %w", err)
	}

	c.enabled = false
	return nil
}

// LaunchTransfer launches a DMA transfer on this channel
func (c *VdmaChannel) LaunchTransfer(descList *DescriptorList, buffer *Buffer, startingDesc uint32, shouldBind bool, firstInterruptsDomain, lastInterruptsDomain driver.InterruptsDomain) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.enabled {
		return fmt.Errorf("channel not enabled")
	}

	params := driver.VdmaLaunchTransferParams{
		EngineIndex:           c.engineIndex,
		ChannelIndex:          c.channelIndex,
		DescHandle:            descList.Handle(),
		StartingDesc:          startingDesc,
		ShouldBind:            shouldBind,
		BuffersCount:          1,
		FirstInterruptsDomain: firstInterruptsDomain,
		LastInterruptsDomain:  lastInterruptsDomain,
		IsDebug:               false,
	}

	// Set up the buffer reference
	params.Buffers[0] = driver.VdmaTransferBuffer{
		BufferType: driver.DmaUserPtrBuffer,
		AddrOrFd:   uintptr(buffer.Handle()),
		Size:       uint32(buffer.Size()),
	}

	return c.device.VdmaLaunchTransfer(&params)
}

// WaitForInterrupt waits for transfer completion interrupt
func (c *VdmaChannel) WaitForInterrupt() error {
	var bitmap [driver.MaxVdmaEngines]uint32
	bitmap[c.engineIndex] = 1 << c.channelIndex

	_, err := c.device.VdmaInterruptsWait(bitmap)
	return err
}

// ChannelSet manages a set of VDMA channels
type ChannelSet struct {
	channels []*VdmaChannel
	device   *driver.DeviceFile
	mu       sync.Mutex
}

// NewChannelSet creates a new channel set
func NewChannelSet(dev *driver.DeviceFile) *ChannelSet {
	return &ChannelSet{
		device:   dev,
		channels: make([]*VdmaChannel, 0),
	}
}

// AddChannel adds a channel to the set
func (cs *ChannelSet) AddChannel(engineIndex, channelIndex uint8) *VdmaChannel {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	ch := NewVdmaChannel(cs.device, engineIndex, channelIndex)
	cs.channels = append(cs.channels, ch)
	return ch
}

// EnableAll enables all channels in the set
func (cs *ChannelSet) EnableAll(enableTimestamps bool) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	var bitmap [driver.MaxVdmaEngines]uint32
	for _, ch := range cs.channels {
		bitmap[ch.engineIndex] |= 1 << ch.channelIndex
	}

	err := cs.device.VdmaEnableChannels(bitmap, enableTimestamps)
	if err != nil {
		return fmt.Errorf("failed to enable channels: %w", err)
	}

	for _, ch := range cs.channels {
		ch.mu.Lock()
		ch.enabled = true
		ch.mu.Unlock()
	}

	return nil
}

// DisableAll disables all channels in the set
func (cs *ChannelSet) DisableAll() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	var bitmap [driver.MaxVdmaEngines]uint32
	for _, ch := range cs.channels {
		bitmap[ch.engineIndex] |= 1 << ch.channelIndex
	}

	err := cs.device.VdmaDisableChannels(bitmap)
	if err != nil {
		return fmt.Errorf("failed to disable channels: %w", err)
	}

	for _, ch := range cs.channels {
		ch.mu.Lock()
		ch.enabled = false
		ch.mu.Unlock()
	}

	return nil
}

// WaitForAnyInterrupt waits for interrupt on any channel in the set
func (cs *ChannelSet) WaitForAnyInterrupt() (*driver.VdmaInterruptsWaitParams, error) {
	cs.mu.Lock()
	var bitmap [driver.MaxVdmaEngines]uint32
	for _, ch := range cs.channels {
		bitmap[ch.engineIndex] |= 1 << ch.channelIndex
	}
	cs.mu.Unlock()

	return cs.device.VdmaInterruptsWait(bitmap)
}

// Channels returns the list of channels
func (cs *ChannelSet) Channels() []*VdmaChannel {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	result := make([]*VdmaChannel, len(cs.channels))
	copy(result, cs.channels)
	return result
}

// Count returns the number of channels
func (cs *ChannelSet) Count() int {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return len(cs.channels)
}
