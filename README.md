# Purple Hailo

Go runtime for Hailo-8 AI Accelerator on Raspberry Pi 5.

## Prerequisites

### Hailo Model Zoo

Add the Hailo Model Zoo as a git submodule to access model configurations:

```bash
git submodule add https://github.com/hailo-ai/hailo_model_zoo.git
git submodule update --init --recursive
```

### Pre-compiled Models

Pre-compiled HEF models are downloaded from Hailo's S3 bucket into the `models/` directory:

```bash
mkdir -p models
curl -L -o models/yolox_s_leaky_hailo8.hef \
  "https://hailo-model-zoo.s3.eu-west-2.amazonaws.com/ModelZoo/Compiled/v2.10.0/hailo8/yolox_s_leaky.hef"
```

## Usage

### Person Detector

```bash
make build-examples
./bin/person-detector ./test-images/two.jpg
```
