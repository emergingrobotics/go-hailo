package device

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/anthropics/purple-hailo/pkg/driver"
)

// DeviceInfo contains discovered device information
type DeviceInfo struct {
	Path      string
	DeviceID  string
	BoardType driver.BoardType
}

// DeviceScanner scans for Hailo devices
type DeviceScanner struct {
	sysfsPath string
	devPath   string
}

// NewScanner creates a new device scanner
func NewScanner() *DeviceScanner {
	return &DeviceScanner{
		sysfsPath: "/sys/class/hailo_chardev",
		devPath:   "/dev",
	}
}

// Scan finds all Hailo devices
func (s *DeviceScanner) Scan() ([]DeviceInfo, error) {
	if s.sysfsPath == "" {
		s.sysfsPath = "/sys/class/hailo_chardev"
	}
	if s.devPath == "" {
		s.devPath = "/dev"
	}

	var devices []DeviceInfo

	// First try sysfs path
	entries, err := os.ReadDir(s.sysfsPath)
	if err == nil {
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
	}

	// If no devices found via sysfs, try direct device path scanning
	if len(devices) == 0 {
		for i := 0; i < 16; i++ {
			name := fmt.Sprintf("hailo%d", i)
			devPath := filepath.Join(s.devPath, name)
			if _, err := os.Stat(devPath); err == nil {
				devices = append(devices, DeviceInfo{
					Path:     devPath,
					DeviceID: name,
				})
			}
		}
	}

	return devices, nil
}

// ScanByType finds devices of a specific type
func (s *DeviceScanner) ScanByType(boardType driver.BoardType) ([]DeviceInfo, error) {
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

// Scan uses the default scanner to find all Hailo devices
func Scan() ([]DeviceInfo, error) {
	return NewScanner().Scan()
}

// isValidHailoDevicePath checks if a path is a valid Hailo device path
func isValidHailoDevicePath(path string) bool {
	if len(path) < 11 { // "/dev/hailo0" minimum
		return false
	}
	return len(path) >= 11 && path[:10] == "/dev/hailo"
}

// BoardTypeHailo8 is provided for backward compatibility
const BoardTypeHailo8 = driver.BoardTypeHailo8
