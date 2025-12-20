//go:build unit

package stream

import (
	"testing"
	"time"

	"github.com/anthropics/purple-hailo/pkg/hef"
)

func TestVStreamInfoFields(t *testing.T) {
	info := VStreamInfo{
		Name:      "input0",
		FrameSize: 640 * 640 * 3,
		Height:    640,
		Width:     640,
		Channels:  3,
		Format:    hef.FormatTypeAuto,
		QuantInfo: hef.QuantInfo{
			Scale:     1.0,
			ZeroPoint: 0.0,
		},
	}

	if info.Name != "input0" {
		t.Errorf("Name = %s, expected input0", info.Name)
	}

	expectedFrameSize := uint64(640 * 640 * 3)
	if info.FrameSize != expectedFrameSize {
		t.Errorf("FrameSize = %d, expected %d", info.FrameSize, expectedFrameSize)
	}

	if info.Height != 640 {
		t.Errorf("Height = %d, expected 640", info.Height)
	}

	if info.Width != 640 {
		t.Errorf("Width = %d, expected 640", info.Width)
	}

	if info.Channels != 3 {
		t.Errorf("Channels = %d, expected 3", info.Channels)
	}
}

func TestDefaultVStreamParams(t *testing.T) {
	params := DefaultVStreamParams()

	if params.FormatType != hef.FormatTypeAuto {
		t.Errorf("FormatType = %v, expected FormatTypeAuto", params.FormatType)
	}

	if params.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, expected 5s", params.Timeout)
	}

	if params.QueueDepth != 2 {
		t.Errorf("QueueDepth = %d, expected 2", params.QueueDepth)
	}

	if params.BatchSize != 1 {
		t.Errorf("BatchSize = %d, expected 1", params.BatchSize)
	}
}

func TestInputVStreamInfoAccess(t *testing.T) {
	info := VStreamInfo{
		Name:      "test_input",
		FrameSize: 1024,
		Height:    32,
		Width:     32,
		Channels:  1,
	}

	// Create a mock input VStream
	vs := &InputVStream{
		info:      info,
		batchSize: 1,
	}

	if vs.Info().Name != info.Name {
		t.Errorf("Info().Name = %s, expected %s", vs.Info().Name, info.Name)
	}

	if vs.FrameSize() != info.FrameSize {
		t.Errorf("FrameSize() = %d, expected %d", vs.FrameSize(), info.FrameSize)
	}
}

func TestOutputVStreamInfoAccess(t *testing.T) {
	info := VStreamInfo{
		Name:      "test_output",
		FrameSize: 2048,
		Height:    1,
		Width:     1,
		Channels:  2048,
	}

	// Create a mock output VStream
	vs := &OutputVStream{
		info:      info,
		batchSize: 1,
	}

	if vs.Info().Name != info.Name {
		t.Errorf("Info().Name = %s, expected %s", vs.Info().Name, info.Name)
	}

	if vs.FrameSize() != info.FrameSize {
		t.Errorf("FrameSize() = %d, expected %d", vs.FrameSize(), info.FrameSize)
	}
}

func TestInputVStreamClosedError(t *testing.T) {
	vs := &InputVStream{
		info:      VStreamInfo{FrameSize: 100},
		closed:    true,
		batchSize: 1,
	}

	err := vs.Write(make([]byte, 100))
	if err != ErrStreamClosed {
		t.Errorf("Write() on closed stream should return ErrStreamClosed, got %v", err)
	}

	err = vs.Flush()
	if err != ErrStreamClosed {
		t.Errorf("Flush() on closed stream should return ErrStreamClosed, got %v", err)
	}
}

func TestOutputVStreamClosedError(t *testing.T) {
	vs := &OutputVStream{
		info:      VStreamInfo{FrameSize: 100},
		closed:    true,
		batchSize: 1,
	}

	_, err := vs.Read()
	if err != ErrStreamClosed {
		t.Errorf("Read() on closed stream should return ErrStreamClosed, got %v", err)
	}
}

func TestInputVStreamInvalidDataSize(t *testing.T) {
	info := VStreamInfo{
		FrameSize: 100,
	}

	vs := &InputVStream{
		info:      info,
		closed:    false,
		batchSize: 1,
	}

	// Wrong size data should fail
	err := vs.Write(make([]byte, 50))
	if err == nil {
		t.Error("Write() with wrong data size should return error")
	}
}

func TestVStreamSetByName(t *testing.T) {
	set := &VStreamSet{
		Inputs: []*InputVStream{
			{info: VStreamInfo{Name: "input0"}},
			{info: VStreamInfo{Name: "input1"}},
		},
		Outputs: []*OutputVStream{
			{info: VStreamInfo{Name: "output0"}},
			{info: VStreamInfo{Name: "output1"}},
		},
	}

	// Test InputByName
	input := set.InputByName("input1")
	if input == nil {
		t.Fatal("InputByName(input1) returned nil")
	}
	if input.info.Name != "input1" {
		t.Errorf("InputByName(input1) returned %s", input.info.Name)
	}

	// Test missing name
	missing := set.InputByName("nonexistent")
	if missing != nil {
		t.Error("InputByName(nonexistent) should return nil")
	}

	// Test OutputByName
	output := set.OutputByName("output0")
	if output == nil {
		t.Fatal("OutputByName(output0) returned nil")
	}
	if output.info.Name != "output0" {
		t.Errorf("OutputByName(output0) returned %s", output.info.Name)
	}
}
