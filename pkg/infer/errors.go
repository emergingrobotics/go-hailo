package infer

// inferError is a simple error type for the infer package
type inferError string

func (e inferError) Error() string { return string(e) }

// Errors for inference operations
const (
	ErrNoInputs           = inferError("model has no inputs")
	ErrNoOutputs          = inferError("model has no outputs")
	ErrSessionClosed      = inferError("session is closed")
	ErrNotImplemented     = inferError("not implemented")
	ErrInvalidStreamName  = inferError("invalid stream name")
	ErrMissingBinding     = inferError("missing binding")
	ErrBufferSizeMismatch = inferError("buffer size mismatch")
	ErrModelClosed        = inferError("model is closed")
	ErrInvalidInput       = inferError("invalid input data")
	ErrInferenceTimeout   = inferError("inference timed out")
	ErrMissingInput       = inferError("missing required input")
	ErrInputSizeMismatch  = inferError("input size mismatch")
	ErrUnknownInput       = inferError("unknown input name")
)
