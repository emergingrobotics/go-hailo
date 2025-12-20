//go:build benchmark

package integration

import (
	"testing"
	"time"
)

// BenchmarkInferenceLatency measures per-frame inference time
func BenchmarkInferenceLatency(b *testing.B) {
	// This would use real hardware in integration tests
	// For now, simulate the measurement pattern

	frameSize := 224 * 224 * 3
	input := make([]byte, frameSize)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate inference
		start := time.Now()
		_ = simulateInference(input)
		_ = time.Since(start)
	}
}

// BenchmarkThroughput measures frames per second
func BenchmarkThroughput(b *testing.B) {
	frameSize := 224 * 224 * 3
	input := make([]byte, frameSize)

	start := time.Now()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = simulateInference(input)
	}

	elapsed := time.Since(start)
	fps := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(fps, "fps")
}

// BenchmarkBufferPoolAcquire measures buffer acquisition time
func BenchmarkBufferPoolAcquire(b *testing.B) {
	pool := newMockBufferPool(100, 4096)
	defer pool.close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := pool.acquire()
		pool.release(buf)
	}
}

// BenchmarkQuantization measures quantization throughput
func BenchmarkQuantization(b *testing.B) {
	input := make([]float32, 224*224*3)
	output := make([]uint8, len(input))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		quantizeBatch(input, output)
	}

	bytesPerOp := int64(len(input) * 4) // float32 input
	b.SetBytes(bytesPerOp)
}

// BenchmarkDequantization measures dequantization throughput
func BenchmarkDequantization(b *testing.B) {
	input := make([]uint8, 1000)
	output := make([]float32, len(input))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dequantizeBatch(input, output)
	}
}

// BenchmarkFormatConversion measures NHWC to NCHW conversion
func BenchmarkFormatConversion(b *testing.B) {
	size := 224 * 224 * 3
	input := make([]byte, size)
	output := make([]byte, size)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		convertNHWCtoNCHW(input, output, 224, 224, 3)
	}

	b.SetBytes(int64(size))
}

// BenchmarkNmsParsing measures NMS output parsing
func BenchmarkNmsParsing(b *testing.B) {
	// Create realistic NMS output
	numClasses := 80
	detectionsPerClass := 100
	dataSize := numClasses * (1 + detectionsPerClass*5)
	data := make([]float32, dataSize)

	// Fill with test data
	idx := 0
	for c := 0; c < numClasses; c++ {
		data[idx] = float32(detectionsPerClass)
		idx++
		for d := 0; d < detectionsPerClass; d++ {
			data[idx] = 0.1   // y_min
			data[idx+1] = 0.1 // x_min
			data[idx+2] = 0.5 // y_max
			data[idx+3] = 0.5 // x_max
			data[idx+4] = 0.9 // score
			idx += 5
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parseNms(data, numClasses)
	}
}

// BenchmarkConcurrentInference measures parallel inference throughput
func BenchmarkConcurrentInference(b *testing.B) {
	numWorkers := 4
	frameSize := 224 * 224 * 3

	b.SetParallelism(numWorkers)
	b.RunParallel(func(pb *testing.PB) {
		input := make([]byte, frameSize)
		for pb.Next() {
			_ = simulateInference(input)
		}
	})
}

// Mock implementations for benchmarking

func simulateInference(input []byte) []float32 {
	// Simulate some work
	result := make([]float32, 1000)
	for i := range result {
		result[i] = float32(i) / 1000.0
	}
	return result
}

type mockBufferPool struct {
	buffers chan []byte
}

func newMockBufferPool(count, size int) *mockBufferPool {
	pool := &mockBufferPool{
		buffers: make(chan []byte, count),
	}
	for i := 0; i < count; i++ {
		pool.buffers <- make([]byte, size)
	}
	return pool
}

func (p *mockBufferPool) acquire() []byte {
	return <-p.buffers
}

func (p *mockBufferPool) release(buf []byte) {
	p.buffers <- buf
}

func (p *mockBufferPool) close() {
	close(p.buffers)
}

func quantizeBatch(input []float32, output []uint8) {
	scale := float32(0.00784314)
	zeroPoint := float32(128.0)
	for i, v := range input {
		q := v/scale + zeroPoint
		if q < 0 {
			output[i] = 0
		} else if q > 255 {
			output[i] = 255
		} else {
			output[i] = uint8(q)
		}
	}
}

func dequantizeBatch(input []uint8, output []float32) {
	scale := float32(0.00784314)
	zeroPoint := float32(128.0)
	for i, v := range input {
		output[i] = (float32(v) - zeroPoint) * scale
	}
}

func convertNHWCtoNCHW(src, dst []byte, h, w, c int) {
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			for ch := 0; ch < c; ch++ {
				srcIdx := (y*w+x)*c + ch
				dstIdx := ch*h*w + y*w + x
				dst[dstIdx] = src[srcIdx]
			}
		}
	}
}

type detection struct {
	yMin, xMin, yMax, xMax, score float32
	classId                       int
}

func parseNms(data []float32, numClasses int) []detection {
	var detections []detection
	idx := 0
	for classId := 0; classId < numClasses; classId++ {
		count := int(data[idx])
		idx++
		for d := 0; d < count && idx+5 <= len(data); d++ {
			detections = append(detections, detection{
				yMin:    data[idx],
				xMin:    data[idx+1],
				yMax:    data[idx+2],
				xMax:    data[idx+3],
				score:   data[idx+4],
				classId: classId,
			})
			idx += 5
		}
	}
	return detections
}
