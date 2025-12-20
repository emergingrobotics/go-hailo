package device

import "errors"

// Errors for device operations
var (
	ErrNoDevices       = errors.New("no Hailo devices found")
	ErrDeviceClosed    = errors.New("device is closed")
	ErrInvalidState    = errors.New("invalid network group state")
	ErrStillActivated  = errors.New("network group still activated")
	ErrNotConfigured   = errors.New("network group not configured")
	ErrAlreadyActivated = errors.New("network group already activated")
)
