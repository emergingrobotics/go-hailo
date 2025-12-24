#include "hailo_c_api.h"
#include "hailo_inference.hpp"
#include <string>
#include <mutex>

// Thread-local error message storage
static thread_local std::string g_last_error;

// Set error message
static void set_error(const std::string& msg) {
    g_last_error = msg;
}

extern "C" {

hailo_inference_t* hailo_create(const char* hef_path) {
    try {
        auto engine = hailo_wrapper::HailoInference::create(hef_path);
        return reinterpret_cast<hailo_inference_t*>(engine.release());
    } catch (const std::exception& e) {
        set_error(e.what());
        return nullptr;
    }
}

void hailo_destroy(hailo_inference_t* h) {
    if (h) {
        auto* engine = reinterpret_cast<hailo_wrapper::HailoInference*>(h);
        delete engine;
    }
}

const char* hailo_get_last_error(void) {
    return g_last_error.c_str();
}

hailo_input_info_t hailo_get_input_info(hailo_inference_t* h) {
    hailo_input_info_t info = {0, 0, 0, 0};
    if (!h) {
        set_error("Invalid handle");
        return info;
    }

    auto* engine = reinterpret_cast<hailo_wrapper::HailoInference*>(h);
    auto cpp_info = engine->getInputInfo();

    info.width = cpp_info.width;
    info.height = cpp_info.height;
    info.channels = cpp_info.channels;
    info.frame_size = cpp_info.frame_size;

    return info;
}

int hailo_detect_people(hailo_inference_t* h,
                        const uint8_t* input_data,
                        size_t input_size) {
    if (!h) {
        set_error("Invalid handle");
        return -1;
    }

    try {
        auto* engine = reinterpret_cast<hailo_wrapper::HailoInference*>(h);
        return engine->detectPeople(input_data, input_size);
    } catch (const std::exception& e) {
        set_error(e.what());
        return -1;
    }
}

int hailo_detect(hailo_inference_t* h,
                 const uint8_t* input_data,
                 size_t input_size,
                 hailo_wrapper_detection_t* detections,
                 int max_detections) {
    if (!h) {
        set_error("Invalid handle");
        return -1;
    }

    try {
        auto* engine = reinterpret_cast<hailo_wrapper::HailoInference*>(h);
        auto cpp_detections = engine->detect(input_data, input_size);

        int count = 0;
        for (const auto& det : cpp_detections) {
            if (count >= max_detections) break;

            detections[count].x_min = det.x_min;
            detections[count].y_min = det.y_min;
            detections[count].x_max = det.x_max;
            detections[count].y_max = det.y_max;
            detections[count].confidence = det.confidence;
            detections[count].class_id = det.class_id;
            count++;
        }

        return count;
    } catch (const std::exception& e) {
        set_error(e.what());
        return -1;
    }
}

} // extern "C"
