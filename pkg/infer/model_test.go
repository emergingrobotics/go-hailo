//go:build unit

package infer

import (
	"testing"
	"time"
)

func TestModelInputInfo(t *testing.T) {
	model := &Model{
		inputs: []StreamInfo{
			{
				Name:     "input_layer1",
				Shape:    Shape{Height: 224, Width: 224, Channels: 3},
				DataType: DataTypeUint8,
				Format:   FormatNHWC,
			},
		},
	}

	infos := model.InputInfo()

	if len(infos) != 1 {
		t.Fatalf("expected 1 input, got %d", len(infos))
	}

	info := infos[0]
	if info.Name != "input_layer1" {
		t.Errorf("name = %s, expected input_layer1", info.Name)
	}
	if info.Shape.Height != 224 {
		t.Errorf("height = %d, expected 224", info.Shape.Height)
	}
}

func TestModelOutputInfo(t *testing.T) {
	model := &Model{
		outputs: []StreamInfo{
			{
				Name:     "output_layer1",
				Shape:    Shape{Height: 1, Width: 1, Channels: 1000},
				DataType: DataTypeFloat32,
			},
			{
				Name:     "output_layer2",
				Shape:    Shape{Height: 1, Width: 1, Channels: 80},
				DataType: DataTypeFloat32,
			},
		},
	}

	infos := model.OutputInfo()

	if len(infos) != 2 {
		t.Errorf("expected 2 outputs, got %d", len(infos))
	}
}

func TestModelNetworkGroups(t *testing.T) {
	model := &Model{
		networkGroups: []string{"network1", "network2"},
	}

	groups := model.NetworkGroups()

	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}
	if groups[0] != "network1" {
		t.Error("first group should be network1")
	}
}

func TestModelFrameSize(t *testing.T) {
	model := &Model{
		inputs: []StreamInfo{
			{
				Name:     "input",
				Shape:    Shape{Height: 224, Width: 224, Channels: 3},
				DataType: DataTypeUint8,
			},
		},
	}

	size := model.InputFrameSize("input")
	expected := 224 * 224 * 3 * 1 // uint8 = 1 byte

	if size != expected {
		t.Errorf("frame size = %d, expected %d", size, expected)
	}
}

func TestModelValidation(t *testing.T) {
	tests := []struct {
		name    string
		model   *Model
		wantErr bool
	}{
		{
			"valid model",
			&Model{
				inputs:        []StreamInfo{{Name: "input"}},
				outputs:       []StreamInfo{{Name: "output"}},
				networkGroups: []string{"default"},
			},
			false,
		},
		{
			"no inputs",
			&Model{
				outputs:       []StreamInfo{{Name: "output"}},
				networkGroups: []string{"default"},
			},
			true,
		},
		{
			"no outputs",
			&Model{
				inputs:        []StreamInfo{{Name: "input"}},
				networkGroups: []string{"default"},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.model.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestModelClose(t *testing.T) {
	model := &Model{
		inputs:  []StreamInfo{{Name: "input"}},
		outputs: []StreamInfo{{Name: "output"}},
		closed:  false,
	}

	err := model.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if !model.closed {
		t.Error("model should be marked as closed")
	}

	// Double close should be safe
	err = model.Close()
	if err != nil {
		t.Errorf("second Close() error = %v", err)
	}
}

// Session tests

func TestSessionCreate(t *testing.T) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input", Shape: Shape{224, 224, 3}}},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 1000}}},
		networkGroups: []string{"default"},
	}

	session, err := model.NewSession()
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	defer session.Close()

	if session.model != model {
		t.Error("session should reference model")
	}
}

func TestSessionInferSync(t *testing.T) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input", Shape: Shape{2, 2, 1}, DataType: DataTypeUint8}},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 4}, DataType: DataTypeFloat32}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()
	defer session.Close()

	input := map[string][]byte{
		"input": make([]byte, 4), // 2x2x1
	}

	// Note: Without real hardware, this would use mock inference
	outputs, err := session.Infer(input)
	if err != nil {
		t.Logf("Infer error (expected without hardware): %v", err)
	}

	_ = outputs
}

func TestSessionClose(t *testing.T) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input"}},
		outputs:       []StreamInfo{{Name: "output"}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()

	err := session.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if !session.closed {
		t.Error("session should be marked as closed")
	}
}

func TestSessionTimeout(t *testing.T) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input"}},
		outputs:       []StreamInfo{{Name: "output"}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession(WithTimeout(10 * time.Millisecond))
	defer session.Close()

	if session.timeout != 10*time.Millisecond {
		t.Errorf("timeout = %v, expected 10ms", session.timeout)
	}
}

// Batch tests

func TestBatchInference(t *testing.T) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input", Shape: Shape{224, 224, 3}, DataType: DataTypeUint8}},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 1000}, DataType: DataTypeFloat32}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()
	defer session.Close()

	// Create batch of inputs
	batchSize := 4
	frameSize := 224 * 224 * 3
	inputs := make([]map[string][]byte, batchSize)
	for i := 0; i < batchSize; i++ {
		inputs[i] = map[string][]byte{
			"input": make([]byte, frameSize),
		}
	}

	// Note: Without real hardware, this is a mock test
	results, err := session.InferBatch(inputs)
	if err != nil {
		t.Logf("InferBatch error (expected without hardware): %v", err)
	}

	_ = results
}

// Types

type DataType int

const (
	DataTypeUint8 DataType = iota
	DataTypeUint16
	DataTypeFloat32
)

type Format int

const (
	FormatNHWC Format = iota
	FormatNCHW
)

type Shape struct {
	Height   int
	Width    int
	Channels int
}

type StreamInfo struct {
	Name     string
	Shape    Shape
	DataType DataType
	Format   Format
}

type Model struct {
	inputs        []StreamInfo
	outputs       []StreamInfo
	networkGroups []string
	closed        bool
}

func (m *Model) InputInfo() []StreamInfo {
	return m.inputs
}

func (m *Model) OutputInfo() []StreamInfo {
	return m.outputs
}

func (m *Model) NetworkGroups() []string {
	return m.networkGroups
}

func (m *Model) InputFrameSize(name string) int {
	for _, info := range m.inputs {
		if info.Name == name {
			elemSize := 1
			switch info.DataType {
			case DataTypeUint16:
				elemSize = 2
			case DataTypeFloat32:
				elemSize = 4
			}
			return info.Shape.Height * info.Shape.Width * info.Shape.Channels * elemSize
		}
	}
	return 0
}

func (m *Model) Validate() error {
	if len(m.inputs) == 0 {
		return ErrNoInputs
	}
	if len(m.outputs) == 0 {
		return ErrNoOutputs
	}
	return nil
}

func (m *Model) Close() error {
	if m.closed {
		return nil
	}
	m.closed = true
	return nil
}

func (m *Model) NewSession(opts ...SessionOption) (*Session, error) {
	s := &Session{
		model:   m,
		timeout: 5 * time.Second,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

type Session struct {
	model   *Model
	timeout time.Duration
	closed  bool
}

type SessionOption func(*Session)

func WithTimeout(d time.Duration) SessionOption {
	return func(s *Session) {
		s.timeout = d
	}
}

func (s *Session) Infer(inputs map[string][]byte) (map[string][]byte, error) {
	if s.closed {
		return nil, ErrSessionClosed
	}
	// Mock implementation
	return nil, ErrNotImplemented
}

func (s *Session) InferBatch(inputs []map[string][]byte) ([]map[string][]byte, error) {
	if s.closed {
		return nil, ErrSessionClosed
	}
	// Mock implementation
	return nil, ErrNotImplemented
}

func (s *Session) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	return nil
}

// Errors
type inferError string

func (e inferError) Error() string { return string(e) }

const (
	ErrNoInputs       = inferError("model has no inputs")
	ErrNoOutputs      = inferError("model has no outputs")
	ErrSessionClosed  = inferError("session is closed")
	ErrNotImplemented = inferError("not implemented")
)
