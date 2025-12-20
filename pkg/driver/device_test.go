//go:build integration

package driver

import (
	"os"
	"testing"
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
