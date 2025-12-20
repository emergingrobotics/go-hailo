//go:build integration

package driver

import (
	"os"
	"sync"
	"testing"
	"time"
)

func skipIfNoDevice(t *testing.T) string {
	t.Helper()
	devices := []string{"/dev/hailo0", "/dev/hailo1"}
	for _, path := range devices {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	t.Skip("No Hailo device available")
	return ""
}

func TestOpenNonExistentDevice(t *testing.T) {
	_, err := OpenDevice("/dev/hailo_nonexistent_device_12345")
	if err == nil {
		t.Error("expected error when opening non-existent device")
	}

	hailoErr, ok := err.(*HailoError)
	if !ok {
		t.Fatalf("expected HailoError, got %T", err)
	}

	if hailoErr.Status != StatusNotFound {
		t.Errorf("expected StatusNotFound, got %v", hailoErr.Status)
	}
}

func TestOpenAndCloseDevice(t *testing.T) {
	path := skipIfNoDevice(t)

	dev, err := OpenDevice(path)
	if err != nil {
		t.Fatalf("failed to open device: %v", err)
	}
	defer dev.Close()

	if dev.Fd() < 0 {
		t.Error("expected valid file descriptor")
	}

	if dev.Path() != path {
		t.Errorf("expected path %s, got %s", path, dev.Path())
	}

	err = dev.Close()
	if err != nil {
		t.Errorf("failed to close device: %v", err)
	}

	if dev.Fd() != -1 {
		t.Error("expected fd to be -1 after close")
	}
}

func TestQueryDeviceProperties(t *testing.T) {
	path := skipIfNoDevice(t)

	dev, err := OpenDevice(path)
	if err != nil {
		t.Fatalf("failed to open device: %v", err)
	}
	defer dev.Close()

	props, err := dev.QueryDeviceProperties()
	if err != nil {
		t.Fatalf("failed to query device properties: %v", err)
	}

	// Verify reasonable values
	if props.DescMaxPageSize == 0 {
		t.Error("DescMaxPageSize should not be 0")
	}

	if props.DmaEnginesCount == 0 || props.DmaEnginesCount > 10 {
		t.Errorf("unexpected DmaEnginesCount: %d", props.DmaEnginesCount)
	}

	// For Hailo-8, board type should be 0
	if props.BoardType != BoardTypeHailo8 {
		t.Logf("BoardType = %d (expected Hailo8=0 for M.2 module)", props.BoardType)
	}

	t.Logf("Device properties: DescMaxPageSize=%d, BoardType=%d, DmaType=%d, DmaEngines=%d, FwLoaded=%v",
		props.DescMaxPageSize, props.BoardType, props.DmaType, props.DmaEnginesCount, props.IsFwLoaded)
}

func TestQueryDriverInfo(t *testing.T) {
	path := skipIfNoDevice(t)

	dev, err := OpenDevice(path)
	if err != nil {
		t.Fatalf("failed to open device: %v", err)
	}
	defer dev.Close()

	info, err := dev.QueryDriverInfo()
	if err != nil {
		t.Fatalf("failed to query driver info: %v", err)
	}

	// Verify driver version matches expected
	if info.MajorVersion != HailoDrvVerMajor {
		t.Errorf("driver major version = %d, expected %d", info.MajorVersion, HailoDrvVerMajor)
	}

	if info.MinorVersion != HailoDrvVerMinor {
		t.Logf("driver minor version = %d (expected %d)", info.MinorVersion, HailoDrvVerMinor)
	}

	t.Logf("Driver version: %d.%d.%d", info.MajorVersion, info.MinorVersion, info.RevisionVersion)
}

func TestDoubleClose(t *testing.T) {
	path := skipIfNoDevice(t)

	dev, err := OpenDevice(path)
	if err != nil {
		t.Fatalf("failed to open device: %v", err)
	}

	// First close should succeed
	err = dev.Close()
	if err != nil {
		t.Errorf("first close failed: %v", err)
	}

	// Second close should be safe (no-op)
	err = dev.Close()
	if err != nil {
		t.Errorf("second close should not fail: %v", err)
	}
}

func TestDeviceOperationsAfterClose(t *testing.T) {
	path := skipIfNoDevice(t)

	dev, err := OpenDevice(path)
	if err != nil {
		t.Fatalf("failed to open device: %v", err)
	}

	dev.Close()

	// Operations should fail after close
	_, err = dev.QueryDeviceProperties()
	if err == nil {
		t.Error("expected error when querying closed device")
	}
}

func TestScanDevices(t *testing.T) {
	devices, err := ScanDevices()
	if err != nil {
		t.Logf("scan returned error (may be expected): %v", err)
	}

	t.Logf("found %d device(s): %v", len(devices), devices)

	// If we found devices, verify they're valid paths
	for _, path := range devices {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("device path %s does not exist", path)
		}
	}
}

func TestBufferMapUnmap(t *testing.T) {
	path := skipIfNoDevice(t)

	dev, err := OpenDevice(path)
	if err != nil {
		t.Fatalf("failed to open device: %v", err)
	}
	defer dev.Close()

	// Allocate a test buffer (page-aligned)
	pageSize := 4096
	bufferSize := uint64(pageSize * 4)

	// This test requires a real buffer allocation
	// For now, just test that the function doesn't panic with invalid input
	_, err = dev.VdmaBufferMap(0, bufferSize, DmaToDevice, DmaUserPtrBuffer)
	if err == nil {
		t.Error("expected error when mapping null address")
	}
}

func TestBufferMapUnmapReal(t *testing.T) {
	path := skipIfNoDevice(t)

	dev, err := OpenDevice(path)
	if err != nil {
		t.Fatalf("failed to open device: %v", err)
	}
	defer dev.Close()

	// Allocate a real buffer
	bufferSize := uint64(4096)
	buf := make([]byte, bufferSize)

	// Map buffer for write (to device)
	handle, err := dev.VdmaBufferMap(
		uintptr(&buf[0]),
		bufferSize,
		DmaToDevice,
		DmaUserPtrBuffer,
	)
	if err != nil {
		t.Fatalf("VdmaBufferMap failed: %v", err)
	}

	t.Logf("Mapped buffer: handle=0x%016X", handle)

	// Unmap the buffer
	err = dev.VdmaBufferUnmap(handle)
	if err != nil {
		t.Fatalf("VdmaBufferUnmap failed: %v", err)
	}

	t.Log("Buffer unmapped successfully")
}

func TestBufferMapMultiple(t *testing.T) {
	path := skipIfNoDevice(t)

	dev, err := OpenDevice(path)
	if err != nil {
		t.Fatalf("failed to open device: %v", err)
	}
	defer dev.Close()

	// Allocate and map multiple buffers
	numBuffers := 4
	bufferSize := uint64(4096)
	buffers := make([][]byte, numBuffers)
	handles := make([]uint64, numBuffers)

	for i := 0; i < numBuffers; i++ {
		buffers[i] = make([]byte, bufferSize)
		handles[i], err = dev.VdmaBufferMap(
			uintptr(&buffers[i][0]),
			bufferSize,
			DmaBidirectional,
			DmaUserPtrBuffer,
		)
		if err != nil {
			// Clean up already mapped buffers
			for j := 0; j < i; j++ {
				dev.VdmaBufferUnmap(handles[j])
			}
			t.Fatalf("failed to map buffer %d: %v", i, err)
		}
		t.Logf("Mapped buffer %d: handle=0x%016X", i, handles[i])
	}

	// Unmap all buffers
	for i, h := range handles {
		err = dev.VdmaBufferUnmap(h)
		if err != nil {
			t.Errorf("failed to unmap buffer %d: %v", i, err)
		}
	}
}

func TestBufferSync(t *testing.T) {
	path := skipIfNoDevice(t)

	dev, err := OpenDevice(path)
	if err != nil {
		t.Fatalf("failed to open device: %v", err)
	}
	defer dev.Close()

	bufferSize := uint64(4096)
	buf := make([]byte, bufferSize)

	// Fill with test data
	for i := range buf {
		buf[i] = byte(i)
	}

	handle, err := dev.VdmaBufferMap(
		uintptr(&buf[0]),
		bufferSize,
		DmaBidirectional,
		DmaUserPtrBuffer,
	)
	if err != nil {
		t.Fatalf("VdmaBufferMap failed: %v", err)
	}
	defer dev.VdmaBufferUnmap(handle)

	// Sync for device (prepare for DMA transfer to device)
	err = dev.VdmaBufferSync(handle, SyncForDevice, 0, bufferSize)
	if err != nil {
		t.Fatalf("SyncForDevice failed: %v", err)
	}

	// Sync for CPU (after DMA transfer from device)
	err = dev.VdmaBufferSync(handle, SyncForCpu, 0, bufferSize)
	if err != nil {
		t.Fatalf("SyncForCpu failed: %v", err)
	}

	t.Log("Buffer sync succeeded")
}

func TestDescListCreateRelease(t *testing.T) {
	path := skipIfNoDevice(t)

	dev, err := OpenDevice(path)
	if err != nil {
		t.Fatalf("failed to open device: %v", err)
	}
	defer dev.Close()

	props, err := dev.QueryDeviceProperties()
	if err != nil {
		t.Fatalf("failed to query properties: %v", err)
	}

	descCount := uint64(64)
	pageSize := props.DescMaxPageSize

	handle, dmaAddr, err := dev.DescListCreate(descCount, pageSize, false)
	if err != nil {
		t.Fatalf("DescListCreate failed: %v", err)
	}

	t.Logf("Descriptor list: handle=%v, dma_addr=0x%016X", handle, dmaAddr)

	err = dev.DescListRelease(handle)
	if err != nil {
		t.Fatalf("DescListRelease failed: %v", err)
	}

	t.Log("Descriptor list released")
}

func TestDescListCircular(t *testing.T) {
	path := skipIfNoDevice(t)

	dev, err := OpenDevice(path)
	if err != nil {
		t.Fatalf("failed to open device: %v", err)
	}
	defer dev.Close()

	props, err := dev.QueryDeviceProperties()
	if err != nil {
		t.Fatalf("failed to query properties: %v", err)
	}

	descCount := uint64(128)
	pageSize := props.DescMaxPageSize

	// Create circular descriptor list
	handle, dmaAddr, err := dev.DescListCreate(descCount, pageSize, true)
	if err != nil {
		t.Fatalf("DescListCreate (circular) failed: %v", err)
	}

	t.Logf("Circular descriptor list: handle=%v, dma_addr=0x%016X", handle, dmaAddr)

	err = dev.DescListRelease(handle)
	if err != nil {
		t.Fatalf("DescListRelease failed: %v", err)
	}
}

func TestResetNnCore(t *testing.T) {
	path := skipIfNoDevice(t)

	dev, err := OpenDevice(path)
	if err != nil {
		t.Fatalf("failed to open device: %v", err)
	}
	defer dev.Close()

	err = dev.ResetNnCore()
	if err != nil {
		// Reset might fail if device is in use
		t.Logf("ResetNnCore returned: %v (may be expected if device is busy)", err)
	} else {
		t.Log("ResetNnCore succeeded")
	}
}

func TestConcurrentDeviceOpen(t *testing.T) {
	path := skipIfNoDevice(t)

	numGoroutines := 4
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			dev, err := OpenDevice(path)
			if err != nil {
				errors <- err
				return
			}
			defer dev.Close()

			// Query properties
			_, err = dev.QueryDeviceProperties()
			if err != nil {
				errors <- err
				return
			}

			time.Sleep(10 * time.Millisecond)
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent operation failed: %v", err)
	}
}

func TestDeviceReopen(t *testing.T) {
	path := skipIfNoDevice(t)

	for i := 0; i < 5; i++ {
		dev, err := OpenDevice(path)
		if err != nil {
			t.Fatalf("iteration %d: failed to open: %v", i, err)
		}

		props, err := dev.QueryDeviceProperties()
		if err != nil {
			dev.Close()
			t.Fatalf("iteration %d: failed to query: %v", i, err)
		}

		if props.DescMaxPageSize == 0 {
			dev.Close()
			t.Fatalf("iteration %d: invalid properties", i)
		}

		err = dev.Close()
		if err != nil {
			t.Fatalf("iteration %d: failed to close: %v", i, err)
		}
	}
}

// Benchmarks

func BenchmarkDeviceOpenClose(b *testing.B) {
	devices, _ := ScanDevices()
	if len(devices) == 0 {
		b.Skip("No Hailo device available")
	}

	path := devices[0]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dev, err := OpenDevice(path)
		if err != nil {
			b.Fatal(err)
		}
		dev.Close()
	}
}

func BenchmarkQueryDeviceProperties(b *testing.B) {
	devices, _ := ScanDevices()
	if len(devices) == 0 {
		b.Skip("No Hailo device available")
	}

	dev, err := OpenDevice(devices[0])
	if err != nil {
		b.Fatal(err)
	}
	defer dev.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := dev.QueryDeviceProperties()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVdmaBufferMapUnmap(b *testing.B) {
	devices, _ := ScanDevices()
	if len(devices) == 0 {
		b.Skip("No Hailo device available")
	}

	dev, err := OpenDevice(devices[0])
	if err != nil {
		b.Fatal(err)
	}
	defer dev.Close()

	bufSize := uint64(4096)
	buf := make([]byte, bufSize)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handle, err := dev.VdmaBufferMap(
			uintptr(&buf[0]),
			bufSize,
			DmaToDevice,
			DmaUserPtrBuffer,
		)
		if err != nil {
			b.Fatal(err)
		}
		dev.VdmaBufferUnmap(handle)
	}
}

func BenchmarkBufferSync(b *testing.B) {
	devices, _ := ScanDevices()
	if len(devices) == 0 {
		b.Skip("No Hailo device available")
	}

	dev, err := OpenDevice(devices[0])
	if err != nil {
		b.Fatal(err)
	}
	defer dev.Close()

	bufSize := uint64(4096)
	buf := make([]byte, bufSize)

	handle, err := dev.VdmaBufferMap(
		uintptr(&buf[0]),
		bufSize,
		DmaBidirectional,
		DmaUserPtrBuffer,
	)
	if err != nil {
		b.Fatal(err)
	}
	defer dev.VdmaBufferUnmap(handle)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dev.VdmaBufferSync(handle, SyncForDevice, 0, bufSize)
		dev.VdmaBufferSync(handle, SyncForCpu, 0, bufSize)
	}
}
