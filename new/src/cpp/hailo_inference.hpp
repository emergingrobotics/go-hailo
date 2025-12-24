#pragma once

#include <hailo/hailort.hpp>
#include <string>
#include <vector>
#include <memory>
#include <cstdint>

namespace hailo_wrapper {

// Detection result from object detection model
struct Detection {
    float x_min;
    float y_min;
    float x_max;
    float y_max;
    float confidence;
    int class_id;
};

// Input information for the model
struct InputInfo {
    int width;
    int height;
    int channels;
    size_t frame_size;
};

// HailoInference - wrapper around HailoRT InferModel API
class HailoInference {
public:
    // Create inference engine from HEF file
    static std::unique_ptr<HailoInference> create(const std::string& hef_path);

    ~HailoInference();

    // Get input requirements
    InputInfo getInputInfo() const;

    // Run inference and get detections
    // Input should be RGB data matching getInputInfo() dimensions
    std::vector<Detection> detect(const uint8_t* input_data, size_t input_size);

    // Count people (class_id == 0 in COCO)
    int detectPeople(const uint8_t* input_data, size_t input_size);

private:
    HailoInference();

    // Parse YOLO-style output into detections
    std::vector<Detection> parseDetections(const std::vector<uint8_t>& output);

    std::unique_ptr<hailort::VDevice> m_vdevice;
    std::shared_ptr<hailort::InferModel> m_infer_model;
    std::shared_ptr<hailort::ConfiguredInferModel> m_configured_model;
    hailort::ConfiguredInferModel::Bindings m_bindings;

    InputInfo m_input_info;
    size_t m_output_size;

    // Detection parameters
    float m_confidence_threshold = 0.5f;
    float m_nms_threshold = 0.45f;
};

} // namespace hailo_wrapper
