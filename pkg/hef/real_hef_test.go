//go:build unit

package hef

import (
	"os"
	"path/filepath"
	"testing"
)

const testModelsDir = "../../models"

func getTestHefPath(t *testing.T) string {
	t.Helper()

	// Try to find a real HEF file
	paths := []string{
		filepath.Join(testModelsDir, "yolox_s_leaky_hailo8.hef"),
		filepath.Join(testModelsDir, "yolov8n.hef"),
		filepath.Join(testModelsDir, "resnet50.hef"),
		"testdata/model.hef",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	t.Skip("No HEF file available for testing")
	return ""
}

func TestParseRealHefFile(t *testing.T) {
	hefPath := getTestHefPath(t)

	data, err := os.ReadFile(hefPath)
	if err != nil {
		t.Fatalf("failed to read HEF file: %v", err)
	}

	// Parse header
	header, err := ParseHeader(data)
	if err != nil {
		t.Fatalf("failed to parse header: %v", err)
	}

	// Verify magic
	if header.Magic != HefMagic {
		t.Errorf("magic = 0x%08X, expected 0x%08X", header.Magic, HefMagic)
	}

	// Verify version is supported
	if header.Version > HefVersionV3 {
		t.Errorf("unsupported version: %d", header.Version)
	}

	t.Logf("Parsed HEF: version=%d, protoSize=%d", header.Version, header.HefProtoSize)
}

func TestParseRealHefHeaderByVersion(t *testing.T) {
	hefPath := getTestHefPath(t)

	data, err := os.ReadFile(hefPath)
	if err != nil {
		t.Fatalf("failed to read HEF file: %v", err)
	}

	// Parse base header
	header, err := ParseHeader(data)
	if err != nil {
		t.Fatalf("failed to parse header: %v", err)
	}

	// Parse version-specific header
	switch header.Version {
	case HefVersionV0:
		v0Header, err := ParseHeaderV0(data)
		if err != nil {
			t.Fatalf("failed to parse V0 header: %v", err)
		}
		t.Logf("V0 Header: MD5=%x", v0Header.ExpectedMD5)

	case HefVersionV2:
		v2Header, err := ParseHeaderV2(data)
		if err != nil {
			t.Fatalf("failed to parse V2 header: %v", err)
		}
		t.Logf("V2 Header: XXH3=0x%016X, CCWSSize=%d",
			v2Header.Xxh3Hash, v2Header.CcwsSize)

	case HefVersionV3:
		v3Header, err := ParseHeaderV3(data)
		if err != nil {
			t.Fatalf("failed to parse V3 header: %v", err)
		}
		t.Logf("V3 Header: XXH3=0x%016X, CCWSSizeWithPadding=%d, AdditionalInfoSize=%d",
			v3Header.Xxh3Hash, v3Header.CcwsSizeWithPadding, v3Header.AdditionalInfoSize)

	default:
		t.Logf("Version %d: Using base header only", header.Version)
	}
}

func TestRealHefFileSizes(t *testing.T) {
	hefPath := getTestHefPath(t)

	info, err := os.Stat(hefPath)
	if err != nil {
		t.Fatalf("failed to stat HEF file: %v", err)
	}

	data, err := os.ReadFile(hefPath)
	if err != nil {
		t.Fatalf("failed to read HEF file: %v", err)
	}

	header, err := ParseHeader(data)
	if err != nil {
		t.Fatalf("failed to parse header: %v", err)
	}

	headerSize, err := HeaderSize(header.Version)
	if err != nil {
		t.Fatalf("failed to get header size: %v", err)
	}

	// Parser now correctly reads proto size in big-endian
	protoSize := header.HefProtoSize

	// Proto size should be reasonable
	if protoSize == 0 {
		t.Error("HefProtoSize should not be 0")
	}
	if protoSize > uint32(info.Size()) {
		t.Errorf("HefProtoSize (%d) exceeds file size (%d)",
			protoSize, info.Size())
	}

	// Calculate expected minimum size
	minSize := int64(headerSize) + int64(protoSize)
	if info.Size() < minSize {
		t.Errorf("file size (%d) < header + proto (%d)",
			info.Size(), minSize)
	}

	t.Logf("HEF file: %d bytes total, %d header, %d proto",
		info.Size(), headerSize, protoSize)
}

func TestRealHefDataRegions(t *testing.T) {
	hefPath := getTestHefPath(t)

	data, err := os.ReadFile(hefPath)
	if err != nil {
		t.Fatalf("failed to read HEF file: %v", err)
	}

	header, err := ParseHeader(data)
	if err != nil {
		t.Fatalf("failed to parse header: %v", err)
	}

	headerSize, _ := HeaderSize(header.Version)
	protoSize := header.HefProtoSize

	// Verify we can access proto region
	protoStart := headerSize
	protoEnd := protoStart + int(protoSize)

	if protoEnd > len(data) {
		t.Fatalf("proto region exceeds file: end=%d, filesize=%d",
			protoEnd, len(data))
	}

	protoData := data[protoStart:protoEnd]

	// Proto data should not be all zeros
	allZero := true
	for _, b := range protoData[:min(1000, len(protoData))] {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("proto data appears to be all zeros")
	}

	t.Logf("Proto region: bytes %d-%d (%d bytes)",
		protoStart, protoEnd, protoSize)
}

func TestMultipleHefFiles(t *testing.T) {
	// Try to find all HEF files in models directory
	matches, err := filepath.Glob(filepath.Join(testModelsDir, "*.hef"))
	if err != nil {
		t.Fatalf("glob failed: %v", err)
	}

	if len(matches) == 0 {
		t.Skip("No HEF files found in models directory")
	}

	for _, path := range matches {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read: %v", err)
			}

			header, err := ParseHeader(data)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}

			if header.Magic != HefMagic {
				t.Error("invalid magic")
			}

			t.Logf("version=%d, protoSize=%d", header.Version, header.HefProtoSize)
		})
	}
}

func TestHefProtobufRegionNonEmpty(t *testing.T) {
	hefPath := getTestHefPath(t)

	data, err := os.ReadFile(hefPath)
	if err != nil {
		t.Fatalf("failed to read HEF file: %v", err)
	}

	header, err := ParseHeader(data)
	if err != nil {
		t.Fatalf("failed to parse header: %v", err)
	}

	headerSize, _ := HeaderSize(header.Version)
	protoSize := header.HefProtoSize

	if protoSize < 100 {
		t.Errorf("proto size %d seems too small", protoSize)
	}

	protoData := data[headerSize : headerSize+int(protoSize)]

	// Count non-zero bytes
	nonZero := 0
	for _, b := range protoData {
		if b != 0 {
			nonZero++
		}
	}

	// At least 30% should be non-zero for valid protobuf
	ratio := float64(nonZero) / float64(len(protoData))
	if ratio < 0.3 {
		t.Errorf("only %.1f%% non-zero bytes in proto data", ratio*100)
	}

	t.Logf("Proto data: %.1f%% non-zero bytes", ratio*100)
}

func BenchmarkHefParsing(b *testing.B) {
	// Find a HEF file for benchmarking
	paths := []string{
		filepath.Join(testModelsDir, "yolox_s_leaky_hailo8.hef"),
		filepath.Join(testModelsDir, "yolov8n.hef"),
	}

	var hefPath string
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			hefPath = path
			break
		}
	}

	if hefPath == "" {
		b.Skip("No HEF file available")
	}

	data, err := os.ReadFile(hefPath)
	if err != nil {
		b.Fatalf("failed to read: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseHeader(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHefV3Parsing(b *testing.B) {
	paths := []string{
		filepath.Join(testModelsDir, "yolox_s_leaky_hailo8.hef"),
	}

	var data []byte
	for _, path := range paths {
		var err error
		data, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}

	if data == nil {
		b.Skip("No HEF file available")
	}

	header, _ := ParseHeader(data)
	if header.Version != HefVersionV3 {
		b.Skip("HEF is not V3")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseHeaderV3(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Test full HEF parsing with protobuf
func TestParseFullHefFile(t *testing.T) {
	hefPath := getTestHefPath(t)

	hef, err := Parse(hefPath)
	if err != nil {
		t.Fatalf("failed to parse HEF: %v", err)
	}

	t.Logf("HEF version: %d", hef.Version)
	t.Logf("Device arch: %s", hef.DeviceArch)
	t.Logf("Network groups: %d", len(hef.NetworkGroups))

	for i, ng := range hef.NetworkGroups {
		t.Logf("  Network Group %d: %s", i, ng.Name)
		t.Logf("    Inputs: %d, Outputs: %d", len(ng.InputStreams), len(ng.OutputStreams))
		t.Logf("    Bottleneck FPS: %.2f", ng.BottleneckFps)

		for _, stream := range ng.InputStreams {
			t.Logf("    Input: %s, shape=%dx%dx%d, format=%s",
				stream.Name, stream.Shape.Height, stream.Shape.Width, stream.Shape.Features,
				stream.Format.Type)
		}

		for _, stream := range ng.OutputStreams {
			t.Logf("    Output: %s, shape=%dx%dx%d, quant=(zp=%.2f, scale=%.6f)",
				stream.Name, stream.Shape.Height, stream.Shape.Width, stream.Shape.Features,
				stream.QuantInfo.ZeroPoint, stream.QuantInfo.Scale)
		}

		if ng.HasNmsOutput() {
			nmsInfo := ng.GetNmsInfo()
			t.Logf("    NMS: classes=%d, max_bbox=%d",
				nmsInfo.NmsShape.NumberOfClasses, nmsInfo.NmsShape.MaxBboxesPerClass)
		}
	}

	// Basic validation
	if len(hef.NetworkGroups) == 0 {
		t.Error("expected at least one network group")
	}
}

func TestParseHefNetworkGroupInfo(t *testing.T) {
	hefPath := getTestHefPath(t)

	hef, err := Parse(hefPath)
	if err != nil {
		t.Fatalf("failed to parse HEF: %v", err)
	}

	ng, err := hef.GetDefaultNetworkGroup()
	if err != nil {
		t.Fatalf("failed to get default network group: %v", err)
	}

	// For YOLOX model, we expect specific structure
	t.Logf("Network group: %s", ng.Name)

	// Should have at least one input and output
	if len(ng.InputStreams) == 0 {
		t.Error("expected at least one input stream")
	}
	if len(ng.OutputStreams) == 0 {
		t.Error("expected at least one output stream")
	}

	// Check input stream properties
	if len(ng.InputStreams) > 0 {
		input := ng.InputStreams[0]
		if input.Direction != StreamDirectionInput {
			t.Error("input stream should have input direction")
		}
		// YOLOX typically has 640x640x3 input
		if input.Shape.Features != 3 && input.Shape.Features != 0 {
			t.Logf("Input features: %d (expected 3 for RGB)", input.Shape.Features)
		}
	}
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
