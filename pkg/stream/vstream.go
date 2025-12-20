package stream

import (
	"fmt"
	"sync"
	"time"

	"github.com/anthropics/purple-hailo/pkg/driver"
	"github.com/anthropics/purple-hailo/pkg/hef"
)

// Errors for stream operations
var (
	ErrStreamClosed   = fmt.Errorf("stream is closed")
	ErrTimeout        = fmt.Errorf("operation timed out")
	ErrInvalidData    = fmt.Errorf("invalid data size")
	ErrBufferNotReady = fmt.Errorf("buffer not ready")
)

// VStreamInfo contains stream configuration
type VStreamInfo struct {
	Name      string
	FrameSize uint64
	Height    uint32
	Width     uint32
	Channels  uint32
	Format    hef.FormatType
	QuantInfo hef.QuantInfo
}

// InputVStream is for writing data to the device
type InputVStream struct {
	info       VStreamInfo
	buffer     *Buffer
	descList   *DescriptorList
	channel    *VdmaChannel
	device     *driver.DeviceFile
	mu         sync.Mutex
	closed     bool
	timeout    time.Duration
	batchSize  uint32
	queueDepth int
}

// InputVStreamConfig holds configuration for creating an input VStream
type InputVStreamConfig struct {
	Info       VStreamInfo
	Device     *driver.DeviceFile
	Channel    *VdmaChannel
	Timeout    time.Duration
	BatchSize  uint32
	QueueDepth int
}

// NewInputVStream creates a new input VStream
func NewInputVStream(cfg InputVStreamConfig) (*InputVStream, error) {
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 1
	}
	if cfg.QueueDepth == 0 {
		cfg.QueueDepth = 2
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Second
	}

	bufferSize := cfg.Info.FrameSize * uint64(cfg.BatchSize)

	// Allocate buffer for input (host to device)
	buffer, err := AllocateBuffer(cfg.Device, bufferSize, driver.DmaToDevice)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate buffer: %w", err)
	}

	// Query device properties to get page size
	props, err := cfg.Device.QueryDeviceProperties()
	if err != nil {
		buffer.Close()
		return nil, fmt.Errorf("failed to query device properties: %w", err)
	}

	// Calculate descriptor count
	descCount := CalculateDescCount(bufferSize, props.DescMaxPageSize)

	// Create descriptor list
	descList, err := CreateDescriptorList(cfg.Device, descCount, props.DescMaxPageSize, false)
	if err != nil {
		buffer.Close()
		return nil, fmt.Errorf("failed to create descriptor list: %w", err)
	}

	return &InputVStream{
		info:       cfg.Info,
		buffer:     buffer,
		descList:   descList,
		channel:    cfg.Channel,
		device:     cfg.Device,
		timeout:    cfg.Timeout,
		batchSize:  cfg.BatchSize,
		queueDepth: cfg.QueueDepth,
	}, nil
}

// Info returns the stream information
func (vs *InputVStream) Info() VStreamInfo {
	return vs.info
}

// FrameSize returns the size of a single frame
func (vs *InputVStream) FrameSize() uint64 {
	return vs.info.FrameSize
}

// Write writes a frame to the device
func (vs *InputVStream) Write(data []byte) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if vs.closed {
		return ErrStreamClosed
	}

	expectedSize := vs.info.FrameSize * uint64(vs.batchSize)
	if uint64(len(data)) != expectedSize {
		return fmt.Errorf("%w: expected %d bytes, got %d", ErrInvalidData, expectedSize, len(data))
	}

	// Copy data to buffer
	copy(vs.buffer.Data(), data)

	// Sync buffer to device
	if err := vs.buffer.SyncForDevice(); err != nil {
		return fmt.Errorf("buffer sync failed: %w", err)
	}

	// Program descriptor list
	err := vs.descList.Program(
		vs.buffer,
		vs.channel.ChannelIndex(),
		vs.batchSize,
		0,
		true,
		driver.InterruptsDomainDevice,
	)
	if err != nil {
		return fmt.Errorf("failed to program descriptor list: %w", err)
	}

	// Launch transfer
	err = vs.channel.LaunchTransfer(
		vs.descList,
		vs.buffer,
		0,
		true,
		driver.InterruptsDomainNone,
		driver.InterruptsDomainDevice,
	)
	if err != nil {
		return fmt.Errorf("failed to launch transfer: %w", err)
	}

	return nil
}

// WriteAsync writes data asynchronously
func (vs *InputVStream) WriteAsync(data []byte) error {
	// For now, async write is same as sync write
	// TODO: implement proper async with callback
	return vs.Write(data)
}

// Flush waits for pending writes to complete
func (vs *InputVStream) Flush() error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if vs.closed {
		return ErrStreamClosed
	}

	return vs.channel.WaitForInterrupt()
}

// Close closes the input stream
func (vs *InputVStream) Close() error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if vs.closed {
		return nil
	}

	vs.closed = true

	var lastErr error

	if err := vs.descList.Release(); err != nil {
		lastErr = err
	}

	if err := vs.buffer.Close(); err != nil {
		lastErr = err
	}

	return lastErr
}

// OutputVStream is for reading data from the device
type OutputVStream struct {
	info       VStreamInfo
	buffer     *Buffer
	descList   *DescriptorList
	channel    *VdmaChannel
	device     *driver.DeviceFile
	mu         sync.Mutex
	closed     bool
	timeout    time.Duration
	batchSize  uint32
	queueDepth int
	pending    bool // whether a read is pending
}

// OutputVStreamConfig holds configuration for creating an output VStream
type OutputVStreamConfig struct {
	Info       VStreamInfo
	Device     *driver.DeviceFile
	Channel    *VdmaChannel
	Timeout    time.Duration
	BatchSize  uint32
	QueueDepth int
}

// NewOutputVStream creates a new output VStream
func NewOutputVStream(cfg OutputVStreamConfig) (*OutputVStream, error) {
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 1
	}
	if cfg.QueueDepth == 0 {
		cfg.QueueDepth = 2
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Second
	}

	bufferSize := cfg.Info.FrameSize * uint64(cfg.BatchSize)

	// Allocate buffer for output (device to host)
	buffer, err := AllocateBuffer(cfg.Device, bufferSize, driver.DmaFromDevice)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate buffer: %w", err)
	}

	// Query device properties to get page size
	props, err := cfg.Device.QueryDeviceProperties()
	if err != nil {
		buffer.Close()
		return nil, fmt.Errorf("failed to query device properties: %w", err)
	}

	// Calculate descriptor count
	descCount := CalculateDescCount(bufferSize, props.DescMaxPageSize)

	// Create descriptor list
	descList, err := CreateDescriptorList(cfg.Device, descCount, props.DescMaxPageSize, false)
	if err != nil {
		buffer.Close()
		return nil, fmt.Errorf("failed to create descriptor list: %w", err)
	}

	return &OutputVStream{
		info:       cfg.Info,
		buffer:     buffer,
		descList:   descList,
		channel:    cfg.Channel,
		device:     cfg.Device,
		timeout:    cfg.Timeout,
		batchSize:  cfg.BatchSize,
		queueDepth: cfg.QueueDepth,
	}, nil
}

// Info returns the stream information
func (vs *OutputVStream) Info() VStreamInfo {
	return vs.info
}

// FrameSize returns the size of a single frame
func (vs *OutputVStream) FrameSize() uint64 {
	return vs.info.FrameSize
}

// StartRead prepares for reading output
func (vs *OutputVStream) StartRead() error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if vs.closed {
		return ErrStreamClosed
	}

	if vs.pending {
		return fmt.Errorf("read already pending")
	}

	// Program descriptor list
	err := vs.descList.Program(
		vs.buffer,
		vs.channel.ChannelIndex(),
		vs.batchSize,
		0,
		true,
		driver.InterruptsDomainHost,
	)
	if err != nil {
		return fmt.Errorf("failed to program descriptor list: %w", err)
	}

	// Launch transfer
	err = vs.channel.LaunchTransfer(
		vs.descList,
		vs.buffer,
		0,
		true,
		driver.InterruptsDomainNone,
		driver.InterruptsDomainHost,
	)
	if err != nil {
		return fmt.Errorf("failed to launch transfer: %w", err)
	}

	vs.pending = true
	return nil
}

// Read reads a frame from the device (blocking)
func (vs *OutputVStream) Read() ([]byte, error) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if vs.closed {
		return nil, ErrStreamClosed
	}

	// If no read is pending, start one
	if !vs.pending {
		// Program descriptor list
		err := vs.descList.Program(
			vs.buffer,
			vs.channel.ChannelIndex(),
			vs.batchSize,
			0,
			true,
			driver.InterruptsDomainHost,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to program descriptor list: %w", err)
		}

		// Launch transfer
		err = vs.channel.LaunchTransfer(
			vs.descList,
			vs.buffer,
			0,
			true,
			driver.InterruptsDomainNone,
			driver.InterruptsDomainHost,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to launch transfer: %w", err)
		}
	}

	// Wait for transfer completion
	if err := vs.channel.WaitForInterrupt(); err != nil {
		vs.pending = false
		return nil, fmt.Errorf("wait for interrupt failed: %w", err)
	}

	vs.pending = false

	// Sync buffer from device
	if err := vs.buffer.SyncForCPU(); err != nil {
		return nil, fmt.Errorf("buffer sync failed: %w", err)
	}

	// Copy data from buffer
	dataSize := vs.info.FrameSize * uint64(vs.batchSize)
	result := make([]byte, dataSize)
	copy(result, vs.buffer.Data()[:dataSize])

	return result, nil
}

// ReadInto reads a frame into the provided buffer
func (vs *OutputVStream) ReadInto(dst []byte) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if vs.closed {
		return ErrStreamClosed
	}

	expectedSize := vs.info.FrameSize * uint64(vs.batchSize)
	if uint64(len(dst)) < expectedSize {
		return fmt.Errorf("%w: buffer too small, need %d bytes", ErrInvalidData, expectedSize)
	}

	// Wait for transfer completion if pending
	if vs.pending {
		if err := vs.channel.WaitForInterrupt(); err != nil {
			vs.pending = false
			return fmt.Errorf("wait for interrupt failed: %w", err)
		}
		vs.pending = false
	}

	// Sync buffer from device
	if err := vs.buffer.SyncForCPU(); err != nil {
		return fmt.Errorf("buffer sync failed: %w", err)
	}

	// Copy data
	copy(dst, vs.buffer.Data()[:expectedSize])

	return nil
}

// Close closes the output stream
func (vs *OutputVStream) Close() error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if vs.closed {
		return nil
	}

	vs.closed = true

	var lastErr error

	if err := vs.descList.Release(); err != nil {
		lastErr = err
	}

	if err := vs.buffer.Close(); err != nil {
		lastErr = err
	}

	return lastErr
}
