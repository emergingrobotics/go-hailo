.PHONY: all build-cpp build-go clean install test

# Default target
all: build-cpp build-go

# Build the C++ shared library
build-cpp:
	@echo "Building C++ library..."
	mkdir -p src/cpp/build
	cd src/cpp/build && cmake .. && make

# Build the Go binary
build-go: build-cpp
	@echo "Building Go binary..."
	CGO_ENABLED=1 go build -o detect ./cmd/detect

# Install the C++ library system-wide (requires sudo)
install:
	@echo "Installing C++ library..."
	cd src/cpp/build && sudo make install
	sudo ldconfig

# Clean build artifacts
clean:
	rm -rf src/cpp/build
	rm -f detect

# Run a test with a sample image
test: build-go
	@echo "Running test..."
	@if [ -f "models/yolov5s.hef" ] && [ -f "test.jpg" ]; then \
		LD_LIBRARY_PATH=src/cpp/build:$$LD_LIBRARY_PATH ./detect -model models/yolov5s.hef -image test.jpg; \
	else \
		echo "Place a HEF model in models/ and test.jpg in project root to run test"; \
	fi

# Help
help:
	@echo "Available targets:"
	@echo "  all        - Build C++ library and Go binary (default)"
	@echo "  build-cpp  - Build only the C++ shared library"
	@echo "  build-go   - Build the Go binary (builds C++ first)"
	@echo "  install    - Install C++ library system-wide (requires sudo)"
	@echo "  clean      - Remove build artifacts"
	@echo "  test       - Run test with sample image"
	@echo "  help       - Show this help"
