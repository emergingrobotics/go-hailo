//go:build unit

package transform

import (
	"math"
	"testing"
)

func TestParseNmsByClass(t *testing.T) {
	data := []float32{
		2.0,
		0.1, 0.2, 0.3, 0.4, 0.9,
		0.5, 0.6, 0.7, 0.8, 0.8,
		1.0,
		0.0, 0.0, 0.5, 0.5, 0.7,
	}

	detections := ParseNmsByClass(data, 2)

	if len(detections) != 3 {
		t.Fatalf("expected 3 detections, got %d", len(detections))
	}

	d0 := detections[0]
	if d0.ClassId != 0 {
		t.Errorf("d0.ClassId = %d, expected 0", d0.ClassId)
	}
	if d0.Score != 0.9 {
		t.Errorf("d0.Score = %f, expected 0.9", d0.Score)
	}
}

func TestParseNmsByScore(t *testing.T) {
	detections := []Detection{
		{Score: 0.5, ClassId: 0},
		{Score: 0.9, ClassId: 1},
		{Score: 0.7, ClassId: 0},
	}

	sorted := SortDetectionsByScore(detections)

	if sorted[0].Score != 0.9 {
		t.Error("first detection should have highest score")
	}
	if sorted[2].Score != 0.5 {
		t.Error("last detection should have lowest score")
	}
}

func TestNmsScoreFiltering(t *testing.T) {
	detections := []Detection{
		{Score: 0.9, ClassId: 0},
		{Score: 0.5, ClassId: 1},
		{Score: 0.3, ClassId: 2},
		{Score: 0.8, ClassId: 0},
	}

	filtered := FilterByScore(detections, 0.6)

	if len(filtered) != 2 {
		t.Fatalf("expected 2 detections after filtering, got %d", len(filtered))
	}

	for _, d := range filtered {
		if d.Score < 0.6 {
			t.Errorf("detection with score %f should have been filtered", d.Score)
		}
	}
}

func TestNmsIouFiltering(t *testing.T) {
	detections := []Detection{
		{
			BBox:    BBox{YMin: 0.0, XMin: 0.0, YMax: 0.5, XMax: 0.5},
			Score:   0.9,
			ClassId: 0,
		},
		{
			BBox:    BBox{YMin: 0.0, XMin: 0.0, YMax: 0.45, XMax: 0.45},
			Score:   0.8,
			ClassId: 0,
		},
		{
			BBox:    BBox{YMin: 0.7, XMin: 0.7, YMax: 1.0, XMax: 1.0},
			Score:   0.7,
			ClassId: 0,
		},
	}

	filtered := ApplyNms(detections, 0.5)

	if len(filtered) != 2 {
		t.Fatalf("expected 2 detections after NMS, got %d", len(filtered))
	}

	if filtered[0].Score != 0.9 {
		t.Error("should keep highest score detection")
	}
}

func TestNmsMaxProposals(t *testing.T) {
	var detections []Detection
	for i := 0; i < 100; i++ {
		detections = append(detections, Detection{
			Score:   float32(100-i) / 100.0,
			ClassId: 0,
		})
	}

	limited := LimitPerClass(detections, 10)

	if len(limited) != 10 {
		t.Errorf("expected 10 detections, got %d", len(limited))
	}

	if limited[0].Score != 1.0 {
		t.Error("should keep highest scoring detections")
	}
}

func TestNmsEmptyOutput(t *testing.T) {
	data := []float32{0.0, 0.0}

	detections := ParseNmsByClass(data, 2)

	if len(detections) != 0 {
		t.Errorf("expected 0 detections, got %d", len(detections))
	}
}

func TestIouCalculation(t *testing.T) {
	tests := []struct {
		name     string
		box1     BBox
		box2     BBox
		expected float32
	}{
		{
			"no overlap",
			BBox{0.0, 0.0, 0.5, 0.5},
			BBox{0.6, 0.6, 1.0, 1.0},
			0.0,
		},
		{
			"full overlap",
			BBox{0.0, 0.0, 1.0, 1.0},
			BBox{0.0, 0.0, 1.0, 1.0},
			1.0,
		},
		{
			"partial overlap",
			BBox{0.0, 0.0, 0.5, 0.5},
			BBox{0.25, 0.25, 0.75, 0.75},
			0.143,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iou := CalculateIou(tt.box1, tt.box2)
			diff := float64(math.Abs(float64(iou - tt.expected)))
			if diff > 0.01 {
				t.Errorf("IOU = %f, expected %f", iou, tt.expected)
			}
		})
	}
}

func TestDetectionBBoxNormalization(t *testing.T) {
	d := Detection{
		BBox:  BBox{YMin: 0.1, XMin: 0.2, YMax: 0.8, XMax: 0.9},
		Score: 0.9,
	}

	if d.BBox.YMin < 0 || d.BBox.YMin > 1 ||
		d.BBox.XMin < 0 || d.BBox.XMin > 1 ||
		d.BBox.YMax < 0 || d.BBox.YMax > 1 ||
		d.BBox.XMax < 0 || d.BBox.XMax > 1 {
		t.Error("BBox coordinates should be normalized")
	}
}

func TestBBoxMethods(t *testing.T) {
	box := BBox{YMin: 0.0, XMin: 0.0, YMax: 0.5, XMax: 0.4}

	if box.Width() != 0.4 {
		t.Errorf("Width() = %f, expected 0.4", box.Width())
	}

	if box.Height() != 0.5 {
		t.Errorf("Height() = %f, expected 0.5", box.Height())
	}

	if box.Area() != 0.2 {
		t.Errorf("Area() = %f, expected 0.2", box.Area())
	}

	cx, cy := box.Center()
	if cx != 0.2 || cy != 0.25 {
		t.Errorf("Center() = (%f, %f), expected (0.2, 0.25)", cx, cy)
	}
}

func BenchmarkParseNmsByClass(b *testing.B) {
	dataSize := 80 * (1 + 100*5)
	data := make([]float32, dataSize)
	idx := 0
	for c := 0; c < 80; c++ {
		data[idx] = 100.0
		idx++
		for d := 0; d < 100; d++ {
			data[idx] = 0.1
			data[idx+1] = 0.1
			data[idx+2] = 0.5
			data[idx+3] = 0.5
			data[idx+4] = 0.9
			idx += 5
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ParseNmsByClass(data, 80)
	}
}
