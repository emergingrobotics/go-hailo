//go:build unit

package transform

import (
	"testing"
)

func TestNHWCtoNCHW(t *testing.T) {
	// Test 2x2x3 image (H=2, W=2, C=3)
	// NHWC: [R00,G00,B00, R01,G01,B01, R10,G10,B10, R11,G11,B11]
	// NCHW: [R00,R01,R10,R11, G00,G01,G10,G11, B00,B01,B10,B11]

	nhwc := []uint8{
		1, 2, 3, // (0,0): R,G,B
		4, 5, 6, // (0,1): R,G,B
		7, 8, 9, // (1,0): R,G,B
		10, 11, 12, // (1,1): R,G,B
	}

	nchw := make([]uint8, len(nhwc))
	ConvertNHWCtoNCHW(nhwc, nchw, 2, 2, 3)

	expected := []uint8{
		1, 4, 7, 10, // R channel
		2, 5, 8, 11, // G channel
		3, 6, 9, 12, // B channel
	}

	for i, e := range expected {
		if nchw[i] != e {
			t.Errorf("nchw[%d] = %d, expected %d", i, nchw[i], e)
		}
	}
}

func TestNCHWtoNHWC(t *testing.T) {
	// Test 2x2x3 image
	nchw := []uint8{
		1, 4, 7, 10, // R channel
		2, 5, 8, 11, // G channel
		3, 6, 9, 12, // B channel
	}

	nhwc := make([]uint8, len(nchw))
	ConvertNCHWtoNHWC(nchw, nhwc, 2, 2, 3)

	expected := []uint8{
		1, 2, 3,
		4, 5, 6,
		7, 8, 9,
		10, 11, 12,
	}

	for i, e := range expected {
		if nhwc[i] != e {
			t.Errorf("nhwc[%d] = %d, expected %d", i, nhwc[i], e)
		}
	}
}

func TestFormatConversionRoundTrip(t *testing.T) {
	// Create random-ish test data
	height, width, channels := 4, 4, 3
	size := height * width * channels

	original := make([]uint8, size)
	for i := range original {
		original[i] = uint8(i)
	}

	// NHWC -> NCHW -> NHWC
	nchw := make([]uint8, size)
	roundTrip := make([]uint8, size)

	ConvertNHWCtoNCHW(original, nchw, height, width, channels)
	ConvertNCHWtoNHWC(nchw, roundTrip, height, width, channels)

	for i := range original {
		if roundTrip[i] != original[i] {
			t.Errorf("round trip mismatch at %d: got %d, expected %d",
				i, roundTrip[i], original[i])
		}
	}
}

func TestRGB888toRGBA(t *testing.T) {
	rgb := []uint8{
		255, 0, 0, // Red
		0, 255, 0, // Green
		0, 0, 255, // Blue
	}

	rgba := make([]uint8, 12) // 3 pixels * 4 channels
	ConvertRGB888toRGBA(rgb, rgba)

	expected := []uint8{
		255, 0, 0, 255, // Red + alpha
		0, 255, 0, 255, // Green + alpha
		0, 0, 255, 255, // Blue + alpha
	}

	for i, e := range expected {
		if rgba[i] != e {
			t.Errorf("rgba[%d] = %d, expected %d", i, rgba[i], e)
		}
	}
}

func TestRGBAtoRGB888(t *testing.T) {
	rgba := []uint8{
		255, 0, 0, 128, // Red with alpha
		0, 255, 0, 255, // Green with alpha
		0, 0, 255, 0, // Blue with alpha
	}

	rgb := make([]uint8, 9) // 3 pixels * 3 channels
	ConvertRGBAtoRGB888(rgba, rgb)

	expected := []uint8{
		255, 0, 0,
		0, 255, 0,
		0, 0, 255,
	}

	for i, e := range expected {
		if rgb[i] != e {
			t.Errorf("rgb[%d] = %d, expected %d", i, rgb[i], e)
		}
	}
}

func TestBGRtoRGB(t *testing.T) {
	bgr := []uint8{
		255, 128, 64, // B=255, G=128, R=64
		0, 128, 255, // B=0, G=128, R=255
	}

	rgb := make([]uint8, len(bgr))
	ConvertBGRtoRGB(bgr, rgb)

	expected := []uint8{
		64, 128, 255, // R=64, G=128, B=255
		255, 128, 0, // R=255, G=128, B=0
	}

	for i, e := range expected {
		if rgb[i] != e {
			t.Errorf("rgb[%d] = %d, expected %d", i, rgb[i], e)
		}
	}
}

func TestPaddingApplication(t *testing.T) {
	// Test adding padding to make dimensions power of 2
	input := make([]uint8, 224*224*3)
	for i := range input {
		input[i] = uint8(i % 256)
	}

	paddedWidth := 256
	paddedHeight := 256
	output := make([]uint8, paddedHeight*paddedWidth*3)

	ApplyPadding(input, output, 224, 224, paddedHeight, paddedWidth, 3)

	// Check that original data is preserved in top-left
	for y := 0; y < 224; y++ {
		for x := 0; x < 224; x++ {
			for c := 0; c < 3; c++ {
				srcIdx := (y*224+x)*3 + c
				dstIdx := (y*paddedWidth+x)*3 + c
				if output[dstIdx] != input[srcIdx] {
					t.Errorf("data mismatch at (%d,%d,%d)", y, x, c)
					return
				}
			}
		}
	}

	// Check that padding area is zeroed
	for y := 0; y < paddedHeight; y++ {
		for x := 0; x < paddedWidth; x++ {
			if x >= 224 || y >= 224 {
				for c := 0; c < 3; c++ {
					idx := (y*paddedWidth+x)*3 + c
					if output[idx] != 0 {
						t.Errorf("padding not zero at (%d,%d,%d): %d", y, x, c, output[idx])
						return
					}
				}
			}
		}
	}
}

func TestPaddingRemoval(t *testing.T) {
	paddedWidth := 256
	paddedHeight := 256
	padded := make([]uint8, paddedHeight*paddedWidth*3)

	// Fill with test pattern
	for y := 0; y < 224; y++ {
		for x := 0; x < 224; x++ {
			for c := 0; c < 3; c++ {
				idx := (y*paddedWidth+x)*3 + c
				padded[idx] = uint8((y + x + c) % 256)
			}
		}
	}

	output := make([]uint8, 224*224*3)
	RemovePadding(padded, output, paddedHeight, paddedWidth, 224, 224, 3)

	// Verify data is correct
	for y := 0; y < 224; y++ {
		for x := 0; x < 224; x++ {
			for c := 0; c < 3; c++ {
				idx := (y*224+x)*3 + c
				expected := uint8((y + x + c) % 256)
				if output[idx] != expected {
					t.Errorf("mismatch at (%d,%d,%d): got %d, expected %d",
						y, x, c, output[idx], expected)
					return
				}
			}
		}
	}
}

func BenchmarkNHWCtoNCHW(b *testing.B) {
	height, width, channels := 224, 224, 3
	size := height * width * channels
	nhwc := make([]uint8, size)
	nchw := make([]uint8, size)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConvertNHWCtoNCHW(nhwc, nchw, height, width, channels)
	}
}

// Format conversion functions

// ConvertNHWCtoNCHW converts from NHWC to NCHW format
func ConvertNHWCtoNCHW(src, dst []uint8, height, width, channels int) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			for c := 0; c < channels; c++ {
				srcIdx := (y*width+x)*channels + c
				dstIdx := c*height*width + y*width + x
				dst[dstIdx] = src[srcIdx]
			}
		}
	}
}

// ConvertNCHWtoNHWC converts from NCHW to NHWC format
func ConvertNCHWtoNHWC(src, dst []uint8, height, width, channels int) {
	for c := 0; c < channels; c++ {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				srcIdx := c*height*width + y*width + x
				dstIdx := (y*width+x)*channels + c
				dst[dstIdx] = src[srcIdx]
			}
		}
	}
}

// ConvertRGB888toRGBA adds alpha channel
func ConvertRGB888toRGBA(src, dst []uint8) {
	pixels := len(src) / 3
	for i := 0; i < pixels; i++ {
		dst[i*4] = src[i*3]     // R
		dst[i*4+1] = src[i*3+1] // G
		dst[i*4+2] = src[i*3+2] // B
		dst[i*4+3] = 255        // A
	}
}

// ConvertRGBAtoRGB888 removes alpha channel
func ConvertRGBAtoRGB888(src, dst []uint8) {
	pixels := len(src) / 4
	for i := 0; i < pixels; i++ {
		dst[i*3] = src[i*4]     // R
		dst[i*3+1] = src[i*4+1] // G
		dst[i*3+2] = src[i*4+2] // B
	}
}

// ConvertBGRtoRGB swaps blue and red channels
func ConvertBGRtoRGB(src, dst []uint8) {
	pixels := len(src) / 3
	for i := 0; i < pixels; i++ {
		dst[i*3] = src[i*3+2]   // R (was B)
		dst[i*3+1] = src[i*3+1] // G
		dst[i*3+2] = src[i*3]   // B (was R)
	}
}

// ApplyPadding adds zero padding to an image
func ApplyPadding(src, dst []uint8, srcH, srcW, dstH, dstW, channels int) {
	// Zero the destination
	for i := range dst {
		dst[i] = 0
	}

	// Copy source to top-left of destination
	for y := 0; y < srcH; y++ {
		for x := 0; x < srcW; x++ {
			for c := 0; c < channels; c++ {
				srcIdx := (y*srcW+x)*channels + c
				dstIdx := (y*dstW+x)*channels + c
				dst[dstIdx] = src[srcIdx]
			}
		}
	}
}

// RemovePadding extracts the original image from a padded image
func RemovePadding(src, dst []uint8, srcH, srcW, dstH, dstW, channels int) {
	for y := 0; y < dstH; y++ {
		for x := 0; x < dstW; x++ {
			for c := 0; c < channels; c++ {
				srcIdx := (y*srcW+x)*channels + c
				dstIdx := (y*dstW+x)*channels + c
				dst[dstIdx] = src[srcIdx]
			}
		}
	}
}
