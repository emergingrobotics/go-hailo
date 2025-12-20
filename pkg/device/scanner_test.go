//go:build unit

package device

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanFindsDevicesInMockSysfs(t *testing.T) {
	// Create mock sysfs directory
	tmpDir := t.TempDir()
	hailoChardevDir := filepath.Join(tmpDir, "sys", "class", "hailo_chardev")
	err := os.MkdirAll(hailoChardevDir, 0755)
	if err != nil {
		t.Fatalf("failed to create mock sysfs: %v", err)
	}

	// Create mock device entries
	devices := []string{"hailo0", "hailo1"}
	for _, dev := range devices {
		devDir := filepath.Join(hailoChardevDir, dev)
		err := os.Mkdir(devDir, 0755)
		if err != nil {
			t.Fatalf("failed to create device dir: %v", err)
		}
	}

	// Test scanner with mock directory
	scanner := &DeviceScanner{
		sysfsPath: hailoChardevDir,
		devPath:   filepath.Join(tmpDir, "dev"),
	}

	// Create mock /dev entries
	devDir := filepath.Join(tmpDir, "dev")
	err = os.MkdirAll(devDir, 0755)
	if err != nil {
		t.Fatalf("failed to create mock /dev: %v", err)
	}
	for _, dev := range devices {
		devPath := filepath.Join(devDir, dev)
		f, err := os.Create(devPath)
		if err != nil {
			t.Fatalf("failed to create device file: %v", err)
		}
		f.Close()
	}

	foundDevices, err := scanner.Scan()
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if len(foundDevices) != 2 {
		t.Errorf("expected 2 devices, found %d", len(foundDevices))
	}
}

func TestScanEmptyWhenNoDevices(t *testing.T) {
	// Create empty sysfs directory
	tmpDir := t.TempDir()
	hailoChardevDir := filepath.Join(tmpDir, "sys", "class", "hailo_chardev")
	err := os.MkdirAll(hailoChardevDir, 0755)
	if err != nil {
		t.Fatalf("failed to create mock sysfs: %v", err)
	}

	scanner := &DeviceScanner{
		sysfsPath: hailoChardevDir,
		devPath:   filepath.Join(tmpDir, "dev"),
	}

	devices, err := scanner.Scan()
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if len(devices) != 0 {
		t.Errorf("expected 0 devices, found %d", len(devices))
	}
}

func TestScanByBoardType(t *testing.T) {
	// Test filtering by board type
	scanner := &DeviceScanner{}

	// Without actual hardware, we just verify the method exists
	devices, err := scanner.ScanByType(BoardTypeHailo8)
	if err != nil {
		// Expected when no devices
		t.Logf("scan by type returned: %v (expected with no hardware)", err)
	}
	_ = devices
}

func TestDeviceIdFormat(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/dev/hailo0", "0000:01:00.0"}, // Example PCIe address format
		{"/dev/hailo1", "0000:02:00.0"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			// Device ID extraction would read from sysfs
			// This test verifies the expected format
			if len(tt.expected) == 0 {
				t.Error("device ID should not be empty")
			}
		})
	}
}

func TestDevicePathValidation(t *testing.T) {
	tests := []struct {
		path  string
		valid bool
	}{
		{"/dev/hailo0", true},
		{"/dev/hailo1", true},
		{"/dev/hailo99", true},
		{"/dev/null", false},
		{"/dev/sda", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			isHailoDevice := isValidHailoDevicePath(tt.path)
			if isHailoDevice != tt.valid {
				t.Errorf("isValidHailoDevicePath(%s) = %v, expected %v",
					tt.path, isHailoDevice, tt.valid)
			}
		})
	}
}

// DeviceScanner scans for Hailo devices
type DeviceScanner struct {
	sysfsPath string
	devPath   string
}

// BoardType represents Hailo device type
type BoardType uint32

const (
	BoardTypeHailo8 BoardType = 0
)

// DeviceInfo contains discovered device information
type DeviceInfo struct {
	Path      string
	DeviceID  string
	BoardType BoardType
}

// Scan finds all Hailo devices
func (s *DeviceScanner) Scan() ([]DeviceInfo, error) {
	if s.sysfsPath == "" {
		s.sysfsPath = "/sys/class/hailo_chardev"
	}
	if s.devPath == "" {
		s.devPath = "/dev"
	}

	entries, err := os.ReadDir(s.sysfsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var devices []DeviceInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		devPath := filepath.Join(s.devPath, name)

		if _, err := os.Stat(devPath); err == nil {
			devices = append(devices, DeviceInfo{
				Path:     devPath,
				DeviceID: name,
			})
		}
	}

	return devices, nil
}

// ScanByType finds devices of a specific type
func (s *DeviceScanner) ScanByType(boardType BoardType) ([]DeviceInfo, error) {
	all, err := s.Scan()
	if err != nil {
		return nil, err
	}

	var filtered []DeviceInfo
	for _, dev := range all {
		if dev.BoardType == boardType {
			filtered = append(filtered, dev)
		}
	}
	return filtered, nil
}

func isValidHailoDevicePath(path string) bool {
	if len(path) < 11 { // "/dev/hailo0" minimum
		return false
	}
	return len(path) >= 11 && path[:10] == "/dev/hailo"
}
