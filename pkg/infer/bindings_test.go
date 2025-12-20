//go:build unit

package infer

import (
	"testing"
)

func TestBindingsCreate(t *testing.T) {
	bindings := NewBindings()

	if bindings == nil {
		t.Fatal("NewBindings returned nil")
	}

	if len(bindings.InputNames()) != 0 {
		t.Error("new bindings should have no inputs")
	}

	if len(bindings.OutputNames()) != 0 {
		t.Error("new bindings should have no outputs")
	}
}

func TestBindingsSetInput(t *testing.T) {
	bindings := NewBindings()

	data := make([]byte, 1000)
	err := bindings.SetInput("input_layer", data)
	if err != nil {
		t.Fatalf("SetInput() error = %v", err)
	}

	buf := bindings.Input("input_layer")
	if buf == nil {
		t.Fatal("Input() returned nil")
	}

	if buf.size != 1000 {
		t.Errorf("size = %d, expected 1000", buf.size)
	}
}

func TestBindingsSetOutput(t *testing.T) {
	bindings := NewBindings()

	data := make([]byte, 4000)
	err := bindings.SetOutput("output_layer", data)
	if err != nil {
		t.Fatalf("SetOutput() error = %v", err)
	}

	buf := bindings.Output("output_layer")
	if buf == nil {
		t.Fatal("Output() returned nil")
	}

	if buf.size != 4000 {
		t.Errorf("size = %d, expected 4000", buf.size)
	}
}

func TestBindingsEmptyName(t *testing.T) {
	bindings := NewBindings()

	err := bindings.SetInput("", make([]byte, 100))
	if err != ErrInvalidStreamName {
		t.Errorf("expected ErrInvalidStreamName, got %v", err)
	}

	err = bindings.SetOutput("", make([]byte, 100))
	if err != ErrInvalidStreamName {
		t.Errorf("expected ErrInvalidStreamName, got %v", err)
	}
}

func TestBindingsNonexistentStream(t *testing.T) {
	bindings := NewBindings()

	buf := bindings.Input("nonexistent")
	if buf != nil {
		t.Error("Input() for nonexistent stream should return nil")
	}

	buf = bindings.Output("nonexistent")
	if buf != nil {
		t.Error("Output() for nonexistent stream should return nil")
	}
}

func TestBindingsMultipleInputs(t *testing.T) {
	bindings := NewBindings()

	bindings.SetInput("input1", make([]byte, 100))
	bindings.SetInput("input2", make([]byte, 200))
	bindings.SetInput("input3", make([]byte, 300))

	names := bindings.InputNames()
	if len(names) != 3 {
		t.Errorf("expected 3 inputs, got %d", len(names))
	}
}

func TestBindingsMultipleOutputs(t *testing.T) {
	bindings := NewBindings()

	bindings.SetOutput("output1", make([]byte, 1000))
	bindings.SetOutput("output2", make([]byte, 2000))

	names := bindings.OutputNames()
	if len(names) != 2 {
		t.Errorf("expected 2 outputs, got %d", len(names))
	}
}

func TestBindingsOverwrite(t *testing.T) {
	bindings := NewBindings()

	bindings.SetInput("input", make([]byte, 100))
	bindings.SetInput("input", make([]byte, 200))

	buf := bindings.Input("input")
	if buf.size != 200 {
		t.Errorf("size = %d, expected 200 after overwrite", buf.size)
	}

	if len(bindings.InputNames()) != 1 {
		t.Error("overwrite should not create duplicate entry")
	}
}

func TestBindingsClear(t *testing.T) {
	bindings := NewBindings()

	bindings.SetInput("input", make([]byte, 100))
	bindings.SetOutput("output", make([]byte, 200))

	bindings.Clear()

	if len(bindings.InputNames()) != 0 {
		t.Error("Clear() should remove all inputs")
	}

	if len(bindings.OutputNames()) != 0 {
		t.Error("Clear() should remove all outputs")
	}
}

func TestBindingsValidate(t *testing.T) {
	bindings := NewBindings()

	bindings.SetInput("input1", make([]byte, 224*224*3))
	bindings.SetOutput("output1", make([]byte, 1000))

	inputSizes := map[string]uint64{
		"input1": 224 * 224 * 3,
	}
	outputSizes := map[string]uint64{
		"output1": 1000,
	}

	err := bindings.Validate(inputSizes, outputSizes)
	if err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestBindingsValidateMissing(t *testing.T) {
	bindings := NewBindings()

	inputSizes := map[string]uint64{
		"input1": 1000,
	}
	outputSizes := map[string]uint64{}

	err := bindings.Validate(inputSizes, outputSizes)
	if err != ErrMissingBinding {
		t.Errorf("expected ErrMissingBinding, got %v", err)
	}
}

func TestBindingsValidateSizeMismatch(t *testing.T) {
	bindings := NewBindings()

	bindings.SetInput("input1", make([]byte, 500))

	inputSizes := map[string]uint64{
		"input1": 1000,
	}
	outputSizes := map[string]uint64{}

	err := bindings.Validate(inputSizes, outputSizes)
	if err != ErrBufferSizeMismatch {
		t.Errorf("expected ErrBufferSizeMismatch, got %v", err)
	}
}

func TestBindingsDataAccess(t *testing.T) {
	bindings := NewBindings()

	data := make([]byte, 10)
	for i := range data {
		data[i] = byte(i)
	}

	bindings.SetInput("input", data)

	buf := bindings.Input("input")

	for i := range buf.data {
		if buf.data[i] != byte(i) {
			t.Errorf("data[%d] = %d, expected %d", i, buf.data[i], i)
		}
	}
}

func TestBindingsDataModification(t *testing.T) {
	bindings := NewBindings()

	data := make([]byte, 10)
	bindings.SetOutput("output", data)

	buf := bindings.Output("output")

	for i := range buf.data {
		buf.data[i] = byte(i * 2)
	}

	for i := range data {
		if data[i] != byte(i*2) {
			t.Errorf("original data[%d] = %d, expected %d", i, data[i], i*2)
		}
	}
}

func BenchmarkBindingsSetInput(b *testing.B) {
	bindings := NewBindings()
	data := make([]byte, 224*224*3)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bindings.SetInput("input", data)
	}
}

func BenchmarkBindingsAccess(b *testing.B) {
	bindings := NewBindings()
	bindings.SetInput("input", make([]byte, 224*224*3))
	bindings.SetOutput("output", make([]byte, 1000))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bindings.Input("input")
		_ = bindings.Output("output")
	}
}

func BenchmarkBindingsValidate(b *testing.B) {
	bindings := NewBindings()
	bindings.SetInput("input1", make([]byte, 224*224*3))
	bindings.SetInput("input2", make([]byte, 640*640*3))
	bindings.SetOutput("output1", make([]byte, 1000))
	bindings.SetOutput("output2", make([]byte, 8400*85))

	inputSizes := map[string]uint64{
		"input1": 224 * 224 * 3,
		"input2": 640 * 640 * 3,
	}
	outputSizes := map[string]uint64{
		"output1": 1000,
		"output2": 8400 * 85,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bindings.Validate(inputSizes, outputSizes)
	}
}
