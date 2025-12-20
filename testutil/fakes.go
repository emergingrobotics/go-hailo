package testutil

import (
	"errors"
	"sync"
)

// FakeDevice implements a mock Hailo device for testing
type FakeDevice struct {
	mu           sync.Mutex
	open         bool
	configured   bool
	properties   DeviceProperties
	inferences   int
	failOnOpen   bool
	failOnConfig bool
	failOnInfer  bool
}

// DeviceProperties contains mock device properties
type DeviceProperties struct {
	BoardType        int
	DescMaxPageSize  uint16
	DmaEnginesCount  int
	IsFirmwareLoaded bool
}

// NewFakeDevice creates a new fake device
func NewFakeDevice() *FakeDevice {
	return &FakeDevice{
		properties: DeviceProperties{
			BoardType:        0, // Hailo-8
			DescMaxPageSize:  4096,
			DmaEnginesCount:  1,
			IsFirmwareLoaded: true,
		},
	}
}

// Open simulates opening the device
func (d *FakeDevice) Open() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.failOnOpen {
		return errors.New("fake open error")
	}
	d.open = true
	return nil
}

// Close simulates closing the device
func (d *FakeDevice) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.open = false
	d.configured = false
	return nil
}

// Configure simulates configuring with a HEF
func (d *FakeDevice) Configure(hef []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.open {
		return errors.New("device not open")
	}
	if d.failOnConfig {
		return errors.New("fake config error")
	}
	d.configured = true
	return nil
}

// Infer simulates running inference
func (d *FakeDevice) Infer(input []byte) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.open {
		return nil, errors.New("device not open")
	}
	if !d.configured {
		return nil, errors.New("device not configured")
	}
	if d.failOnInfer {
		return nil, errors.New("fake infer error")
	}

	d.inferences++

	// Return mock output (1000 float32 values)
	output := make([]byte, 4000)
	return output, nil
}

// Properties returns device properties
func (d *FakeDevice) Properties() DeviceProperties {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.properties
}

// InferenceCount returns number of inferences run
func (d *FakeDevice) InferenceCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.inferences
}

// SetFailOnOpen makes Open() fail
func (d *FakeDevice) SetFailOnOpen(fail bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.failOnOpen = fail
}

// SetFailOnConfig makes Configure() fail
func (d *FakeDevice) SetFailOnConfig(fail bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.failOnConfig = fail
}

// SetFailOnInfer makes Infer() fail
func (d *FakeDevice) SetFailOnInfer(fail bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.failOnInfer = fail
}

// FakeHef creates a minimal valid HEF buffer for testing
func FakeHef() []byte {
	// Create minimal V2 HEF header
	data := make([]byte, 100)

	// Magic: 0x01484546 (FEH\x01)
	data[0] = 0x46
	data[1] = 0x45
	data[2] = 0x48
	data[3] = 0x01

	// Version: 2
	data[4] = 0x02
	data[5] = 0x00
	data[6] = 0x00
	data[7] = 0x00

	// Proto size: 60
	data[8] = 0x3C
	data[9] = 0x00
	data[10] = 0x00
	data[11] = 0x00

	return data
}

// FakeInput creates a fake input buffer
func FakeInput(height, width, channels int) []byte {
	return make([]byte, height*width*channels)
}

// FakeBufferPool implements a mock buffer pool
type FakeBufferPool struct {
	mu        sync.Mutex
	buffers   [][]byte
	available []int
	size      int
}

// NewFakeBufferPool creates a fake buffer pool
func NewFakeBufferPool(bufferSize, count int) *FakeBufferPool {
	pool := &FakeBufferPool{
		buffers:   make([][]byte, count),
		available: make([]int, 0, count),
		size:      bufferSize,
	}
	for i := 0; i < count; i++ {
		pool.buffers[i] = make([]byte, bufferSize)
		pool.available = append(pool.available, i)
	}
	return pool
}

// Acquire gets a buffer from the pool
func (p *FakeBufferPool) Acquire() ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.available) == 0 {
		return nil, errors.New("pool exhausted")
	}

	idx := p.available[len(p.available)-1]
	p.available = p.available[:len(p.available)-1]
	return p.buffers[idx], nil
}

// Release returns a buffer to the pool
func (p *FakeBufferPool) Release(buf []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, b := range p.buffers {
		if &b[0] == &buf[0] {
			p.available = append(p.available, i)
			return
		}
	}
}

// Available returns number of available buffers
func (p *FakeBufferPool) Available() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.available)
}
