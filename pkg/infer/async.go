package infer

import (
	"context"
	"sync"
)

// AsyncCallback is called when async inference completes
type AsyncCallback func(outputs map[string][]byte, err error)

// AsyncRequest represents an async inference request
type AsyncRequest struct {
	ID       uint64
	Inputs   map[string][]byte
	Callback AsyncCallback
	Done     chan struct{}
}

// AsyncSession wraps Session with async capabilities
type AsyncSession struct {
	session    *Session
	pending    map[uint64]*AsyncRequest
	nextID     uint64
	mu         sync.Mutex
	workerPool chan struct{}
	closed     bool
}

// NewAsyncSession creates a new async session
func NewAsyncSession(session *Session, numWorkers int) *AsyncSession {
	return &AsyncSession{
		session:    session,
		pending:    make(map[uint64]*AsyncRequest),
		workerPool: make(chan struct{}, numWorkers),
	}
}

// InferAsync submits an async inference request
func (as *AsyncSession) InferAsync(inputs map[string][]byte, callback AsyncCallback) uint64 {
	as.mu.Lock()
	defer as.mu.Unlock()

	if as.closed {
		callback(nil, ErrSessionClosed)
		return 0
	}

	as.nextID++
	id := as.nextID

	req := &AsyncRequest{
		ID:       id,
		Inputs:   inputs,
		Callback: callback,
		Done:     make(chan struct{}),
	}
	as.pending[id] = req

	go as.processRequest(req)

	return id
}

func (as *AsyncSession) processRequest(req *AsyncRequest) {
	// Acquire worker slot
	as.workerPool <- struct{}{}
	defer func() { <-as.workerPool }()

	outputs, err := as.session.Infer(req.Inputs)
	req.Callback(outputs, err)

	close(req.Done)

	as.mu.Lock()
	delete(as.pending, req.ID)
	as.mu.Unlock()
}

// Wait waits for a specific request to complete
func (as *AsyncSession) Wait(id uint64) error {
	as.mu.Lock()
	req, ok := as.pending[id]
	as.mu.Unlock()

	if !ok {
		return nil // Already completed
	}

	<-req.Done
	return nil
}

// WaitAll waits for all pending requests to complete
func (as *AsyncSession) WaitAll() {
	as.mu.Lock()
	pending := make([]*AsyncRequest, 0, len(as.pending))
	for _, req := range as.pending {
		pending = append(pending, req)
	}
	as.mu.Unlock()

	for _, req := range pending {
		<-req.Done
	}
}

// Close closes the async session
func (as *AsyncSession) Close() error {
	as.mu.Lock()
	as.closed = true
	as.mu.Unlock()

	as.WaitAll()
	return as.session.Close()
}

// PendingCount returns the number of pending requests
func (as *AsyncSession) PendingCount() int {
	as.mu.Lock()
	defer as.mu.Unlock()
	return len(as.pending)
}

// InferencePipeline chains preprocessing, inference, and postprocessing
type InferencePipeline struct {
	preprocess  func([]byte) []byte
	session     *Session
	postprocess func(map[string][]byte) interface{}
}

// NewPipeline creates a new inference pipeline
func NewPipeline(session *Session) *InferencePipeline {
	return &InferencePipeline{
		session:     session,
		preprocess:  func(data []byte) []byte { return data },
		postprocess: func(outputs map[string][]byte) interface{} { return outputs },
	}
}

// WithPreprocess sets the preprocessing function
func (p *InferencePipeline) WithPreprocess(fn func([]byte) []byte) *InferencePipeline {
	p.preprocess = fn
	return p
}

// WithPostprocess sets the postprocessing function
func (p *InferencePipeline) WithPostprocess(fn func(map[string][]byte) interface{}) *InferencePipeline {
	p.postprocess = fn
	return p
}

// Run executes the full pipeline
func (p *InferencePipeline) Run(input []byte) (interface{}, error) {
	processed := p.preprocess(input)
	outputs, err := p.session.Infer(map[string][]byte{"input": processed})
	if err != nil {
		return nil, err
	}
	return p.postprocess(outputs), nil
}

// InferWithContext runs inference with context for cancellation
func InferWithContext(ctx context.Context, session *Session, inputs map[string][]byte) (map[string][]byte, error) {
	done := make(chan struct{})
	var outputs map[string][]byte
	var err error

	go func() {
		outputs, err = session.Infer(inputs)
		close(done)
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-done:
		return outputs, err
	}
}
