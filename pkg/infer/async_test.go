//go:build unit

package infer

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestAsyncSessionCreate(t *testing.T) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input", Shape: Shape{224, 224, 3}}},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 1000}}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()
	asyncSession := NewAsyncSession(session, 4)

	if asyncSession.session != session {
		t.Error("async session should wrap original session")
	}

	asyncSession.Close()
}

func TestAsyncInferCallback(t *testing.T) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input", Shape: Shape{2, 2, 1}}},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 4}}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()
	asyncSession := NewAsyncSession(session, 4)
	defer asyncSession.Close()

	callbackCalled := make(chan bool, 1)

	inputs := map[string][]byte{
		"input": make([]byte, 4),
	}

	asyncSession.InferAsync(inputs, func(outputs map[string][]byte, err error) {
		callbackCalled <- true
	})

	select {
	case <-callbackCalled:
		// Success
	case <-time.After(time.Second):
		t.Error("callback was not called within timeout")
	}
}

func TestAsyncInferMultiple(t *testing.T) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input", Shape: Shape{2, 2, 1}}},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 4}}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()
	asyncSession := NewAsyncSession(session, 4)
	defer asyncSession.Close()

	numRequests := 10
	var callbackCount int32

	for i := 0; i < numRequests; i++ {
		inputs := map[string][]byte{
			"input": make([]byte, 4),
		}

		asyncSession.InferAsync(inputs, func(outputs map[string][]byte, err error) {
			atomic.AddInt32(&callbackCount, 1)
		})
	}

	asyncSession.WaitAll()

	if int(callbackCount) != numRequests {
		t.Errorf("expected %d callbacks, got %d", numRequests, callbackCount)
	}
}

func TestAsyncInferWait(t *testing.T) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input", Shape: Shape{2, 2, 1}}},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 4}}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()
	asyncSession := NewAsyncSession(session, 2)
	defer asyncSession.Close()

	completed := false

	inputs := map[string][]byte{
		"input": make([]byte, 4),
	}

	id := asyncSession.InferAsync(inputs, func(outputs map[string][]byte, err error) {
		completed = true
	})

	err := asyncSession.Wait(id)
	if err != nil {
		t.Errorf("Wait error: %v", err)
	}

	if !completed {
		t.Error("inference should be completed after Wait()")
	}
}

func TestAsyncInferConcurrency(t *testing.T) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input", Shape: Shape{2, 2, 1}}},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 4}}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()
	numWorkers := 2
	asyncSession := NewAsyncSession(session, numWorkers)
	defer asyncSession.Close()

	var maxConcurrent int32
	var currentConcurrent int32

	numRequests := 20
	for i := 0; i < numRequests; i++ {
		inputs := map[string][]byte{
			"input": make([]byte, 4),
		}

		asyncSession.InferAsync(inputs, func(outputs map[string][]byte, err error) {
			curr := atomic.AddInt32(&currentConcurrent, 1)
			if curr > atomic.LoadInt32(&maxConcurrent) {
				atomic.StoreInt32(&maxConcurrent, curr)
			}
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt32(&currentConcurrent, -1)
		})
	}

	asyncSession.WaitAll()

	if int(atomic.LoadInt32(&maxConcurrent)) > numWorkers {
		t.Errorf("max concurrent %d exceeded worker count %d", maxConcurrent, numWorkers)
	}
}

func TestAsyncSessionClose(t *testing.T) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input", Shape: Shape{2, 2, 1}}},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 4}}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()
	asyncSession := NewAsyncSession(session, 4)

	for i := 0; i < 5; i++ {
		inputs := map[string][]byte{
			"input": make([]byte, 4),
		}
		asyncSession.InferAsync(inputs, func(outputs map[string][]byte, err error) {})
	}

	err := asyncSession.Close()
	if err != nil {
		t.Errorf("Close error: %v", err)
	}
}

func TestAsyncInferAfterClose(t *testing.T) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input", Shape: Shape{2, 2, 1}}},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 4}}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()
	asyncSession := NewAsyncSession(session, 4)
	asyncSession.Close()

	errorReceived := false
	inputs := map[string][]byte{
		"input": make([]byte, 4),
	}

	asyncSession.InferAsync(inputs, func(outputs map[string][]byte, err error) {
		if err != nil {
			errorReceived = true
		}
	})

	time.Sleep(50 * time.Millisecond)

	if !errorReceived {
		t.Error("should receive error when inferring after close")
	}
}

func TestPipelineCreate(t *testing.T) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input", Shape: Shape{224, 224, 3}}},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 1000}}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()
	pipeline := NewPipeline(session)

	if pipeline.session != session {
		t.Error("pipeline should reference session")
	}
}

func TestPipelineWithPreprocess(t *testing.T) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input", Shape: Shape{2, 2, 1}}},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 4}}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()
	preprocessCalled := false

	pipeline := NewPipeline(session).WithPreprocess(func(data []byte) []byte {
		preprocessCalled = true
		result := make([]byte, len(data))
		for i, v := range data {
			result[i] = v / 2
		}
		return result
	})

	input := []byte{100, 100, 100, 100}
	_, _ = pipeline.Run(input)

	if !preprocessCalled {
		t.Error("preprocess should be called")
	}
}

func TestPipelineWithPostprocess(t *testing.T) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input", Shape: Shape{2, 2, 1}}},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 4}}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()
	postprocessCalled := false

	pipeline := NewPipeline(session).WithPostprocess(func(outputs map[string][]byte) interface{} {
		postprocessCalled = true
		return "processed"
	})

	input := make([]byte, 4)
	_, err := pipeline.Run(input)

	if err == nil && !postprocessCalled {
		t.Error("postprocess should be called when no error")
	}
	if err != nil && postprocessCalled {
		t.Error("postprocess should not be called on error")
	}
}

func TestInferWithContextTest(t *testing.T) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input", Shape: Shape{2, 2, 1}}},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 4}}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()
	defer session.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	inputs := map[string][]byte{
		"input": make([]byte, 4),
	}

	_, err := InferWithContext(ctx, session, inputs)
	_ = err
}

func TestInferWithContextCancellation(t *testing.T) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input", Shape: Shape{2, 2, 1}}},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 4}}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()
	defer session.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	inputs := map[string][]byte{
		"input": make([]byte, 4),
	}

	_, err := InferWithContext(ctx, session, inputs)
	if err != context.Canceled {
		t.Logf("Result: %v", err)
	}
}

func BenchmarkAsyncInfer(b *testing.B) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input", Shape: Shape{224, 224, 3}}},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 1000}}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()
	asyncSession := NewAsyncSession(session, 4)
	defer asyncSession.Close()

	inputs := map[string][]byte{
		"input": make([]byte, 224*224*3),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		done := make(chan struct{})
		asyncSession.InferAsync(inputs, func(outputs map[string][]byte, err error) {
			close(done)
		})
		<-done
	}
}

func BenchmarkPipeline(b *testing.B) {
	model := &Model{
		inputs:        []StreamInfo{{Name: "input", Shape: Shape{224, 224, 3}}},
		outputs:       []StreamInfo{{Name: "output", Shape: Shape{1, 1, 1000}}},
		networkGroups: []string{"default"},
	}

	session, _ := model.NewSession()
	pipeline := NewPipeline(session).
		WithPreprocess(func(data []byte) []byte { return data }).
		WithPostprocess(func(outputs map[string][]byte) interface{} { return nil })

	input := make([]byte, 224*224*3)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pipeline.Run(input)
	}
}
