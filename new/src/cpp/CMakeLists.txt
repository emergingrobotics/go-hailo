cmake_minimum_required(VERSION 3.14)
project(hailo_wrapper)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)

# Find HailoRT - installed by hailo-all package on Raspberry Pi
find_package(HailoRT REQUIRED)

# Build the shared library for cgo
add_library(hailo_wrapper SHARED
    hailo_inference.cpp
    hailo_c_api.cpp
)

target_include_directories(hailo_wrapper PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}
)

target_link_libraries(hailo_wrapper
    HailoRT::libhailort
)

# Install rules
install(TARGETS hailo_wrapper
    LIBRARY DESTINATION lib
    ARCHIVE DESTINATION lib
)

install(FILES hailo_c_api.h
    DESTINATION include
)
