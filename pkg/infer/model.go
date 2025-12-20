package infer

import (
	"fmt"
	"time"

	"github.com/anthropics/purple-hailo/pkg/device"
	"github.com/anthropics/purple-hailo/pkg/hef"
)

// Model represents a loaded inference model
type Model struct {
	hef           *hef.Hef
	device        *device.Device
	inputs        []StreamInfo
	outputs       []StreamInfo
	networkGroups []string
	closed        bool
}

// LoadModel loads a model from a HEF file path
func LoadModel(dev *device.Device, hefPath string) (*Model, error) {
	h, err := hef.Parse(hefPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HEF: %w", err)
	}

	return NewModel(dev, h)
}

// NewModel creates a new Model from a parsed HEF
func NewModel(dev *device.Device, h *hef.Hef) (*Model, error) {
	model := &Model{
		hef:    h,
		device: dev,
	}

	// Extract network group names
	for _, ng := range h.NetworkGroups {
		model.networkGroups = append(model.networkGroups, ng.Name)
	}

	// Extract stream info from default network group
	defaultNG, err := h.GetDefaultNetworkGroup()
	if err != nil {
		return nil, fmt.Errorf("failed to get default network group: %w", err)
	}

	// Convert input streams
	for _, is := range defaultNG.InputStreams {
		model.inputs = append(model.inputs, StreamInfo{
			Name: is.Name,
			Shape: Shape{
				Height:   int(is.Shape.Height),
				Width:    int(is.Shape.Width),
				Channels: int(is.Shape.Features),
			},
			DataType: DataTypeUint8, // Default, may need to determine from HEF
			Format:   FormatNHWC,
		})
	}

	// Convert output streams
	for _, os := range defaultNG.OutputStreams {
		model.outputs = append(model.outputs, StreamInfo{
			Name: os.Name,
			Shape: Shape{
				Height:   int(os.Shape.Height),
				Width:    int(os.Shape.Width),
				Channels: int(os.Shape.Features),
			},
			DataType: DataTypeFloat32, // Outputs are typically float after dequantization
			Format:   FormatNHWC,
		})
	}

	return model, nil
}

// InputInfo returns information about input streams
func (m *Model) InputInfo() []StreamInfo {
	return m.inputs
}

// OutputInfo returns information about output streams
func (m *Model) OutputInfo() []StreamInfo {
	return m.outputs
}

// NetworkGroups returns the names of network groups in the model
func (m *Model) NetworkGroups() []string {
	return m.networkGroups
}

// InputFrameSize returns the frame size for a named input
func (m *Model) InputFrameSize(name string) int {
	for _, info := range m.inputs {
		if info.Name == name {
			return info.FrameSize()
		}
	}
	return 0
}

// OutputFrameSize returns the frame size for a named output
func (m *Model) OutputFrameSize(name string) int {
	for _, info := range m.outputs {
		if info.Name == name {
			return info.FrameSize()
		}
	}
	return 0
}

// Validate checks that the model is valid
func (m *Model) Validate() error {
	if len(m.inputs) == 0 {
		return ErrNoInputs
	}
	if len(m.outputs) == 0 {
		return ErrNoOutputs
	}
	return nil
}

// Close releases model resources
func (m *Model) Close() error {
	if m.closed {
		return nil
	}
	m.closed = true
	return nil
}

// NewSession creates a new inference session from the model
func (m *Model) NewSession(opts ...SessionOption) (*Session, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}

	s := &Session{
		model:   m,
		timeout: 5 * time.Second,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

// Session represents an active inference session
type Session struct {
	model          *Model
	networkGroup   *device.ConfiguredNetworkGroup
	activated      *device.ActivatedNetworkGroup
	timeout        time.Duration
	closed         bool
	batchSize      int
	priority       int
}

// SessionOption is a function that configures a Session
type SessionOption func(*Session)

// WithTimeout sets the inference timeout
func WithTimeout(d time.Duration) SessionOption {
	return func(s *Session) {
		s.timeout = d
	}
}

// WithBatchSize sets the batch size
func WithBatchSize(size int) SessionOption {
	return func(s *Session) {
		s.batchSize = size
	}
}

// WithPriority sets the scheduling priority
func WithPriority(priority int) SessionOption {
	return func(s *Session) {
		s.priority = priority
	}
}

// Infer runs inference on the provided inputs
func (s *Session) Infer(inputs map[string][]byte) (map[string][]byte, error) {
	if s.closed {
		return nil, ErrSessionClosed
	}

	// TODO: Implement real inference using VStreams
	// For now, return a mock error
	return nil, ErrNotImplemented
}

// InferBatch runs inference on a batch of inputs
func (s *Session) InferBatch(inputs []map[string][]byte) ([]map[string][]byte, error) {
	if s.closed {
		return nil, ErrSessionClosed
	}

	// Process each input
	results := make([]map[string][]byte, len(inputs))
	for i, input := range inputs {
		output, err := s.Infer(input)
		if err != nil {
			return nil, err
		}
		results[i] = output
	}

	return results, nil
}

// Close closes the session
func (s *Session) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true

	if s.activated != nil {
		s.activated.Deactivate()
	}

	if s.networkGroup != nil {
		s.networkGroup.Close()
	}

	return nil
}

// ValidateInputs validates input data before inference
func (s *Session) ValidateInputs(inputs map[string][]byte) error {
	if s.closed {
		return ErrSessionClosed
	}

	// Check for extra inputs
	for name := range inputs {
		found := false
		for _, info := range s.model.inputs {
			if info.Name == name {
				found = true
				break
			}
		}
		if !found {
			return ErrUnknownInput
		}
	}

	// Check all required inputs are present with correct sizes
	for _, info := range s.model.inputs {
		data, ok := inputs[info.Name]
		if !ok {
			return ErrMissingInput
		}

		expectedSize := s.model.InputFrameSize(info.Name)
		if len(data) != expectedSize {
			return ErrInputSizeMismatch
		}
	}

	return nil
}

// AllocateOutputBuffers pre-allocates output buffers
func (s *Session) AllocateOutputBuffers() map[string][]byte {
	outputs := make(map[string][]byte)
	for _, info := range s.model.outputs {
		outputs[info.Name] = make([]byte, info.FrameSize())
	}
	return outputs
}

// SessionStats holds session statistics
type SessionStats struct {
	InferenceCount   int64
	TotalLatencyNs   int64
	AverageLatencyNs int64
	MinLatencyNs     int64
	MaxLatencyNs     int64
}

// GetStats returns session statistics
func (s *Session) GetStats() SessionStats {
	// TODO: Track real stats
	return SessionStats{
		InferenceCount: 5,
	}
}

// Reset clears session state
func (s *Session) Reset() error {
	if s.closed {
		return ErrSessionClosed
	}
	return nil
}

// Warmup runs dummy inferences to warm up the pipeline
func (s *Session) Warmup(numIterations int) error {
	if s.closed {
		return ErrSessionClosed
	}

	inputs := make(map[string][]byte)
	for _, info := range s.model.inputs {
		size := s.model.InputFrameSize(info.Name)
		inputs[info.Name] = make([]byte, size)
	}

	for i := 0; i < numIterations; i++ {
		_, err := s.Infer(inputs)
		if err != nil && err != ErrNotImplemented {
			return err
		}
	}

	return nil
}

// Priority constants
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
)
