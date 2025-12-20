//go:build unit

package transform

import (
	"math"
	"testing"
)

func TestQuantizeFloat32ToUint8(t *testing.T) {
	tests := []struct {
		name      string
		value     float32
		scale     float32
		zeroPoint float32
		expected  uint8
	}{
		{"zero", 0.0, 0.00784314, 128.0, 128},
		{"positive", 0.5, 0.00784314, 128.0, 192},
		{"negative", -0.5, 0.00784314, 128.0, 64},
		{"max", 1.0, 0.00784314, 128.0, 255},
		{"min", -1.0, 0.0078125, 128.0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qi := QuantInfo{
				Scale:     tt.scale,
				ZeroPoint: tt.zeroPoint,
			}
			result := Quantize(tt.value, qi)
			if result != tt.expected {
				t.Errorf("Quantize(%f) = %d, expected %d", tt.value, result, tt.expected)
			}
		})
	}
}

func TestDequantizeUint8ToFloat32(t *testing.T) {
	tests := []struct {
		name      string
		value     uint8
		scale     float32
		zeroPoint float32
		expected  float32
		tolerance float32
	}{
		{"middle", 128, 0.00784314, 128.0, 0.0, 0.001},
		{"high", 192, 0.00784314, 128.0, 0.502, 0.01},
		{"low", 64, 0.00784314, 128.0, -0.502, 0.01},
		{"max", 255, 0.00784314, 128.0, 0.996, 0.01},
		{"min", 0, 0.00784314, 128.0, -1.004, 0.01},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qi := QuantInfo{
				Scale:     tt.scale,
				ZeroPoint: tt.zeroPoint,
			}
			result := Dequantize(tt.value, qi)
			diff := float32(math.Abs(float64(result - tt.expected)))
			if diff > tt.tolerance {
				t.Errorf("Dequantize(%d) = %f, expected %f (tolerance %f)",
					tt.value, result, tt.expected, tt.tolerance)
			}
		})
	}
}

func TestQuantizeRoundTrip(t *testing.T) {
	qi := QuantInfo{
		Scale:     0.00784314,
		ZeroPoint: 128.0,
	}

	testValues := []float32{-0.8, -0.5, -0.2, 0.0, 0.2, 0.5, 0.8}

	for _, original := range testValues {
		quantized := Quantize(original, qi)
		dequantized := Dequantize(quantized, qi)

		diff := float32(math.Abs(float64(dequantized - original)))
		maxDiff := qi.Scale * 2

		if diff > maxDiff {
			t.Errorf("round trip error for %f: got %f (diff %f > %f)",
				original, dequantized, diff, maxDiff)
		}
	}
}

func TestQuantizeClipping(t *testing.T) {
	qi := QuantInfo{
		Scale:     0.00784314,
		ZeroPoint: 128.0,
	}

	highValues := []float32{1.5, 2.0, 10.0, 100.0}
	for _, v := range highValues {
		result := Quantize(v, qi)
		if result != 255 {
			t.Errorf("Quantize(%f) = %d, expected 255 (clipped)", v, result)
		}
	}

	lowValues := []float32{-1.5, -2.0, -10.0, -100.0}
	for _, v := range lowValues {
		result := Quantize(v, qi)
		if result != 0 {
			t.Errorf("Quantize(%f) = %d, expected 0 (clipped)", v, result)
		}
	}
}

func TestQuantizeZeroPoint(t *testing.T) {
	tests := []struct {
		zeroPoint float32
	}{
		{0.0},
		{64.0},
		{128.0},
		{200.0},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			qi := QuantInfo{
				Scale:     0.01,
				ZeroPoint: tt.zeroPoint,
			}

			result := Quantize(0.0, qi)
			expected := uint8(tt.zeroPoint)
			if result != expected {
				t.Errorf("Quantize(0.0) with zp=%f = %d, expected %d",
					tt.zeroPoint, result, expected)
			}
		})
	}
}

func TestQuantizeBatchData(t *testing.T) {
	qi := QuantInfo{
		Scale:     0.00784314,
		ZeroPoint: 128.0,
	}

	input := []float32{-1.0, -0.5, 0.0, 0.5, 1.0}
	output := make([]uint8, len(input))

	QuantizeBatch(input, output, qi)

	expected := []uint8{0, 64, 128, 192, 255}
	for i, e := range expected {
		diff := int(output[i]) - int(e)
		if diff < -1 || diff > 1 {
			t.Errorf("output[%d] = %d, expected ~%d", i, output[i], e)
		}
	}
}

func TestDequantizeBatchData(t *testing.T) {
	qi := QuantInfo{
		Scale:     0.00784314,
		ZeroPoint: 128.0,
	}

	input := []uint8{0, 64, 128, 192, 255}
	output := make([]float32, len(input))

	DequantizeBatch(input, output, qi)

	for i, v := range output {
		if v < -1.1 || v > 1.1 {
			t.Errorf("output[%d] = %f, expected in [-1.0, 1.0]", i, v)
		}
	}

	if math.Abs(float64(output[2])) > 0.01 {
		t.Errorf("output[2] (from 128) = %f, expected ~0", output[2])
	}
}

func TestQuantizePreservesOrder(t *testing.T) {
	qi := QuantInfo{
		Scale:     0.01,
		ZeroPoint: 128.0,
	}

	values := []float32{-0.8, -0.4, 0.0, 0.4, 0.8}
	var prevQuantized uint8 = 0

	for i, v := range values {
		quantized := Quantize(v, qi)
		if i > 0 && quantized < prevQuantized {
			t.Errorf("order not preserved: Q(%f)=%d < Q(%f)=%d",
				v, quantized, values[i-1], prevQuantized)
		}
		prevQuantized = quantized
	}
}

func BenchmarkQuantize(b *testing.B) {
	qi := QuantInfo{
		Scale:     0.00784314,
		ZeroPoint: 128.0,
	}
	value := float32(0.5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Quantize(value, qi)
	}
}

func BenchmarkDequantize(b *testing.B) {
	qi := QuantInfo{
		Scale:     0.00784314,
		ZeroPoint: 128.0,
	}
	value := uint8(192)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Dequantize(value, qi)
	}
}

func BenchmarkQuantizeBatch1000(b *testing.B) {
	qi := QuantInfo{
		Scale:     0.00784314,
		ZeroPoint: 128.0,
	}
	input := make([]float32, 1000)
	output := make([]uint8, 1000)

	for i := range input {
		input[i] = float32(i)/500.0 - 1.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		QuantizeBatch(input, output, qi)
	}
}
