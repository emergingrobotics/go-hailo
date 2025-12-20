//go:build unit

package stream

import (
	"testing"
)

func TestPageSizeConstant(t *testing.T) {
	if PageSize != 4096 {
		t.Errorf("PageSize = %d, expected 4096", PageSize)
	}
}

func TestCalculateDescCount(t *testing.T) {
	tests := []struct {
		bufferSize uint64
		pageSize   uint16
		expected   uint64
	}{
		{4096, 4096, 1},
		{8192, 4096, 2},
		{4097, 4096, 2},
		{1, 4096, 1},
		{0, 4096, 0},
		{12288, 4096, 3},
	}

	for _, tt := range tests {
		result := CalculateDescCount(tt.bufferSize, tt.pageSize)
		if result != tt.expected {
			t.Errorf("CalculateDescCount(%d, %d) = %d, expected %d",
				tt.bufferSize, tt.pageSize, result, tt.expected)
		}
	}
}

// MockBuffer tests basic buffer operations without a real device
func TestBufferDataAccess(t *testing.T) {
	// Create a mock buffer for testing
	mockData := make([]byte, PageSize)
	for i := range mockData {
		mockData[i] = byte(i % 256)
	}

	// Test buffer operations
	buf := &Buffer{
		data:          mockData,
		size:          uint64(len(mockData)),
		allocatedSize: uint64(len(mockData)),
		pageAligned:   false,
	}

	if buf.Size() != uint64(len(mockData)) {
		t.Errorf("Size() = %d, expected %d", buf.Size(), len(mockData))
	}

	data := buf.Data()
	if len(data) != len(mockData) {
		t.Errorf("Data() len = %d, expected %d", len(data), len(mockData))
	}

	for i := range data {
		if data[i] != mockData[i] {
			t.Errorf("Data()[%d] = %d, expected %d", i, data[i], mockData[i])
			break
		}
	}
}

func TestBufferPoolCapacity(t *testing.T) {
	// Test pool capacity tracking without real device
	poolSize := 5
	pool := &BufferPool{
		pool: make(chan *Buffer, poolSize),
	}

	// Add mock buffers
	for i := 0; i < poolSize; i++ {
		pool.pool <- &Buffer{
			data: make([]byte, PageSize),
			size: PageSize,
		}
	}

	if pool.Available() != poolSize {
		t.Errorf("Available() = %d, expected %d", pool.Available(), poolSize)
	}

	// Take one
	<-pool.pool
	if pool.Available() != poolSize-1 {
		t.Errorf("After taking one, Available() = %d, expected %d", pool.Available(), poolSize-1)
	}
}

func TestBufferPoolTryGet(t *testing.T) {
	pool := &BufferPool{
		pool: make(chan *Buffer, 2),
	}

	// Should return nil when empty
	buf := pool.TryGet()
	if buf != nil {
		t.Error("TryGet() should return nil when pool is empty")
	}

	// Add a buffer
	mockBuf := &Buffer{
		data: make([]byte, PageSize),
		size: PageSize,
	}
	pool.pool <- mockBuf

	// Should now return the buffer
	buf = pool.TryGet()
	if buf == nil {
		t.Error("TryGet() should return buffer when pool is not empty")
	}
	if buf != mockBuf {
		t.Error("TryGet() returned wrong buffer")
	}
}
