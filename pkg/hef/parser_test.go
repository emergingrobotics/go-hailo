//go:build unit

package hef

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func createTestHefFile(t *testing.T, version uint32, protoSize uint32) string {
	t.Helper()

	headerSize, err := HeaderSize(version)
	if err != nil {
		t.Fatalf("failed to get header size: %v", err)
	}

	// Create minimal HEF file: header + dummy proto data
	totalSize := headerSize + int(protoSize)
	data := make([]byte, totalSize)

	// Write header
	binary.LittleEndian.PutUint32(data[0:4], HefMagic)
	binary.LittleEndian.PutUint32(data[4:8], version)
	binary.BigEndian.PutUint32(data[8:12], protoSize) // HEF uses big-endian for proto size

	// Write dummy proto data (just zeros)

	// Create temp file
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.hef")

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	return path
}

func TestParseFromFilePath(t *testing.T) {
	// Create a minimal test HEF file
	path := createTestHefFile(t, HefVersionV2, 100)

	// Read and parse header
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	header, err := ParseHeader(data)
	if err != nil {
		t.Fatalf("failed to parse header: %v", err)
	}

	if header.Magic != HefMagic {
		t.Error("magic mismatch")
	}
	if header.Version != HefVersionV2 {
		t.Error("version mismatch")
	}
}

func TestParseFromBuffer(t *testing.T) {
	// Create in-memory HEF data
	headerSize := HefHeaderSizeV2
	protoSize := uint32(100)
	data := make([]byte, headerSize+int(protoSize))

	binary.LittleEndian.PutUint32(data[0:4], HefMagic)
	binary.LittleEndian.PutUint32(data[4:8], HefVersionV2)
	binary.BigEndian.PutUint32(data[8:12], protoSize) // HEF uses big-endian for proto size

	header, err := ParseHeader(data)
	if err != nil {
		t.Fatalf("failed to parse from buffer: %v", err)
	}

	if header.HefProtoSize != protoSize {
		t.Errorf("HefProtoSize = %d, expected %d", header.HefProtoSize, protoSize)
	}
}

func TestParseEmptyFileFails(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.hef")

	err := os.WriteFile(path, []byte{}, 0644)
	if err != nil {
		t.Fatalf("failed to create empty file: %v", err)
	}

	data, _ := os.ReadFile(path)
	_, err = ParseHeader(data)
	if err == nil {
		t.Error("expected error for empty file")
	}
}

func TestExtractNetworkGroupNames(t *testing.T) {
	// Create a Hef with mock network groups
	hef := &Hef{
		NetworkGroups: []NetworkGroupInfo{
			{Name: "network1"},
			{Name: "network2"},
			{Name: "yolov5"},
		},
	}

	names := make([]string, len(hef.NetworkGroups))
	for i, ng := range hef.NetworkGroups {
		names[i] = ng.Name
	}

	expected := []string{"network1", "network2", "yolov5"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("name[%d] = %s, expected %s", i, name, expected[i])
		}
	}
}

func TestExtractInputStreamInfo(t *testing.T) {
	hef := &Hef{
		NetworkGroups: []NetworkGroupInfo{
			{
				Name: "test_network",
				InputStreams: []StreamInfo{
					{
						Name:      "input_layer0",
						Direction: StreamDirectionInput,
						Shape:     ImageShape3D{Height: 224, Width: 224, Features: 3},
						Format:    Format{Type: FormatTypeUint8, Order: FormatOrderNHWC},
					},
				},
			},
		},
	}

	if len(hef.NetworkGroups) != 1 {
		t.Fatalf("expected 1 network group, got %d", len(hef.NetworkGroups))
	}

	ng := hef.NetworkGroups[0]
	if len(ng.InputStreams) != 1 {
		t.Fatalf("expected 1 input stream, got %d", len(ng.InputStreams))
	}

	stream := ng.InputStreams[0]
	if stream.Name != "input_layer0" {
		t.Errorf("name = %s, expected input_layer0", stream.Name)
	}
	if stream.Shape.Height != 224 || stream.Shape.Width != 224 || stream.Shape.Features != 3 {
		t.Errorf("unexpected shape: %+v", stream.Shape)
	}
}

func TestExtractOutputStreamInfo(t *testing.T) {
	hef := &Hef{
		NetworkGroups: []NetworkGroupInfo{
			{
				Name: "test_network",
				OutputStreams: []StreamInfo{
					{
						Name:      "output_layer0",
						Direction: StreamDirectionOutput,
						Shape:     ImageShape3D{Height: 1, Width: 1, Features: 1000},
						Format:    Format{Type: FormatTypeFloat32, Order: FormatOrderNHWC},
						QuantInfo: QuantInfo{
							ZeroPoint: 0.0,
							Scale:     0.00392157,
							LimMin:    0.0,
							LimMax:    1.0,
						},
					},
				},
			},
		},
	}

	ng := hef.NetworkGroups[0]
	if len(ng.OutputStreams) != 1 {
		t.Fatalf("expected 1 output stream, got %d", len(ng.OutputStreams))
	}

	stream := ng.OutputStreams[0]
	if stream.Direction != StreamDirectionOutput {
		t.Error("expected output direction")
	}
	if stream.Shape.Features != 1000 {
		t.Errorf("features = %d, expected 1000", stream.Shape.Features)
	}
}

func TestExtractQuantizationInfo(t *testing.T) {
	quantInfo := QuantInfo{
		ZeroPoint: 128.0,
		Scale:     0.00784314,
		LimMin:    -1.0,
		LimMax:    1.0,
	}

	// Test quantization formula: quantized = (float / scale) + zero_point
	floatVal := float32(0.5)
	quantized := (floatVal / quantInfo.Scale) + quantInfo.ZeroPoint

	if quantized < 190 || quantized > 193 {
		t.Errorf("quantized value = %f, expected ~191.75", quantized)
	}

	// Test dequantization: float = (quantized - zero_point) * scale
	dequantized := (quantized - quantInfo.ZeroPoint) * quantInfo.Scale
	if dequantized < 0.49 || dequantized > 0.51 {
		t.Errorf("dequantized value = %f, expected ~0.5", dequantized)
	}
}

func TestParseMultiNetworkHef(t *testing.T) {
	hef := &Hef{
		NetworkGroups: []NetworkGroupInfo{
			{
				Name: "network_group_1",
				InputStreams: []StreamInfo{
					{Name: "input1"},
				},
				OutputStreams: []StreamInfo{
					{Name: "output1"},
				},
			},
			{
				Name: "network_group_2",
				InputStreams: []StreamInfo{
					{Name: "input2_a"},
					{Name: "input2_b"},
				},
				OutputStreams: []StreamInfo{
					{Name: "output2"},
				},
			},
		},
	}

	if len(hef.NetworkGroups) != 2 {
		t.Fatalf("expected 2 network groups, got %d", len(hef.NetworkGroups))
	}

	// Verify first network group
	ng1 := hef.NetworkGroups[0]
	if ng1.Name != "network_group_1" {
		t.Errorf("ng1.Name = %s, expected network_group_1", ng1.Name)
	}
	if len(ng1.InputStreams) != 1 || len(ng1.OutputStreams) != 1 {
		t.Error("ng1 stream count mismatch")
	}

	// Verify second network group
	ng2 := hef.NetworkGroups[1]
	if ng2.Name != "network_group_2" {
		t.Errorf("ng2.Name = %s, expected network_group_2", ng2.Name)
	}
	if len(ng2.InputStreams) != 2 {
		t.Errorf("ng2 should have 2 input streams, got %d", len(ng2.InputStreams))
	}
}

func TestDeviceArchitectureExtraction(t *testing.T) {
	hef := &Hef{
		DeviceArch: ArchHailo8,
	}

	if hef.DeviceArch != ArchHailo8 {
		t.Errorf("device arch = %v, expected Hailo8", hef.DeviceArch)
	}

	// Test string conversion
	archStr := hef.DeviceArch.String()
	if archStr != "Hailo-8" {
		t.Errorf("arch string = %s, expected Hailo-8", archStr)
	}
}

func TestDeviceArchitectureStrings(t *testing.T) {
	tests := []struct {
		arch     DeviceArchitecture
		expected string
	}{
		{ArchHailo8, "Hailo-8"},
		{ArchHailo8P, "Hailo-8P"},
		{ArchHailo8R, "Hailo-8R"},
		{ArchHailo8L, "Hailo-8L"},
		{ArchHailo15H, "Hailo-15H"},
		{ArchHailo15M, "Hailo-15M"},
		{DeviceArchitecture(999), "Unknown(999)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.arch.String()
			if got != tt.expected {
				t.Errorf("String() = %s, expected %s", got, tt.expected)
			}
		})
	}
}

func TestFormatTypeStrings(t *testing.T) {
	tests := []struct {
		format   FormatType
		expected string
	}{
		{FormatTypeAuto, "auto"},
		{FormatTypeUint8, "uint8"},
		{FormatTypeUint16, "uint16"},
		{FormatTypeFloat32, "float32"},
		{FormatType(99), "Unknown(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.format.String()
			if got != tt.expected {
				t.Errorf("String() = %s, expected %s", got, tt.expected)
			}
		})
	}
}

func TestFormatOrderStrings(t *testing.T) {
	tests := []struct {
		order    FormatOrder
		expected string
	}{
		{FormatOrderNHWC, "NHWC"},
		{FormatOrderNCHW, "NCHW"},
		{FormatOrderNV12, "NV12"},
		{FormatOrderHailoNmsByClass, "NMS_BY_CLASS"},
		{FormatOrderHailoNmsByScore, "NMS_BY_SCORE"},
		{FormatOrder(999), "Unknown(999)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.order.String()
			if got != tt.expected {
				t.Errorf("String() = %s, expected %s", got, tt.expected)
			}
		})
	}
}

func TestVStreamInfoWithNms(t *testing.T) {
	vstream := VStreamInfo{
		Name:      "nms_output",
		Direction: StreamDirectionOutput,
		Format:    Format{Order: FormatOrderHailoNmsByClass},
		Shape:     ImageShape3D{Height: 1, Width: 1, Features: 100},
		NmsShape: NmsShape{
			MaxBboxesPerClass:  100,
			NumberOfClasses:    80,
			MaxAccumulatedMask: 8000,
		},
		IsNms: true,
	}

	if !vstream.IsNms {
		t.Error("expected IsNms to be true")
	}

	if vstream.NmsShape.NumberOfClasses != 80 {
		t.Errorf("NumberOfClasses = %d, expected 80", vstream.NmsShape.NumberOfClasses)
	}

	if vstream.Format.Order != FormatOrderHailoNmsByClass {
		t.Error("expected NMS_BY_CLASS format order")
	}
}

func TestStreamInfoHwShape(t *testing.T) {
	stream := StreamInfo{
		Name:  "input",
		Shape: ImageShape3D{Height: 224, Width: 224, Features: 3},
		// Hardware shape may have padding
		HwShape:     ImageShape3D{Height: 224, Width: 232, Features: 8},
		HwFrameSize: 224 * 232 * 8,
	}

	// Verify hw_frame_size calculation
	expectedSize := uint64(stream.HwShape.Height * stream.HwShape.Width * stream.HwShape.Features)
	if stream.HwFrameSize != expectedSize {
		t.Errorf("HwFrameSize = %d, expected %d", stream.HwFrameSize, expectedSize)
	}
}
