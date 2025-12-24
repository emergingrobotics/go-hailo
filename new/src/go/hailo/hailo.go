// Package hailo provides Go bindings for Hailo-8 AI accelerator inference.
//
// This package wraps the official HailoRT C++ SDK via cgo, providing a simple
// interface for running neural network inference on Hailo hardware.
package hailo

/*
#cgo CFLAGS: -I${SRCDIR}/../../cpp
#cgo LDFLAGS: -L${SRCDIR}/../../cpp/build -lhailo_wrapper -lhailort -lstdc++

#include "hailo_c_api.h"
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"unsafe"
)

// InputInfo describes the model's input requirements.
type InputInfo struct {
	Width     int
	Height    int
	Channels  int
	FrameSize int
}

// Detection represents a single object detection result.
type Detection struct {
	XMin       float32
	YMin       float32
	XMax       float32
	YMax       float32
	Confidence float32
	ClassID    int
}

// Inference represents a Hailo inference engine instance.
type Inference struct {
	handle *C.hailo_inference_t
}

// NewInference creates a new inference engine from a HEF model file.
// The HEF file should be a compiled Hailo model (e.g., yolov5s.hef).
func NewInference(hefPath string) (*Inference, error) {
	cPath := C.CString(hefPath)
	defer C.free(unsafe.Pointer(cPath))

	handle := C.hailo_create(cPath)
	if handle == nil {
		errMsg := C.GoString(C.hailo_get_last_error())
		return nil, errors.New("failed to create inference: " + errMsg)
	}

	return &Inference{handle: handle}, nil
}

// Close releases resources associated with the inference engine.
func (i *Inference) Close() {
	if i.handle != nil {
		C.hailo_destroy(i.handle)
		i.handle = nil
	}
}

// GetInputInfo returns the model's input requirements.
func (i *Inference) GetInputInfo() InputInfo {
	info := C.hailo_get_input_info(i.handle)
	return InputInfo{
		Width:     int(info.width),
		Height:    int(info.height),
		Channels:  int(info.channels),
		FrameSize: int(info.frame_size),
	}
}

// DetectPeople runs inference and returns the number of people detected.
// The input should be RGB image data matching the model's input dimensions.
func (i *Inference) DetectPeople(imageData []byte) (int, error) {
	if len(imageData) == 0 {
		return 0, errors.New("empty image data")
	}

	count := C.hailo_detect_people(
		i.handle,
		(*C.uint8_t)(unsafe.Pointer(&imageData[0])),
		C.size_t(len(imageData)),
	)

	if count < 0 {
		errMsg := C.GoString(C.hailo_get_last_error())
		return 0, errors.New("detection failed: " + errMsg)
	}

	return int(count), nil
}

// Detect runs inference and returns all detected objects.
// The input should be RGB image data matching the model's input dimensions.
func (i *Inference) Detect(imageData []byte) ([]Detection, error) {
	if len(imageData) == 0 {
		return nil, errors.New("empty image data")
	}

	// Allocate buffer for detections
	const maxDetections = 100
	cDetections := make([]C.hailo_detection_t, maxDetections)

	count := C.hailo_detect(
		i.handle,
		(*C.uint8_t)(unsafe.Pointer(&imageData[0])),
		C.size_t(len(imageData)),
		&cDetections[0],
		C.int(maxDetections),
	)

	if count < 0 {
		errMsg := C.GoString(C.hailo_get_last_error())
		return nil, errors.New("detection failed: " + errMsg)
	}

	// Convert to Go slice
	detections := make([]Detection, count)
	for j := 0; j < int(count); j++ {
		detections[j] = Detection{
			XMin:       float32(cDetections[j].x_min),
			YMin:       float32(cDetections[j].y_min),
			XMax:       float32(cDetections[j].x_max),
			YMax:       float32(cDetections[j].y_max),
			Confidence: float32(cDetections[j].confidence),
			ClassID:    int(cDetections[j].class_id),
		}
	}

	return detections, nil
}
