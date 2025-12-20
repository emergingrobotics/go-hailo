package stream

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/anthropics/purple-hailo/pkg/driver"
	"golang.org/x/sys/unix"
)

// PageSize is the system page size (typically 4096 bytes)
const PageSize = 4096

// Buffer represents a DMA-capable buffer
type Buffer struct {
	data          []byte
	size          uint64
	mappedHandle  uint64
	direction     driver.DmaDataDirection
	device        *driver.DeviceFile
	mu            sync.Mutex
	mapped        bool
	pageAligned   bool
	allocatedSize uint64 // includes alignment padding
}

// AllocateBuffer allocates a page-aligned buffer for DMA
func AllocateBuffer(dev *driver.DeviceFile, size uint64, direction driver.DmaDataDirection) (*Buffer, error) {
	if size == 0 {
		return nil, fmt.Errorf("buffer size cannot be zero")
	}

	// Round up to page size for alignment
	alignedSize := ((size + PageSize - 1) / PageSize) * PageSize

	// Use mmap for page-aligned allocation
	data, err := unix.Mmap(-1, 0, int(alignedSize),
		unix.PROT_READ|unix.PROT_WRITE,
		unix.MAP_PRIVATE|unix.MAP_ANONYMOUS)
	if err != nil {
		return nil, fmt.Errorf("mmap failed: %w", err)
	}

	buf := &Buffer{
		data:          data[:size], // Return only requested size
		size:          size,
		direction:     direction,
		device:        dev,
		pageAligned:   true,
		allocatedSize: alignedSize,
	}

	// Map the buffer for DMA
	handle, err := dev.VdmaBufferMap(
		uintptr(unsafe.Pointer(&data[0])),
		alignedSize,
		direction,
		driver.DmaUserPtrBuffer,
	)
	if err != nil {
		unix.Munmap(data)
		return nil, fmt.Errorf("VdmaBufferMap failed: %w", err)
	}

	buf.mappedHandle = handle
	buf.mapped = true

	return buf, nil
}

// WrapBuffer wraps an existing byte slice for DMA
// The slice must be page-aligned for proper DMA operation
func WrapBuffer(dev *driver.DeviceFile, data []byte, direction driver.DmaDataDirection) (*Buffer, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("buffer cannot be empty")
	}

	addr := uintptr(unsafe.Pointer(&data[0]))
	if addr%PageSize != 0 {
		return nil, fmt.Errorf("buffer is not page-aligned")
	}

	handle, err := dev.VdmaBufferMap(
		addr,
		uint64(len(data)),
		direction,
		driver.DmaUserPtrBuffer,
	)
	if err != nil {
		return nil, fmt.Errorf("VdmaBufferMap failed: %w", err)
	}

	return &Buffer{
		data:          data,
		size:          uint64(len(data)),
		mappedHandle:  handle,
		direction:     direction,
		device:        dev,
		mapped:        true,
		pageAligned:   false, // external buffer, we don't manage it
		allocatedSize: uint64(len(data)),
	}, nil
}

// Data returns the buffer data
func (b *Buffer) Data() []byte {
	return b.data
}

// Size returns the usable buffer size
func (b *Buffer) Size() uint64 {
	return b.size
}

// Handle returns the DMA mapped handle
func (b *Buffer) Handle() uint64 {
	return b.mappedHandle
}

// Direction returns the DMA direction
func (b *Buffer) Direction() driver.DmaDataDirection {
	return b.direction
}

// SyncForDevice synchronizes buffer for device access (before DMA to device)
func (b *Buffer) SyncForDevice() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.mapped {
		return fmt.Errorf("buffer not mapped")
	}

	return b.device.VdmaBufferSync(b.mappedHandle, driver.SyncForDevice, 0, b.allocatedSize)
}

// SyncForCPU synchronizes buffer for CPU access (after DMA from device)
func (b *Buffer) SyncForCPU() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.mapped {
		return fmt.Errorf("buffer not mapped")
	}

	return b.device.VdmaBufferSync(b.mappedHandle, driver.SyncForCpu, 0, b.allocatedSize)
}

// Close releases the buffer resources
func (b *Buffer) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.mapped {
		err := b.device.VdmaBufferUnmap(b.mappedHandle)
		if err != nil {
			return fmt.Errorf("VdmaBufferUnmap failed: %w", err)
		}
		b.mapped = false
	}

	// Only munmap if we allocated the memory
	if b.pageAligned && len(b.data) > 0 {
		// Need to use the original allocated size for munmap
		originalData := unsafe.Slice(&b.data[0], int(b.allocatedSize))
		if err := unix.Munmap(originalData); err != nil {
			return fmt.Errorf("munmap failed: %w", err)
		}
		b.data = nil
	}

	return nil
}

// BufferPool manages a pool of reusable buffers
type BufferPool struct {
	device    *driver.DeviceFile
	bufSize   uint64
	direction driver.DmaDataDirection
	pool      chan *Buffer
	mu        sync.Mutex
	closed    bool
}

// NewBufferPool creates a new buffer pool
func NewBufferPool(dev *driver.DeviceFile, bufSize uint64, poolSize int, direction driver.DmaDataDirection) (*BufferPool, error) {
	if poolSize <= 0 {
		return nil, fmt.Errorf("pool size must be positive")
	}

	bp := &BufferPool{
		device:    dev,
		bufSize:   bufSize,
		direction: direction,
		pool:      make(chan *Buffer, poolSize),
	}

	// Pre-allocate buffers
	for i := 0; i < poolSize; i++ {
		buf, err := AllocateBuffer(dev, bufSize, direction)
		if err != nil {
			// Clean up already allocated buffers
			bp.Close()
			return nil, fmt.Errorf("failed to allocate buffer %d: %w", i, err)
		}
		bp.pool <- buf
	}

	return bp, nil
}

// Get gets a buffer from the pool (blocks until available)
func (bp *BufferPool) Get() (*Buffer, error) {
	bp.mu.Lock()
	if bp.closed {
		bp.mu.Unlock()
		return nil, fmt.Errorf("pool is closed")
	}
	bp.mu.Unlock()

	buf := <-bp.pool
	return buf, nil
}

// TryGet tries to get a buffer from the pool without blocking
func (bp *BufferPool) TryGet() *Buffer {
	select {
	case buf := <-bp.pool:
		return buf
	default:
		return nil
	}
}

// Put returns a buffer to the pool
func (bp *BufferPool) Put(buf *Buffer) {
	bp.mu.Lock()
	if bp.closed {
		bp.mu.Unlock()
		buf.Close()
		return
	}
	bp.mu.Unlock()

	select {
	case bp.pool <- buf:
	default:
		// Pool is full, close the extra buffer
		buf.Close()
	}
}

// Close closes the buffer pool and releases all buffers
func (bp *BufferPool) Close() error {
	bp.mu.Lock()
	if bp.closed {
		bp.mu.Unlock()
		return nil
	}
	bp.closed = true
	bp.mu.Unlock()

	close(bp.pool)

	var lastErr error
	for buf := range bp.pool {
		if err := buf.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// Available returns the number of available buffers in the pool
func (bp *BufferPool) Available() int {
	return len(bp.pool)
}
