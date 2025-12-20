# Hailo-AI Repository Applicability for Hailo-8 M.2 + Raspberry Pi 5

## Product Overview

**Product:** Waveshare Hailo-8 M.2 AI Accelerator Module
**Amazon Link:** https://www.amazon.com/dp/B0D928WG5L

### Specifications
- **Chip:** Hailo-8 AI Processor
- **Performance:** 26 TOPS (Tera Operations Per Second)
- **Form Factor:** M.2 module
- **Interface:** PCIe
- **Compatibility:** Raspberry Pi 5, Linux/Windows

---

## Repository Applicability Summary

The Hailo-AI GitHub organization (https://github.com/hailo-ai) contains 43+ repositories. Many are specifically for the newer Hailo-10/Hailo-15 Vision Processing Units and do **not** apply to the Hailo-8 M.2 module. Below is a comprehensive analysis.

---

## Applicable Repositories (Hailo-8 + Raspberry Pi 5)

### Core Runtime & Drivers

#### 1. [hailort](https://github.com/hailo-ai/hailort)
**Applicability: ESSENTIAL**

The HailoRT is the core inference runtime framework required for all Hailo-8 operations.

| Branch/Tag | Device Support | Notes |
|------------|----------------|-------|
| `master` | Hailo-10/15 only | Do NOT use for Hailo-8 |
| `hailo8` | Hailo-8, Hailo-8R, Hailo-8L | **Use this branch** |
| `v4.23.0` (tag) | Hailo-8 | Latest stable release for Hailo-8 |

- Provides C/C++ and Python APIs for inference
- Includes command-line tools for device control
- GStreamer integration support

```bash
# Clone the correct branch
git clone -b hailo8 https://github.com/hailo-ai/hailort.git

# Or checkout a specific version tag
git clone https://github.com/hailo-ai/hailort.git
cd hailort
git checkout v4.23.0
```

**Available v4.x Tags (Hailo-8):**
- v4.23.0 (Sep 2025) - Latest
- v4.22.0 (Jul 2025)
- v4.21.0 (Apr 2025)
- v4.20.1 (Jan 2025)
- v4.20.0 (Dec 2024)

#### 2. [hailort-drivers](https://github.com/hailo-ai/hailort-drivers)
**Applicability: ESSENTIAL**

PCIe drivers required for the M.2 module to communicate with the Raspberry Pi 5.

| Branch/Tag | Device Support | Notes |
|------------|----------------|-------|
| `master` | Hailo-10/15 only | Do NOT use for Hailo-8 |
| `hailo8` | Hailo-8, Hailo-8R, Hailo-8L | **Use this branch** |
| `v4.23.0` (tag) | Hailo-8 | Latest stable release |

- Enables PCIe interface communication
- Required for firmware loading

```bash
# Clone the correct branch
git clone -b hailo8 https://github.com/hailo-ai/hailort-drivers.git
```

---

### Raspberry Pi 5 Specific

#### 3. [hailo-rpi5-examples](https://github.com/hailo-ai/hailo-rpi5-examples)
**Applicability: ESSENTIAL**

Official example applications specifically designed for Raspberry Pi 5 with Hailo accelerators.

| Branch | Purpose | Notes |
|--------|---------|-------|
| `main` | Production | **Use this branch (default)** |
| `dev` | Development | Bleeding edge features |

- Supports both Hailo-8 (26 TOPS) and Hailo-8L (13 TOPS)
- Includes pipelines for:
  - Object Detection (with tracker support)
  - Pose Estimation
  - Instance Segmentation
  - Depth Estimation
- Works with Raspberry Pi AI Kit, AI HAT, and M.2 modules
- Python-based examples with installation script

```bash
# Clone (default main branch is correct)
git clone https://github.com/hailo-ai/hailo-rpi5-examples.git
```

---

### Model Zoo & AI Models

#### 4. [hailo_model_zoo](https://github.com/hailo-ai/hailo_model_zoo)
**Applicability: ESSENTIAL**

Pre-trained models and tools for model optimization and compilation.

| Branch/Tag | Device Support | Notes |
|------------|----------------|-------|
| `master` | Hailo-10/15 only | Do NOT use for Hailo-8 |
| `v2.17` (tag) | Hailo-8, Hailo-8R, Hailo-8L | **Latest for Hailo-8** |

**Important:** There is no `v2.x` branch - use version **tags** instead.

- Includes pre-trained models for:
  - Classification
  - Object Detection
  - Semantic Segmentation
  - Pose Estimation
  - And more
- Model quantization tools
- Compilation to HEF (Hailo Executable Format)
- Performance benchmarking tools

```bash
# Clone and checkout the latest v2.x tag for Hailo-8
git clone https://github.com/hailo-ai/hailo_model_zoo.git
cd hailo_model_zoo
git checkout v2.17
```

**Available v2.x Tags (Hailo-8):**
- v2.17 (Oct 2025) - Latest
- v2.16 (Sep 2025)
- v2.15 (Apr 2025)
- v2.14 (Jan 2025)
- v2.13 (Sep 2024)

---

### Application Frameworks

#### 5. [tappas](https://github.com/hailo-ai/tappas)
**Applicability: HIGHLY RECOMMENDED**

Template AI Application Pipelines - high-performance video processing pipelines.

| Branch/Tag | Device Support | Notes |
|------------|----------------|-------|
| `master` | Hailo-8 AND Hailo-10H | **Use this (supports both)** |
| `v5.1.0` (tag) | Hailo-8 + Hailo-10H | Latest release |

**Key Compatibility Note:** TAPPAS v5.1.0 requires:
- HailoRT v4.23.0 for Hailo-8 devices
- HailoRT v5.1.0 for Hailo-10H devices

- Raspberry Pi OS support included
- GStreamer-based pipelines
- Pre-built application templates
- For RPi5-specific examples, references hailo-rpi5-examples

```bash
# Clone (master branch supports Hailo-8)
git clone https://github.com/hailo-ai/tappas.git
```

**Available Tags:**
- v5.1.0, v5.0.0 (support both Hailo-8 and Hailo-10H)
- v3.31.0, v3.30.0, v3.29.x, v3.28.x, v3.27.0, v3.26.0 (older versions)

#### 6. [Hailo-Application-Code-Examples](https://github.com/hailo-ai/Hailo-Application-Code-Examples)
**Applicability: HIGHLY RECOMMENDED**

C++ and Python code examples for various AI tasks.

| Branch | Purpose | Notes |
|--------|---------|-------|
| `main` | Production | **Use this branch (default)** |

- Explicitly supports Hailo-8, Hailo-8L, Hailo-10, and Hailo-15
- Runtime examples:
  - Object detection and tracking
  - Image classification
  - Semantic/instance segmentation
  - Pose estimation
  - Speech recognition
  - Super-resolution
  - YOLO11 OBB (oriented object detection)
- Compilation optimization examples

```bash
# Clone (default main branch is correct)
git clone https://github.com/hailo-ai/Hailo-Application-Code-Examples.git
```

#### 7. [hailo-apps-infra](https://github.com/hailo-ai/hailo-apps-infra)
**Applicability: RECOMMENDED**

Python infrastructure library for building AI applications.

| Branch | Purpose | Notes |
|--------|---------|-------|
| `main` | Production | **Use this branch (default)** |
| `dev` | Development | Bleeding edge |

- Supports x86_64 Ubuntu and Raspberry Pi
- Pre-built applications:
  - `hailo-detect` - Object detection
  - `hailo-pose` - Pose estimation
  - `hailo-seg` - Segmentation
- Modular components for custom development

```bash
# Clone (default main branch is correct)
git clone https://github.com/hailo-ai/hailo-apps-infra.git
```

---

### Specialized Applications

#### 8. [hailo-CLIP](https://github.com/hailo-ai/hailo-CLIP)
**Applicability: COMPATIBLE**

Real-time zero-shot classification using CLIP.

| Branch | Purpose | Notes |
|--------|---------|-------|
| `main` | Production | **Use this branch (only branch)** |

- Explicitly supports Hailo-8 and Hailo-8L
- Works on Raspberry Pi 5 (8GB recommended)
- Tested with TAPPAS 3.30.0 and 3.31.0
- Accelerates image embeddings on Hailo, text embeddings on host

```bash
git clone https://github.com/hailo-ai/hailo-CLIP.git
```

#### 9. [hailo-BEV](https://github.com/hailo-ai/hailo-BEV)
**Applicability: COMPATIBLE**

Bird's Eye View perception for autonomous vehicles.

| Branch | Purpose | Notes |
|--------|---------|-------|
| `main` | Production | **Use this branch (only branch)** |

- Explicitly supports Hailo-8
- Uses PETRv2 model
- Processes multi-camera inputs from nuScenes dataset
- Generates 3D bounding boxes

```bash
git clone https://github.com/hailo-ai/hailo-BEV.git
```

---

### Yocto/Embedded Linux Integration

#### 10. [meta-hailo](https://github.com/hailo-ai/meta-hailo)
**Applicability: SPECIALIZED**

Yocto layers for integrating Hailo software stack.

| Branch | Yocto Version | Hailo-8 Support | Notes |
|--------|---------------|-----------------|-------|
| `master` | Various | Yes | Default |
| `kirkstone` | Kirkstone | Yes | **Recommended for new projects** |
| `hailo8-kirkstone` | Kirkstone | Yes | Hailo-8 specific |
| `hailo8-scarthgap` | Scarthgap | Yes | Hailo-8 + newer Yocto |

- Includes Hailo-8 firmware recipes
- Provides:
  - PCIe driver
  - HailoRT library
  - Python API
  - GStreamer integration
  - TAPPAS framework
- Use if building custom embedded Linux images

```bash
# For Kirkstone-based Yocto builds with Hailo-8
git clone -b hailo8-kirkstone https://github.com/hailo-ai/meta-hailo.git
```

---

## NOT Applicable Repositories (Hailo-10/Hailo-15 Only)

The following repositories are for Hailo's Vision Processing Units (VPU) and do **not** support the Hailo-8 M.2 module:

| Repository | Reason |
|------------|--------|
| `hailort` (master branch) | Hailo-10/15 only |
| `hailort-drivers` (master branch) | Hailo-10/15 only |
| `hailo_model_zoo` (master branch) | Hailo-10/15 only |
| `hailo_model_zoo_genai` | Hailo-10H only (LLM support) |
| `hailo-camera-apps` | Hailo-15 only |
| `meta-hailo-soc` | Hailo-15 VPU only |
| `linux-yocto-hailo` | Hailo-15 VPU only |
| `hailo-u-boot` | Hailo-15 VPU only |
| `arm-trusted-firmware` | Hailo-15 VPU only |
| `hailodsp` | Hailo-15 DSP accelerator |
| `hailo-media-library` | Hailo-15 VPU only |
| `hailo-imaging` | Hailo-15 VPU only |
| `hailo-camera-configurations` | Hailo-15 VPU only |
| `Hailo-SoC-Profiler` | Hailo SoC (VPU) profiling |

---

## Forked ML Framework Repositories

These are forks of popular ML frameworks used for model training and conversion. They may contain Hailo-specific modifications for model optimization:

| Repository | Purpose |
|------------|---------|
| `ultralytics` | YOLOv8 training |
| `yolov5` | YOLOv5 training |
| `YOLOX` | YOLOX variant |
| `DAMO-YOLO` | DAMO-YOLO training |
| `pytorch-YOLOv4` | YOLOv4 training |
| `nanodet` | Lightweight detection |
| `mmdetection` | OpenMMLab detection |
| `mmsegmentation` | OpenMMLab segmentation |
| `mmpose` | OpenMMLab pose estimation |
| `detr` | DETR transformers |
| `deep-person-reid` | Person re-identification |
| `insightface` | Face analysis |
| `centerpose` | Pose estimation |
| `yolact` | Instance segmentation |
| `darknet` | YOLO framework |
| `LPRNet_Pytorch` | License plate recognition |
| `PETR` | 3D object detection |
| `pytorch-image-models` | Pre-trained models |
| `onnxruntime` | ONNX inference |

**Note:** These are useful for training custom models that can then be compiled for Hailo-8 using the model zoo tools.

---

## Quick Start Recommendations

For getting started with the Hailo-8 M.2 on Raspberry Pi 5:

### 1. Essential Setup
```bash
# Install HailoRT (use hailo8 branch or Raspberry Pi packages)
# The RPi5 typically uses apt packages from Hailo's repository

# Clone examples
git clone https://github.com/hailo-ai/hailo-rpi5-examples.git
cd hailo-rpi5-examples
./install.sh
```

### 2. For Model Development
```bash
# Clone model zoo and checkout the correct tag for Hailo-8
git clone https://github.com/hailo-ai/hailo_model_zoo.git
cd hailo_model_zoo
git checkout v2.17  # Latest v2.x tag for Hailo-8
```

### 3. For Advanced Pipelines
```bash
# Clone TAPPAS for GStreamer pipelines (master supports Hailo-8)
git clone https://github.com/hailo-ai/tappas.git

# Clone application examples
git clone https://github.com/hailo-ai/Hailo-Application-Code-Examples.git
```

---

## Version Compatibility Matrix

### Verified Compatible Versions (as of Dec 2025)

| Component | Branch/Tag | Version | Notes |
|-----------|------------|---------|-------|
| HailoRT | `hailo8` branch | v4.23.0 | Latest for Hailo-8 |
| HailoRT Drivers | `hailo8` branch | v4.23.0 | Must match HailoRT |
| Model Zoo | `v2.17` tag | v2.17 | Use tags, not branches |
| TAPPAS | `master` branch | v5.1.0 | Requires HailoRT v4.23.0 |
| RPi5 Examples | `main` branch | - | Default branch |
| Application Examples | `main` branch | - | Default branch |
| hailo-apps-infra | `main` branch | - | Default branch |
| hailo-CLIP | `main` branch | - | Only branch |
| hailo-BEV | `main` branch | - | Only branch |
| meta-hailo | `hailo8-kirkstone` | - | For Yocto Kirkstone |

### Version Alignment Rules

1. **HailoRT and Drivers**: Must use matching versions (e.g., both v4.23.0)
2. **TAPPAS**: Check release notes for required HailoRT version
3. **Model Zoo**: v2.x tags for Hailo-8, v5.x/master for Hailo-10/15

---

## Summary Table

| Repository | Branch/Tag for Hailo-8 | Default Works? | Priority |
|------------|------------------------|----------------|----------|
| hailort | `hailo8` branch or v4.x tags | NO | Essential |
| hailort-drivers | `hailo8` branch or v4.x tags | NO | Essential |
| hailo-rpi5-examples | `main` (default) | YES | Essential |
| hailo_model_zoo | v2.x tags (e.g., v2.17) | NO | Essential |
| tappas | `master` (default) | YES | High |
| Hailo-Application-Code-Examples | `main` (default) | YES | High |
| hailo-apps-infra | `main` (default) | YES | Medium |
| hailo-CLIP | `main` (default) | YES | Optional |
| hailo-BEV | `main` (default) | YES | Optional |
| meta-hailo | `hailo8-kirkstone` or `hailo8-scarthgap` | NO | Specialized |

**Legend:**
- "Default Works?" = Can you just `git clone` without specifying a branch?
- NO = Must specify branch/tag explicitly for Hailo-8 support

---

## Additional Resources

- **Hailo Developer Zone:** https://hailo.ai/developer-zone/ (registration required)
- **Hailo Community Forum:** For technical support and discussions
- **Raspberry Pi AI Documentation:** Official Raspberry Pi documentation for AI Kit/HAT

---

*Document generated: 2025-12-20*
