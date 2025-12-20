//go:build unit

package transform

import (
	"math"
	"sort"
	"testing"
)

func TestParseNmsByClass(t *testing.T) {
	// Create mock NMS output data
	// Format: [class0_count, det0, det1, ..., class1_count, det0, ...]
	// Detection: [y_min, x_min, y_max, x_max, score]

	data := []float32{
		// Class 0: 2 detections
		2.0,
		0.1, 0.2, 0.3, 0.4, 0.9, // Detection 1
		0.5, 0.6, 0.7, 0.8, 0.8, // Detection 2
		// Class 1: 1 detection
		1.0,
		0.0, 0.0, 0.5, 0.5, 0.7, // Detection 1
	}

	detections := ParseNmsByClass(data, 2) // 2 classes

	if len(detections) != 3 {
		t.Fatalf("expected 3 detections, got %d", len(detections))
	}

	// Verify first detection
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
			// Overlaps significantly with first (IOU > 0.5)
			BBox:    BBox{YMin: 0.0, XMin: 0.0, YMax: 0.45, XMax: 0.45},
			Score:   0.8,
			ClassId: 0,
		},
		{
			// Doesn't overlap
			BBox:    BBox{YMin: 0.7, XMin: 0.7, YMax: 1.0, XMax: 1.0},
			Score:   0.7,
			ClassId: 0,
		},
	}

	filtered := ApplyNms(detections, 0.5) // 50% IOU threshold

	if len(filtered) != 2 {
		t.Fatalf("expected 2 detections after NMS, got %d", len(filtered))
	}

	// Should keep highest scoring non-overlapping
	if filtered[0].Score != 0.9 {
		t.Error("should keep highest score detection")
	}
}

func TestNmsMaxProposals(t *testing.T) {
	// Create many detections
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

	// Should keep highest scoring
	if limited[0].Score != 1.0 {
		t.Error("should keep highest scoring detections")
	}
}

func TestNmsEmptyOutput(t *testing.T) {
	data := []float32{0.0, 0.0} // No detections for 2 classes

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
			// Intersection = 0.25 * 0.25 = 0.0625
			// Union = 0.25 + 0.25 - 0.0625 = 0.4375
			// IOU = 0.0625 / 0.4375 = 0.143
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
	// BBox coordinates should be normalized [0, 1]
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

func BenchmarkParseNmsByClass(b *testing.B) {
	// Create mock data with 80 classes, 100 detections each
	dataSize := 80 * (1 + 100*5) // count + detections per class
	data := make([]float32, dataSize)
	idx := 0
	for c := 0; c < 80; c++ {
		data[idx] = 100.0 // 100 detections per class
		idx++
		for d := 0; d < 100; d++ {
			data[idx] = 0.1   // y_min
			data[idx+1] = 0.1 // x_min
			data[idx+2] = 0.5 // y_max
			data[idx+3] = 0.5 // x_max
			data[idx+4] = 0.9 // score
			idx += 5
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ParseNmsByClass(data, 80)
	}
}

// Types

type BBox struct {
	YMin, XMin, YMax, XMax float32
}

type Detection struct {
	BBox    BBox
	Score   float32
	ClassId int
}

// Functions

func ParseNmsByClass(data []float32, numClasses int) []Detection {
	var detections []Detection
	idx := 0

	for classId := 0; classId < numClasses; classId++ {
		if idx >= len(data) {
			break
		}

		count := int(data[idx])
		idx++

		for i := 0; i < count && idx+5 <= len(data); i++ {
			d := Detection{
				BBox: BBox{
					YMin: data[idx],
					XMin: data[idx+1],
					YMax: data[idx+2],
					XMax: data[idx+3],
				},
				Score:   data[idx+4],
				ClassId: classId,
			}
			detections = append(detections, d)
			idx += 5
		}
	}

	return detections
}

func SortDetectionsByScore(detections []Detection) []Detection {
	sorted := make([]Detection, len(detections))
	copy(sorted, detections)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Score > sorted[j].Score
	})
	return sorted
}

func FilterByScore(detections []Detection, threshold float32) []Detection {
	var filtered []Detection
	for _, d := range detections {
		if d.Score >= threshold {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

func ApplyNms(detections []Detection, iouThreshold float32) []Detection {
	sorted := SortDetectionsByScore(detections)
	var kept []Detection
	suppressed := make([]bool, len(sorted))

	for i, d := range sorted {
		if suppressed[i] {
			continue
		}
		kept = append(kept, d)

		for j := i + 1; j < len(sorted); j++ {
			if suppressed[j] {
				continue
			}
			if sorted[j].ClassId != d.ClassId {
				continue
			}
			if CalculateIou(d.BBox, sorted[j].BBox) > iouThreshold {
				suppressed[j] = true
			}
		}
	}

	return kept
}

func LimitPerClass(detections []Detection, maxPerClass int) []Detection {
	sorted := SortDetectionsByScore(detections)
	classCount := make(map[int]int)
	var limited []Detection

	for _, d := range sorted {
		if classCount[d.ClassId] < maxPerClass {
			limited = append(limited, d)
			classCount[d.ClassId]++
		}
	}

	return limited
}

func CalculateIou(b1, b2 BBox) float32 {
	// Calculate intersection
	yMin := max32(b1.YMin, b2.YMin)
	xMin := max32(b1.XMin, b2.XMin)
	yMax := min32(b1.YMax, b2.YMax)
	xMax := min32(b1.XMax, b2.XMax)

	if yMax <= yMin || xMax <= xMin {
		return 0.0
	}

	intersection := (yMax - yMin) * (xMax - xMin)

	// Calculate union
	area1 := (b1.YMax - b1.YMin) * (b1.XMax - b1.XMin)
	area2 := (b2.YMax - b2.YMin) * (b2.XMax - b2.XMin)
	union := area1 + area2 - intersection

	if union <= 0 {
		return 0.0
	}

	return intersection / union
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
