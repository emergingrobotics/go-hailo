package transform

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

// ConvertNHWCtoNCHWF32 converts float32 data from NHWC to NCHW format
func ConvertNHWCtoNCHWF32(src, dst []float32, height, width, channels int) {
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

// ConvertNCHWtoNHWCF32 converts float32 data from NCHW to NHWC format
func ConvertNCHWtoNHWCF32(src, dst []float32, height, width, channels int) {
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

// ConvertRGBtoBGR swaps red and blue channels
func ConvertRGBtoBGR(src, dst []uint8) {
	ConvertBGRtoRGB(src, dst) // Same operation
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

// ResizeBilinear resizes an image using bilinear interpolation
func ResizeBilinear(src, dst []uint8, srcH, srcW, dstH, dstW, channels int) {
	xRatio := float32(srcW) / float32(dstW)
	yRatio := float32(srcH) / float32(dstH)

	for y := 0; y < dstH; y++ {
		for x := 0; x < dstW; x++ {
			srcX := float32(x) * xRatio
			srcY := float32(y) * yRatio

			x0 := int(srcX)
			y0 := int(srcY)
			x1 := x0 + 1
			y1 := y0 + 1

			if x1 >= srcW {
				x1 = srcW - 1
			}
			if y1 >= srcH {
				y1 = srcH - 1
			}

			xFrac := srcX - float32(x0)
			yFrac := srcY - float32(y0)

			for c := 0; c < channels; c++ {
				v00 := float32(src[(y0*srcW+x0)*channels+c])
				v01 := float32(src[(y0*srcW+x1)*channels+c])
				v10 := float32(src[(y1*srcW+x0)*channels+c])
				v11 := float32(src[(y1*srcW+x1)*channels+c])

				v0 := v00*(1-xFrac) + v01*xFrac
				v1 := v10*(1-xFrac) + v11*xFrac
				value := v0*(1-yFrac) + v1*yFrac

				dst[(y*dstW+x)*channels+c] = uint8(value)
			}
		}
	}
}
