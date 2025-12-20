package stream

import (
	"fmt"
	"sync"

	"github.com/anthropics/purple-hailo/pkg/driver"
)

// DescriptorList represents a DMA descriptor list
type DescriptorList struct {
	handle     uintptr
	dmaAddress uint64
	descCount  uint64
	pageSize   uint16
	isCircular bool
	device     *driver.DeviceFile
	mu         sync.Mutex
	released   bool
}

// CreateDescriptorList creates a new descriptor list
func CreateDescriptorList(dev *driver.DeviceFile, descCount uint64, pageSize uint16, isCircular bool) (*DescriptorList, error) {
	if descCount == 0 {
		return nil, fmt.Errorf("descriptor count cannot be zero")
	}

	handle, dmaAddr, err := dev.DescListCreate(descCount, pageSize, isCircular)
	if err != nil {
		return nil, fmt.Errorf("DescListCreate failed: %w", err)
	}

	return &DescriptorList{
		handle:     handle,
		dmaAddress: dmaAddr,
		descCount:  descCount,
		pageSize:   pageSize,
		isCircular: isCircular,
		device:     dev,
	}, nil
}

// Handle returns the descriptor list handle
func (dl *DescriptorList) Handle() uintptr {
	return dl.handle
}

// DmaAddress returns the DMA address of the descriptor list
func (dl *DescriptorList) DmaAddress() uint64 {
	return dl.dmaAddress
}

// DescCount returns the number of descriptors
func (dl *DescriptorList) DescCount() uint64 {
	return dl.descCount
}

// IsCircular returns whether the descriptor list is circular
func (dl *DescriptorList) IsCircular() bool {
	return dl.isCircular
}

// Program programs the descriptor list with buffer information
func (dl *DescriptorList) Program(buffer *Buffer, channelIndex uint8, batchSize uint32, startingDesc uint32, shouldBind bool, interruptsDomain driver.InterruptsDomain) error {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	if dl.released {
		return fmt.Errorf("descriptor list already released")
	}

	params := driver.DescListProgramParams{
		BufferHandle:         buffer.Handle(),
		BufferSize:           buffer.Size(),
		BufferOffset:         0,
		BatchSize:            batchSize,
		DescHandle:           dl.handle,
		ChannelIndex:         channelIndex,
		StartingDesc:         startingDesc,
		ShouldBind:           shouldBind,
		LastInterruptsDomain: interruptsDomain,
		IsDebug:              false,
		Stride:               0,
	}

	return dl.device.DescListProgram(&params)
}

// ProgramWithOffset programs the descriptor list with buffer information and offset
func (dl *DescriptorList) ProgramWithOffset(buffer *Buffer, offset uint64, size uint64, channelIndex uint8, batchSize uint32, startingDesc uint32, shouldBind bool, interruptsDomain driver.InterruptsDomain) error {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	if dl.released {
		return fmt.Errorf("descriptor list already released")
	}

	params := driver.DescListProgramParams{
		BufferHandle:         buffer.Handle(),
		BufferSize:           size,
		BufferOffset:         offset,
		BatchSize:            batchSize,
		DescHandle:           dl.handle,
		ChannelIndex:         channelIndex,
		StartingDesc:         startingDesc,
		ShouldBind:           shouldBind,
		LastInterruptsDomain: interruptsDomain,
		IsDebug:              false,
		Stride:               0,
	}

	return dl.device.DescListProgram(&params)
}

// Release releases the descriptor list
func (dl *DescriptorList) Release() error {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	if dl.released {
		return nil
	}

	err := dl.device.DescListRelease(dl.handle)
	if err != nil {
		return fmt.Errorf("DescListRelease failed: %w", err)
	}

	dl.released = true
	return nil
}

// CalculateDescCount calculates the number of descriptors needed for a buffer
func CalculateDescCount(bufferSize uint64, pageSize uint16) uint64 {
	pageSizeU64 := uint64(pageSize)
	return (bufferSize + pageSizeU64 - 1) / pageSizeU64
}
