#include "hailo_inference.hpp"
#include <stdexcept>
#include <algorithm>
#include <cmath>
#include <chrono>

namespace hailo_wrapper {

HailoInference::HailoInference() = default;
HailoInference::~HailoInference() = default;

std::unique_ptr<HailoInference> HailoInference::create(const std::string& hef_path) {
    auto engine = std::unique_ptr<HailoInference>(new HailoInference());

    // Create virtual device (auto-discovers Hailo hardware)
    auto vdevice_exp = hailort::VDevice::create();
    if (!vdevice_exp) {
        throw std::runtime_error("Failed to create VDevice: " +
            std::to_string(static_cast<int>(vdevice_exp.status())));
    }
    engine->m_vdevice = vdevice_exp.release();

    // Create InferModel from HEF
    auto infer_model_exp = engine->m_vdevice->create_infer_model(hef_path);
    if (!infer_model_exp) {
        throw std::runtime_error("Failed to create InferModel: " +
            std::to_string(static_cast<int>(infer_model_exp.status())));
    }
    engine->m_infer_model = infer_model_exp.release();

    // Configure the model
    auto configured_exp = engine->m_infer_model->configure();
    if (!configured_exp) {
        throw std::runtime_error("Failed to configure model: " +
            std::to_string(static_cast<int>(configured_exp.status())));
    }
    engine->m_configured_model = std::make_shared<hailort::ConfiguredInferModel>(
        configured_exp.release());

    // Create bindings
    auto bindings_exp = engine->m_configured_model->create_bindings();
    if (!bindings_exp) {
        throw std::runtime_error("Failed to create bindings: " +
            std::to_string(static_cast<int>(bindings_exp.status())));
    }
    engine->m_bindings = bindings_exp.release();

    // Get input info - input() returns Expected<InferStream>, need to unwrap
    auto input_exp = engine->m_infer_model->input();
    if (!input_exp) {
        throw std::runtime_error("Failed to get input stream: " +
            std::to_string(static_cast<int>(input_exp.status())));
    }
    const auto& input = input_exp.value();
    auto shape = input.shape();
    engine->m_input_info.height = shape.height;
    engine->m_input_info.width = shape.width;
    engine->m_input_info.channels = shape.features;
    engine->m_input_info.frame_size = input.get_frame_size();

    // Get output size
    auto output_exp = engine->m_infer_model->output();
    if (!output_exp) {
        throw std::runtime_error("Failed to get output stream: " +
            std::to_string(static_cast<int>(output_exp.status())));
    }
    engine->m_output_size = output_exp.value().get_frame_size();

    return engine;
}

InputInfo HailoInference::getInputInfo() const {
    return m_input_info;
}

std::vector<Detection> HailoInference::detect(const uint8_t* input_data, size_t input_size) {
    if (input_size != m_input_info.frame_size) {
        throw std::runtime_error("Input size mismatch: expected " +
            std::to_string(m_input_info.frame_size) + ", got " + std::to_string(input_size));
    }

    // Set input buffer
    auto status = m_bindings.input()->set_buffer(
        hailort::MemoryView(const_cast<uint8_t*>(input_data), input_size));
    if (status != HAILO_SUCCESS) {
        throw std::runtime_error("Failed to set input buffer");
    }

    // Allocate output buffer
    std::vector<uint8_t> output_buffer(m_output_size);
    status = m_bindings.output()->set_buffer(
        hailort::MemoryView(output_buffer.data(), m_output_size));
    if (status != HAILO_SUCCESS) {
        throw std::runtime_error("Failed to set output buffer");
    }

    // Run inference (with default timeout of 10 seconds)
    status = m_configured_model->run(m_bindings, std::chrono::milliseconds(10000));
    if (status != HAILO_SUCCESS) {
        throw std::runtime_error("Inference failed: " +
            std::to_string(static_cast<int>(status)));
    }

    // Parse and return detections
    return parseDetections(output_buffer);
}

int HailoInference::detectPeople(const uint8_t* input_data, size_t input_size) {
    auto detections = detect(input_data, input_size);

    int count = 0;
    for (const auto& det : detections) {
        // COCO class 0 is "person"
        if (det.class_id == 0 && det.confidence >= m_confidence_threshold) {
            count++;
        }
    }
    return count;
}

// Helper: Intersection over Union for NMS
static float iou(const Detection& a, const Detection& b) {
    float x1 = std::max(a.x_min, b.x_min);
    float y1 = std::max(a.y_min, b.y_min);
    float x2 = std::min(a.x_max, b.x_max);
    float y2 = std::min(a.y_max, b.y_max);

    float intersection = std::max(0.0f, x2 - x1) * std::max(0.0f, y2 - y1);
    float area_a = (a.x_max - a.x_min) * (a.y_max - a.y_min);
    float area_b = (b.x_max - b.x_min) * (b.y_max - b.y_min);
    float union_area = area_a + area_b - intersection;

    return union_area > 0 ? intersection / union_area : 0;
}

std::vector<Detection> HailoInference::parseDetections(const std::vector<uint8_t>& output) {
    std::vector<Detection> detections;

    // YOLO output format varies by model version
    // Common format: [batch, num_detections, 85] where 85 = 4 (box) + 1 (obj_conf) + 80 (classes)
    // Or NMS-processed: [batch, num_detections, 6] where 6 = 4 (box) + 1 (conf) + 1 (class)

    // The HEF post-processing typically outputs in a specific format
    // This is a simplified parser - real implementation depends on model

    const float* data = reinterpret_cast<const float*>(output.data());
    size_t num_floats = output.size() / sizeof(float);

    // Assume format: [x_min, y_min, x_max, y_max, confidence, class_id] per detection
    const size_t detection_size = 6;
    size_t num_detections = num_floats / detection_size;

    for (size_t i = 0; i < num_detections; i++) {
        size_t offset = i * detection_size;

        Detection det;
        det.x_min = data[offset + 0];
        det.y_min = data[offset + 1];
        det.x_max = data[offset + 2];
        det.y_max = data[offset + 3];
        det.confidence = data[offset + 4];
        det.class_id = static_cast<int>(data[offset + 5]);

        // Filter by confidence
        if (det.confidence >= m_confidence_threshold) {
            detections.push_back(det);
        }
    }

    // Apply NMS
    std::sort(detections.begin(), detections.end(),
        [](const Detection& a, const Detection& b) {
            return a.confidence > b.confidence;
        });

    std::vector<Detection> nms_detections;
    std::vector<bool> suppressed(detections.size(), false);

    for (size_t i = 0; i < detections.size(); i++) {
        if (suppressed[i]) continue;

        nms_detections.push_back(detections[i]);

        for (size_t j = i + 1; j < detections.size(); j++) {
            if (suppressed[j]) continue;
            if (detections[i].class_id == detections[j].class_id &&
                iou(detections[i], detections[j]) > m_nms_threshold) {
                suppressed[j] = true;
            }
        }
    }

    return nms_detections;
}

} // namespace hailo_wrapper
