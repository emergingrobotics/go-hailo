//go:build unit

package infer

import (
	"sync"
	"testing"
	"time"
)

// Extended Session tests

func TestSessionInputValidation(t *testing.T) {
	model := &Model{
		inputs: []StreamInfo{
			{Name: "input1", Shape: Shape{224, 224, 3}, DataType: DataTypeUint8},
			{Name: "input2", Shape: Shape{100, 100, 1}, DataType: DataTypeUint8},
		},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 1000}}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()
	defer session.Close()

	tests := []struct {
		name    string
		inputs  map[string][]byte
		wantErr bool
	}{
		{
			name: "valid inputs",
			inputs: map[string][]byte{
				"input1": make([]byte, 224*224*3),
				"input2": make([]byte, 100*100*1),
			},
			wantErr: false,
		},
		{
			name: "missing input",
			inputs: map[string][]byte{
				"input1": make([]byte, 224*224*3),
				// input2 missing
			},
			wantErr: true,
		},
		{
			name: "wrong size",
			inputs: map[string][]byte{
				"input1": make([]byte, 100), // Wrong size
				"input2": make([]byte, 100*100*1),
			},
			wantErr: true,
		},
		{
			name: "unknown input",
			inputs: map[string][]byte{
				"input1":  make([]byte, 224*224*3),
				"input2":  make([]byte, 100*100*1),
				"unknown": make([]byte, 100), // Extra unknown input
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := session.ValidateInputs(tt.inputs)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateInputs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSessionOutputAllocation(t *testing.T) {
	model := &Model{
		inputs: []StreamInfo{{Name: "input", Shape: Shape{224, 224, 3}}},
		outputs: []StreamInfo{
			{Name: "output1", Shape: Shape{1, 1, 1000}, DataType: DataTypeFloat32},
			{Name: "output2", Shape: Shape{1, 1, 80}, DataType: DataTypeFloat32},
		},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()
	defer session.Close()

	outputs := session.AllocateOutputBuffers()

	if len(outputs) != 2 {
		t.Fatalf("expected 2 output buffers, got %d", len(outputs))
	}

	// Check output1 size (1000 * 4 bytes for float32)
	if len(outputs["output1"]) != 1000*4 {
		t.Errorf("output1 size = %d, expected %d", len(outputs["output1"]), 1000*4)
	}

	// Check output2 size (80 * 4 bytes for float32)
	if len(outputs["output2"]) != 80*4 {
		t.Errorf("output2 size = %d, expected %d", len(outputs["output2"]), 80*4)
	}
}

func TestSessionConcurrentInference(t *testing.T) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input", Shape: Shape{2, 2, 1}}},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 4}}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()
	defer session.Close()

	numGoroutines := 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			inputs := map[string][]byte{
				"input": make([]byte, 4),
			}
			_, err := session.Infer(inputs)
			if err != nil && err != ErrNotImplemented {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent inference error: %v", err)
	}
}

func TestSessionStatistics(t *testing.T) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input", Shape: Shape{2, 2, 1}}},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 4}}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()
	defer session.Close()

	// Run some inferences
	for i := 0; i < 5; i++ {
		inputs := map[string][]byte{
			"input": make([]byte, 4),
		}
		session.Infer(inputs)
	}

	stats := session.GetStats()

	if stats.InferenceCount != 5 {
		t.Errorf("inference count = %d, expected 5", stats.InferenceCount)
	}
}

func TestSessionReset(t *testing.T) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input", Shape: Shape{2, 2, 1}}},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 4}}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()
	defer session.Close()

	// Run some inferences
	inputs := map[string][]byte{
		"input": make([]byte, 4),
	}
	session.Infer(inputs)

	// Reset session
	err := session.Reset()
	if err != nil {
		t.Errorf("Reset() error: %v", err)
	}
}

func TestSessionWarmup(t *testing.T) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input", Shape: Shape{224, 224, 3}}},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 1000}}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()
	defer session.Close()

	// Warmup with dummy data
	err := session.Warmup(3)
	if err != nil && err != ErrNotImplemented {
		t.Errorf("Warmup() error: %v", err)
	}
}

func TestSessionTimeoutEnforcement(t *testing.T) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input", Shape: Shape{2, 2, 1}}},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 4}}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession(WithTimeout(1 * time.Millisecond))
	defer session.Close()

	if session.timeout != 1*time.Millisecond {
		t.Errorf("timeout = %v, expected 1ms", session.timeout)
	}
}

func TestSessionOptions(t *testing.T) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input", Shape: Shape{224, 224, 3}}},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 1000}}},
		networkGroups: []string{"default"},
	}

	// Test multiple options
	session, _ := model.NewSession(
		WithTimeout(100*time.Millisecond),
		WithBatchSize(4),
		WithPriority(int(PriorityHigh)),
	)
	defer session.Close()

	if session.timeout != 100*time.Millisecond {
		t.Errorf("timeout = %v, expected 100ms", session.timeout)
	}
	if session.batchSize != 4 {
		t.Errorf("batch size = %d, expected 4", session.batchSize)
	}
	if session.priority != int(PriorityHigh) {
		t.Errorf("priority = %d, expected High", session.priority)
	}
}

// Benchmarks

func BenchmarkSessionInfer(b *testing.B) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input", Shape: Shape{224, 224, 3}}},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 1000}}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()
	defer session.Close()

	inputs := map[string][]byte{
		"input": make([]byte, 224*224*3),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.Infer(inputs)
	}
}

func BenchmarkSessionValidation(b *testing.B) {
	model := &Model{
		inputs: []StreamInfo{
			{Name: "input1", Shape: Shape{224, 224, 3}, DataType: DataTypeUint8},
			{Name: "input2", Shape: Shape{100, 100, 1}, DataType: DataTypeUint8},
		},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 1000}}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()
	defer session.Close()

	inputs := map[string][]byte{
		"input1": make([]byte, 224*224*3),
		"input2": make([]byte, 100*100*1),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.ValidateInputs(inputs)
	}
}

func BenchmarkAllocateOutputBuffers(b *testing.B) {
	model := &Model{
		inputs: []StreamInfo{{Name: "input", Shape: Shape{224, 224, 3}}},
		outputs: []StreamInfo{
			{Name: "output1", Shape: Shape{1, 1, 1000}, DataType: DataTypeFloat32},
			{Name: "output2", Shape: Shape{80, 1, 85}, DataType: DataTypeFloat32},
		},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()
	defer session.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = session.AllocateOutputBuffers()
	}
}
