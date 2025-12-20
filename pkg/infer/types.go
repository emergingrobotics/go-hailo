package infer

// DataType represents the data type of a tensor
type DataType int

const (
	DataTypeUint8 DataType = iota
	DataTypeUint16
	DataTypeFloat32
)

// Format represents the tensor format
type Format int

const (
	FormatNHWC Format = iota
	FormatNCHW
)

// Shape represents tensor dimensions
type Shape struct {
	Height   int
	Width    int
	Channels int
}

// Size returns the total number of elements
func (s Shape) Size() int {
	return s.Height * s.Width * s.Channels
}

// StreamInfo describes an input or output stream
type StreamInfo struct {
	Name     string
	Shape    Shape
	DataType DataType
	Format   Format
}

// FrameSize returns the size in bytes for a single frame
func (s StreamInfo) FrameSize() int {
	elemSize := 1
	switch s.DataType {
	case DataTypeUint16:
		elemSize = 2
	case DataTypeFloat32:
		elemSize = 4
	}
	return s.Shape.Size() * elemSize
}
