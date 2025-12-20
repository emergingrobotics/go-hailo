//go:build unit

package transform

import (
	"testing"
)

func TestNHWCtoNCHW(t *testing.T) {
	nhwc := []uint8{
		1, 2, 3,
		4, 5, 6,
		7, 8, 9,
		10, 11, 12,
	}

	nchw := make([]uint8, len(nhwc))
	ConvertNHWCtoNCHW(nhwc, nchw, 2, 2, 3)

	expected := []uint8{
		1, 4, 7, 10,
		2, 5, 8, 11,
		3, 6, 9, 12,
	}

	for i, e := range expected {
		if nchw[i] != e {
			t.Errorf("nchw[%d] = %d, expected %d", i, nchw[i], e)
		}
	}
}

func TestNCHWtoNHWC(t *testing.T) {
	nchw := []uint8{
		1, 4, 7, 10,
		2, 5, 8, 11,
		3, 6, 9, 12,
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
	height, width, channels := 4, 4, 3
	size := height * width * channels

	original := make([]uint8, size)
	for i := range original {
		original[i] = uint8(i)
	}

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
		255, 0, 0,
		0, 255, 0,
		0, 0, 255,
	}

	rgba := make([]uint8, 12)
	ConvertRGB888toRGBA(rgb, rgba)

	expected := []uint8{
		255, 0, 0, 255,
		0, 255, 0, 255,
		0, 0, 255, 255,
	}

	for i, e := range expected {
		if rgba[i] != e {
			t.Errorf("rgba[%d] = %d, expected %d", i, rgba[i], e)
		}
	}
}

func TestRGBAtoRGB888(t *testing.T) {
	rgba := []uint8{
		255, 0, 0, 128,
		0, 255, 0, 255,
		0, 0, 255, 0,
	}

	rgb := make([]uint8, 9)
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
		255, 128, 64,
		0, 128, 255,
	}

	rgb := make([]uint8, len(bgr))
	ConvertBGRtoRGB(bgr, rgb)

	expected := []uint8{
		64, 128, 255,
		255, 128, 0,
	}

	for i, e := range expected {
		if rgb[i] != e {
			t.Errorf("rgb[%d] = %d, expected %d", i, rgb[i], e)
		}
	}
}

func TestPaddingApplication(t *testing.T) {
	input := make([]uint8, 224*224*3)
	for i := range input {
		input[i] = uint8(i % 256)
	}

	paddedWidth := 256
	paddedHeight := 256
	output := make([]uint8, paddedHeight*paddedWidth*3)

	ApplyPadding(input, output, 224, 224, paddedHeight, paddedWidth, 3)

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
