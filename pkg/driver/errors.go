package driver

import (
	"errors"
	"fmt"

	"golang.org/x/sys/unix"
)

// Status represents a Hailo operation status code
type Status int

// Hailo status codes matching hailort C library
const (
	StatusSuccess                    Status = 0
	StatusUninitialized              Status = 1
	StatusInvalidArgument            Status = 2
	StatusOutOfHostMemory            Status = 3
	StatusOutOfHostCmaMemory         Status = 4
	StatusTimeout                    Status = 5
	StatusInvalidOperation           Status = 6
	StatusNotFound                   Status = 7
	StatusStreamAbort                Status = 8
	StatusInternalFailure            Status = 9
	StatusDriverOperationFailed      Status = 10
	StatusDriverTimeout              Status = 11
	StatusDriverInterrupted          Status = 12
	StatusDriverInvalidIoctl         Status = 13
	StatusDriverWaitCanceled         Status = 14
	StatusConnectionRefused          Status = 15
	StatusInvalidHef                 Status = 16
	StatusHefNotCompatibleWithDevice Status = 17
	StatusFirmwareControlFailure     Status = 18
	StatusInvalidFrame               Status = 19
	StatusInvalidProtocolVersion     Status = 20
	StatusCommunicationClosed        Status = 21
)

var statusMessages = map[Status]string{
	StatusSuccess:                    "success",
	StatusUninitialized:              "uninitialized",
	StatusInvalidArgument:            "invalid argument",
	StatusOutOfHostMemory:            "out of host memory",
	StatusOutOfHostCmaMemory:         "out of host CMA memory",
	StatusTimeout:                    "timeout",
	StatusInvalidOperation:           "invalid operation",
	StatusNotFound:                   "not found",
	StatusStreamAbort:                "stream aborted",
	StatusInternalFailure:            "internal failure",
	StatusDriverOperationFailed:      "driver operation failed",
	StatusDriverTimeout:              "driver timeout",
	StatusDriverInterrupted:          "driver interrupted",
	StatusDriverInvalidIoctl:         "driver invalid ioctl (version mismatch)",
	StatusDriverWaitCanceled:         "driver wait canceled",
	StatusConnectionRefused:          "connection refused",
	StatusInvalidHef:                 "invalid HEF",
	StatusHefNotCompatibleWithDevice: "HEF not compatible with device",
	StatusFirmwareControlFailure:     "firmware control failure",
	StatusInvalidFrame:               "invalid frame",
	StatusInvalidProtocolVersion:     "invalid protocol version",
	StatusCommunicationClosed:        "communication closed",
}

// String returns the human-readable status message
func (s Status) String() string {
	if msg, ok := statusMessages[s]; ok {
		return msg
	}
	return fmt.Sprintf("unknown status (%d)", int(s))
}

// HailoError represents an error from the Hailo driver or runtime
type HailoError struct {
	Status  Status
	Context string
	Cause   error
}

// Error implements the error interface
func (e *HailoError) Error() string {
	if e.Context != "" {
		if e.Cause != nil {
			return fmt.Sprintf("%s: %s: %v", e.Context, e.Status.String(), e.Cause)
		}
		return fmt.Sprintf("%s: %s", e.Context, e.Status.String())
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Status.String(), e.Cause)
	}
	return e.Status.String()
}

// Unwrap returns the underlying cause
func (e *HailoError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches a target status
func (e *HailoError) Is(target error) bool {
	var hailoErr *HailoError
	if errors.As(target, &hailoErr) {
		return e.Status == hailoErr.Status
	}
	return false
}

// NewError creates a new HailoError with the given status
func NewError(status Status, context string) *HailoError {
	return &HailoError{
		Status:  status,
		Context: context,
	}
}

// NewErrorWithCause creates a new HailoError with an underlying cause
func NewErrorWithCause(status Status, context string, cause error) *HailoError {
	return &HailoError{
		Status:  status,
		Context: context,
		Cause:   cause,
	}
}

// ErrnoToStatus converts a Linux errno to a Hailo status
func ErrnoToStatus(errno unix.Errno) Status {
	switch errno {
	case unix.ENOBUFS:
		return StatusOutOfHostCmaMemory
	case unix.ENOMEM:
		return StatusOutOfHostMemory
	case unix.EFAULT:
		return StatusInvalidOperation
	case unix.ECONNRESET:
		return StatusStreamAbort
	case unix.ENOTTY:
		return StatusDriverInvalidIoctl
	case unix.ETIMEDOUT:
		return StatusDriverTimeout
	case unix.EINTR:
		return StatusDriverInterrupted
	case unix.ECONNREFUSED:
		return StatusConnectionRefused
	case unix.ECANCELED:
		return StatusDriverWaitCanceled
	case unix.ENOENT:
		return StatusNotFound
	case unix.EINVAL:
		return StatusInvalidArgument
	default:
		return StatusDriverOperationFailed
	}
}

// StatusFromErrno creates a HailoError from an errno
func StatusFromErrno(errno unix.Errno, context string) *HailoError {
	return &HailoError{
		Status:  ErrnoToStatus(errno),
		Context: context,
		Cause:   errno,
	}
}
