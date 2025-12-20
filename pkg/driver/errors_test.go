//go:build unit

package driver

import (
	"errors"
	"testing"

	"golang.org/x/sys/unix"
)

func TestAllStatusCodesHaveMessages(t *testing.T) {
	statuses := []Status{
		StatusSuccess,
		StatusUninitialized,
		StatusInvalidArgument,
		StatusOutOfHostMemory,
		StatusOutOfHostCmaMemory,
		StatusTimeout,
		StatusInvalidOperation,
		StatusNotFound,
		StatusStreamAbort,
		StatusInternalFailure,
		StatusDriverOperationFailed,
		StatusDriverTimeout,
		StatusDriverInterrupted,
		StatusDriverInvalidIoctl,
		StatusDriverWaitCanceled,
		StatusConnectionRefused,
		StatusInvalidHef,
		StatusHefNotCompatibleWithDevice,
		StatusFirmwareControlFailure,
		StatusInvalidFrame,
		StatusInvalidProtocolVersion,
		StatusCommunicationClosed,
	}

	for _, status := range statuses {
		msg := status.String()
		if msg == "" {
			t.Errorf("status %d has empty message", status)
		}
		if len(msg) >= 8 && msg[:8] == "unknown " {
			t.Errorf("status %d has no defined message: %s", status, msg)
		}
	}
}

func TestStatusStringReturnsUnknownForUndefinedStatus(t *testing.T) {
	unknownStatus := Status(9999)
	msg := unknownStatus.String()
	if msg != "unknown status (9999)" {
		t.Errorf("expected 'unknown status (9999)', got '%s'", msg)
	}
}

func TestHailoErrorImplementsError(t *testing.T) {
	var err error = &HailoError{
		Status:  StatusInvalidArgument,
		Context: "test context",
	}

	if err.Error() == "" {
		t.Error("Error() returned empty string")
	}
}

func TestHailoErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      *HailoError
		expected string
	}{
		{
			name: "status only",
			err: &HailoError{
				Status: StatusInvalidArgument,
			},
			expected: "invalid argument",
		},
		{
			name: "with context",
			err: &HailoError{
				Status:  StatusInvalidArgument,
				Context: "opening device",
			},
			expected: "opening device: invalid argument",
		},
		{
			name: "with cause",
			err: &HailoError{
				Status: StatusDriverOperationFailed,
				Cause:  unix.ENOENT,
			},
			expected: "driver operation failed: no such file or directory",
		},
		{
			name: "with context and cause",
			err: &HailoError{
				Status:  StatusDriverOperationFailed,
				Context: "opening device",
				Cause:   unix.ENOENT,
			},
			expected: "opening device: driver operation failed: no such file or directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, got)
			}
		})
	}
}

func TestHailoErrorUnwrap(t *testing.T) {
	cause := unix.ENOENT
	err := &HailoError{
		Status: StatusNotFound,
		Cause:  cause,
	}

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Unwrap() returned %v, expected %v", unwrapped, cause)
	}
}

func TestHailoErrorUnwrapNil(t *testing.T) {
	err := &HailoError{
		Status: StatusNotFound,
	}

	unwrapped := err.Unwrap()
	if unwrapped != nil {
		t.Errorf("Unwrap() returned %v, expected nil", unwrapped)
	}
}

func TestHailoErrorIs(t *testing.T) {
	err1 := &HailoError{Status: StatusInvalidArgument}
	err2 := &HailoError{Status: StatusInvalidArgument}
	err3 := &HailoError{Status: StatusTimeout}

	if !errors.Is(err1, err2) {
		t.Error("errors.Is should return true for same status")
	}

	if errors.Is(err1, err3) {
		t.Error("errors.Is should return false for different status")
	}
}

func TestNewError(t *testing.T) {
	err := NewError(StatusTimeout, "waiting for device")

	if err.Status != StatusTimeout {
		t.Errorf("expected status %d, got %d", StatusTimeout, err.Status)
	}
	if err.Context != "waiting for device" {
		t.Errorf("expected context 'waiting for device', got '%s'", err.Context)
	}
	if err.Cause != nil {
		t.Error("expected nil cause")
	}
}

func TestNewErrorWithCause(t *testing.T) {
	cause := unix.ETIMEDOUT
	err := NewErrorWithCause(StatusDriverTimeout, "ioctl", cause)

	if err.Status != StatusDriverTimeout {
		t.Errorf("expected status %d, got %d", StatusDriverTimeout, err.Status)
	}
	if err.Context != "ioctl" {
		t.Errorf("expected context 'ioctl', got '%s'", err.Context)
	}
	if err.Cause != cause {
		t.Errorf("expected cause %v, got %v", cause, err.Cause)
	}
}

func TestErrnoToStatus(t *testing.T) {
	tests := []struct {
		errno    unix.Errno
		expected Status
	}{
		{unix.ENOBUFS, StatusOutOfHostCmaMemory},
		{unix.ENOMEM, StatusOutOfHostMemory},
		{unix.EFAULT, StatusInvalidOperation},
		{unix.ECONNRESET, StatusStreamAbort},
		{unix.ENOTTY, StatusDriverInvalidIoctl},
		{unix.ETIMEDOUT, StatusDriverTimeout},
		{unix.EINTR, StatusDriverInterrupted},
		{unix.ECONNREFUSED, StatusConnectionRefused},
		{unix.ECANCELED, StatusDriverWaitCanceled},
		{unix.ENOENT, StatusNotFound},
		{unix.EINVAL, StatusInvalidArgument},
		{unix.EPERM, StatusDriverOperationFailed}, // unmapped errno
	}

	for _, tt := range tests {
		t.Run(tt.errno.Error(), func(t *testing.T) {
			got := ErrnoToStatus(tt.errno)
			if got != tt.expected {
				t.Errorf("ErrnoToStatus(%v) = %d, expected %d", tt.errno, got, tt.expected)
			}
		})
	}
}

func TestStatusFromErrno(t *testing.T) {
	err := StatusFromErrno(unix.ETIMEDOUT, "waiting for interrupt")

	if err.Status != StatusDriverTimeout {
		t.Errorf("expected StatusDriverTimeout, got %d", err.Status)
	}
	if err.Context != "waiting for interrupt" {
		t.Errorf("expected context 'waiting for interrupt', got '%s'", err.Context)
	}
	if err.Cause != unix.ETIMEDOUT {
		t.Errorf("expected cause ETIMEDOUT, got %v", err.Cause)
	}
}

func TestStatusSuccessIsZero(t *testing.T) {
	if StatusSuccess != 0 {
		t.Errorf("StatusSuccess should be 0, got %d", StatusSuccess)
	}
}
