package transform

// QuantInfo contains quantization parameters
type QuantInfo struct {
	ZeroPoint float32
	Scale     float32
	LimMin    float32
	LimMax    float32
}

// Quantize converts a float32 to uint8 using quantization parameters
func Quantize(value float32, qi QuantInfo) uint8 {
	// quantized = value / scale + zero_point
	quantized := value/qi.Scale + qi.ZeroPoint

	// Clip to uint8 range
	if quantized < 0 {
		return 0
	}
	if quantized > 255 {
		return 255
	}
	return uint8(quantized + 0.5) // Round
}

// Dequantize converts a uint8 to float32 using quantization parameters
func Dequantize(value uint8, qi QuantInfo) float32 {
	// float = (quantized - zero_point) * scale
	return (float32(value) - qi.ZeroPoint) * qi.Scale
}

// QuantizeBatch quantizes a batch of float32 values
func QuantizeBatch(input []float32, output []uint8, qi QuantInfo) {
	for i, v := range input {
		output[i] = Quantize(v, qi)
	}
}

// DequantizeBatch dequantizes a batch of uint8 values
func DequantizeBatch(input []uint8, output []float32, qi QuantInfo) {
	for i, v := range input {
		output[i] = Dequantize(v, qi)
	}
}

// QuantizeU16 converts a float32 to uint16 using quantization parameters
func QuantizeU16(value float32, qi QuantInfo) uint16 {
	quantized := value/qi.Scale + qi.ZeroPoint

	if quantized < 0 {
		return 0
	}
	if quantized > 65535 {
		return 65535
	}
	return uint16(quantized + 0.5)
}

// DequantizeU16 converts a uint16 to float32 using quantization parameters
func DequantizeU16(value uint16, qi QuantInfo) float32 {
	return (float32(value) - qi.ZeroPoint) * qi.Scale
}
