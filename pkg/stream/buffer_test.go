//go:build unit

package stream

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

func TestAllocateAlignedBuffer(t *testing.T) {
	testCases := []struct {
		size      int
		alignment int
	}{
		{4096, 4096},   // Page aligned
		{1024, 64},     // Cache line aligned
		{100, 16},      // 16-byte aligned
		{8192, 4096},   // 2 pages
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			buf := AllocateAligned(tc.size, tc.alignment)
			if buf == nil {
				t.Fatal("allocation returned nil")
			}

			// Check alignment
			addr := uintptr(unsafe.Pointer(&buf[0]))
			if addr%uintptr(tc.alignment) != 0 {
				t.Errorf("buffer not aligned: addr=0x%X, alignment=%d",
					addr, tc.alignment)
			}

			// Check size
			if len(buf) != tc.size {
				t.Errorf("buffer size = %d, expected %d", len(buf), tc.size)
			}
		})
	}
}

func TestBufferReferenceCount(t *testing.T) {
	buf := &MappedBuffer{
		refs: 1,
	}

	// Acquire should increment
	buf.Acquire()
	if buf.refs != 2 {
		t.Errorf("refs = %d after Acquire, expected 2", buf.refs)
	}

	// Release should decrement
	remaining := buf.Release()
	if remaining != 1 {
		t.Errorf("Release returned %d, expected 1", remaining)
	}
	if buf.refs != 1 {
		t.Errorf("refs = %d after Release, expected 1", buf.refs)
	}

	// Final release
	remaining = buf.Release()
	if remaining != 0 {
		t.Errorf("final Release returned %d, expected 0", remaining)
	}
}

func TestBufferPoolAcquireRelease(t *testing.T) {
	pool := NewBufferPool(4096, 3)
	defer pool.Close()

	// Acquire all buffers
	var buffers []*PooledBuffer
	for i := 0; i < 3; i++ {
		buf, err := pool.Acquire()
		if err != nil {
			t.Fatalf("Acquire %d failed: %v", i, err)
		}
		buffers = append(buffers, buf)
	}

	// Pool should be exhausted
	if pool.Available() != 0 {
		t.Errorf("available = %d, expected 0", pool.Available())
	}

	// Release one
	pool.Release(buffers[0])
	if pool.Available() != 1 {
		t.Errorf("available = %d after release, expected 1", pool.Available())
	}

	// Should be able to acquire again
	buf, err := pool.Acquire()
	if err != nil {
		t.Fatalf("re-acquire failed: %v", err)
	}
	pool.Release(buf)
}

func TestBufferPoolExhaustion(t *testing.T) {
	pool := NewBufferPool(4096, 2)
	defer pool.Close()

	// Acquire all
	buf1, _ := pool.Acquire()
	buf2, _ := pool.Acquire()
	_ = buf1
	_ = buf2

	// Try to acquire with timeout
	done := make(chan bool)
	go func() {
		_, err := pool.AcquireWithTimeout(50 * time.Millisecond)
		if err == nil {
			t.Error("expected timeout error")
		}
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(200 * time.Millisecond):
		t.Error("AcquireWithTimeout didn't timeout")
	}
}

func TestBufferPoolConcurrent(t *testing.T) {
	pool := NewBufferPool(4096, 10)
	defer pool.Close()

	var wg sync.WaitGroup
	var acquireCount int64

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				buf, err := pool.AcquireWithTimeout(100 * time.Millisecond)
				if err != nil {
					continue
				}
				atomic.AddInt64(&acquireCount, 1)
				time.Sleep(time.Microsecond)
				pool.Release(buf)
			}
		}()
	}

	wg.Wait()

	if acquireCount == 0 {
		t.Error("no successful acquires")
	}
	t.Logf("successful acquires: %d", acquireCount)

	// All buffers should be returned
	if pool.Available() != 10 {
		t.Errorf("not all buffers returned: available = %d", pool.Available())
	}
}

func TestBufferPoolClose(t *testing.T) {
	pool := NewBufferPool(4096, 3)

	// Acquire one buffer
	buf, _ := pool.Acquire()

	// Close pool (should not panic)
	pool.Close()

	// Release after close (should not panic)
	pool.Release(buf)

	// Acquire after close should fail
	_, err := pool.Acquire()
	if err == nil {
		t.Error("Acquire after Close should fail")
	}
}

func TestPageAlignment(t *testing.T) {
	pageSize := 4096

	tests := []struct {
		size     int
		expected int
	}{
		{1, pageSize},
		{4096, 4096},
		{4097, 8192},
		{8192, 8192},
		{10000, 12288},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			aligned := AlignToPage(tt.size)
			if aligned != tt.expected {
				t.Errorf("AlignToPage(%d) = %d, expected %d",
					tt.size, aligned, tt.expected)
			}
		})
	}
}

func TestBufferZeroInitialized(t *testing.T) {
	buf := AllocateAligned(1024, 64)

	for i, b := range buf {
		if b != 0 {
			t.Errorf("buffer[%d] = %d, expected 0", i, b)
			break
		}
	}
}

// MappedBuffer represents a buffer mapped for DMA
type MappedBuffer struct {
	Data    []byte
	Handle  uint64
	Address uintptr
	Size    uint64
	refs    int32
}

func (b *MappedBuffer) Acquire() {
	atomic.AddInt32(&b.refs, 1)
}

func (b *MappedBuffer) Release() int32 {
	return atomic.AddInt32(&b.refs, -1)
}

// PooledBuffer is a buffer from a pool
type PooledBuffer struct {
	*MappedBuffer
	pool *BufferPool
}

// BufferPool manages pre-allocated buffers
type BufferPool struct {
	buffers  chan *PooledBuffer
	size     int
	count    int
	closed   bool
	mu       sync.RWMutex
}

func NewBufferPool(size, count int) *BufferPool {
	pool := &BufferPool{
		buffers: make(chan *PooledBuffer, count),
		size:    size,
		count:   count,
	}

	for i := 0; i < count; i++ {
		buf := &PooledBuffer{
			MappedBuffer: &MappedBuffer{
				Data: make([]byte, size),
				refs: 1,
			},
			pool: pool,
		}
		pool.buffers <- buf
	}

	return pool
}

func (p *BufferPool) Acquire() (*PooledBuffer, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, ErrPoolClosed
	}
	p.mu.RUnlock()

	select {
	case buf := <-p.buffers:
		return buf, nil
	default:
		return nil, ErrPoolExhausted
	}
}

func (p *BufferPool) AcquireWithTimeout(timeout time.Duration) (*PooledBuffer, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, ErrPoolClosed
	}
	p.mu.RUnlock()

	select {
	case buf := <-p.buffers:
		return buf, nil
	case <-time.After(timeout):
		return nil, ErrPoolTimeout
	}
}

func (p *BufferPool) Release(buf *PooledBuffer) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return
	}

	select {
	case p.buffers <- buf:
	default:
		// Pool is full, discard
	}
}

func (p *BufferPool) Available() int {
	return len(p.buffers)
}

func (p *BufferPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}
	p.closed = true
	close(p.buffers)
}

// AllocateAligned allocates an aligned buffer
func AllocateAligned(size, alignment int) []byte {
	// Allocate extra bytes to ensure alignment
	buf := make([]byte, size+alignment)
	addr := uintptr(unsafe.Pointer(&buf[0]))
	alignmentUintptr := uintptr(alignment)
	remainder := addr % alignmentUintptr
	offset := 0
	if remainder != 0 {
		offset = int(alignmentUintptr - remainder)
	}
	return buf[offset : offset+size]
}

// AlignToPage rounds up to page boundary
func AlignToPage(size int) int {
	const pageSize = 4096
	return ((size + pageSize - 1) / pageSize) * pageSize
}

// Errors
type bufferError string

func (e bufferError) Error() string { return string(e) }

const (
	ErrPoolClosed    = bufferError("buffer pool is closed")
	ErrPoolExhausted = bufferError("buffer pool exhausted")
	ErrPoolTimeout   = bufferError("buffer pool acquire timeout")
)
