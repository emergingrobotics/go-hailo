#ifndef HAILO_C_API_H
#define HAILO_C_API_H

#include <stdint.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

// Opaque handle to inference engine
typedef struct hailo_inference_t hailo_inference_t;

// Input information
typedef struct {
    int width;
    int height;
    int channels;
    size_t frame_size;
} hailo_input_info_t;

// Detection result
typedef struct {
    float x_min;
    float y_min;
    float x_max;
    float y_max;
    float confidence;
    int class_id;
} hailo_wrapper_detection_t;

// Create inference engine from HEF file
// Returns NULL on error, sets error message retrievable via hailo_get_last_error()
hailo_inference_t* hailo_create(const char* hef_path);

// Destroy inference engine and free resources
void hailo_destroy(hailo_inference_t* h);

// Get last error message
const char* hailo_get_last_error(void);

// Get input requirements
hailo_input_info_t hailo_get_input_info(hailo_inference_t* h);

// Run detection and return number of people found
// Input should be RGB data matching hailo_get_input_info() dimensions
// Returns -1 on error
int hailo_detect_people(hailo_inference_t* h,
                        const uint8_t* input_data,
                        size_t input_size);

// Run detection and get all detections
// detections: output array to fill
// max_detections: maximum number of detections to return
// Returns number of detections found, or -1 on error
int hailo_detect(hailo_inference_t* h,
                 const uint8_t* input_data,
                 size_t input_size,
                 hailo_wrapper_detection_t* detections,
                 int max_detections);

#ifdef __cplusplus
}
#endif

#endif // HAILO_C_API_H
