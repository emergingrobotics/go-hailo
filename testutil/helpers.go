package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// SkipIfNoDevice skips test if no Hailo device is present
func SkipIfNoDevice(t *testing.T) string {
	t.Helper()

	devices := []string{"/dev/hailo0", "/dev/hailo1", "/dev/hailo2"}
	for _, path := range devices {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	t.Skip("No Hailo device available")
	return ""
}

// SkipIfNoHef skips test if no HEF file is available
func SkipIfNoHef(t *testing.T) string {
	t.Helper()

	hefPaths := []string{
		"testdata/resnet50.hef",
		"testdata/yolov5.hef",
		"../testdata/model.hef",
	}
	for _, path := range hefPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	t.Skip("No HEF file available")
	return ""
}

// TempDir creates a temporary directory for test artifacts
func TempDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// TempFile creates a temporary file with given content
func TempFile(t *testing.T, name string, content []byte) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, content, 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	return path
}

// MakeTestImage creates a test image buffer
func MakeTestImage(width, height, channels int) []byte {
	size := width * height * channels
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	return data
}

// MakeRandomBytes creates random test data
func MakeRandomBytes(size int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte((i * 17 + 11) % 256)
	}
	return data
}

// AssertEqual fails if values are not equal
func AssertEqual(t *testing.T, got, want interface{}, msg string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %v, want %v", msg, got, want)
	}
}

// AssertNoError fails if error is not nil
func AssertNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", msg, err)
	}
}

// AssertError fails if error is nil
func AssertError(t *testing.T, err error, msg string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: expected error, got nil", msg)
	}
}

// AssertBytesEqual compares byte slices
func AssertBytesEqual(t *testing.T, got, want []byte, msg string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: length mismatch: got %d, want %d", msg, len(got), len(want))
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%s: mismatch at index %d: got %d, want %d", msg, i, got[i], want[i])
			return
		}
	}
}

// AssertFloat32Near checks if floats are approximately equal
func AssertFloat32Near(t *testing.T, got, want, tolerance float32, msg string) {
	t.Helper()
	diff := got - want
	if diff < 0 {
		diff = -diff
	}
	if diff > tolerance {
		t.Errorf("%s: got %f, want %f (tolerance %f)", msg, got, want, tolerance)
	}
}

// MustParse is a helper for tests that panics on parse error
func MustParse(t *testing.T, parser func() (interface{}, error)) interface{} {
	t.Helper()
	result, err := parser()
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	return result
}

// Cleanup registers a cleanup function
func Cleanup(t *testing.T, fn func()) {
	t.Helper()
	t.Cleanup(fn)
}
