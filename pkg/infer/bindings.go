package infer

// Bindings represents input/output buffer bindings for inference
type Bindings struct {
	inputs  map[string]*BoundBuffer
	outputs map[string]*BoundBuffer
}

// BoundBuffer represents a bound buffer
type BoundBuffer struct {
	data []byte
	size uint64
}

// NewBindings creates new empty bindings
func NewBindings() *Bindings {
	return &Bindings{
		inputs:  make(map[string]*BoundBuffer),
		outputs: make(map[string]*BoundBuffer),
	}
}

// SetInput binds input data to a named stream
func (b *Bindings) SetInput(name string, data []byte) error {
	if len(name) == 0 {
		return ErrInvalidStreamName
	}
	b.inputs[name] = &BoundBuffer{
		data: data,
		size: uint64(len(data)),
	}
	return nil
}

// SetOutput binds output buffer to a named stream
func (b *Bindings) SetOutput(name string, data []byte) error {
	if len(name) == 0 {
		return ErrInvalidStreamName
	}
	b.outputs[name] = &BoundBuffer{
		data: data,
		size: uint64(len(data)),
	}
	return nil
}

// Input returns the bound input buffer for a named stream
func (b *Bindings) Input(name string) *BoundBuffer {
	return b.inputs[name]
}

// Output returns the bound output buffer for a named stream
func (b *Bindings) Output(name string) *BoundBuffer {
	return b.outputs[name]
}

// InputNames returns all bound input stream names
func (b *Bindings) InputNames() []string {
	names := make([]string, 0, len(b.inputs))
	for name := range b.inputs {
		names = append(names, name)
	}
	return names
}

// OutputNames returns all bound output stream names
func (b *Bindings) OutputNames() []string {
	names := make([]string, 0, len(b.outputs))
	for name := range b.outputs {
		names = append(names, name)
	}
	return names
}

// Clear removes all bindings
func (b *Bindings) Clear() {
	b.inputs = make(map[string]*BoundBuffer)
	b.outputs = make(map[string]*BoundBuffer)
}

// Validate checks that all required bindings are present with correct sizes
func (b *Bindings) Validate(inputSizes, outputSizes map[string]uint64) error {
	// Check all inputs are bound with correct sizes
	for name, expectedSize := range inputSizes {
		buf := b.inputs[name]
		if buf == nil {
			return ErrMissingBinding
		}
		if buf.size != expectedSize {
			return ErrBufferSizeMismatch
		}
	}

	// Check all outputs are bound with correct sizes
	for name, expectedSize := range outputSizes {
		buf := b.outputs[name]
		if buf == nil {
			return ErrMissingBinding
		}
		if buf.size != expectedSize {
			return ErrBufferSizeMismatch
		}
	}

	return nil
}

// Data returns the underlying data slice
func (bb *BoundBuffer) Data() []byte {
	return bb.data
}

// Size returns the buffer size
func (bb *BoundBuffer) Size() uint64 {
	return bb.size
}
