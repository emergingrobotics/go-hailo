package transform

import "sort"

// BBox represents a bounding box with normalized coordinates [0, 1]
type BBox struct {
	YMin, XMin, YMax, XMax float32
}

// Detection represents a detection result
type Detection struct {
	BBox    BBox
	Score   float32
	ClassId int
	Mask    []byte // Optional, for instance segmentation
}

// Width returns the width of the bounding box
func (b BBox) Width() float32 {
	return b.XMax - b.XMin
}

// Height returns the height of the bounding box
func (b BBox) Height() float32 {
	return b.YMax - b.YMin
}

// Area returns the area of the bounding box
func (b BBox) Area() float32 {
	return b.Width() * b.Height()
}

// Center returns the center point of the bounding box
func (b BBox) Center() (x, y float32) {
	return (b.XMin + b.XMax) / 2, (b.YMin + b.YMax) / 2
}

// ParseNmsByClass parses NMS output in BY_CLASS format
// Format: [class0_count, det0, det1, ..., class1_count, det0, ...]
// Detection: [y_min, x_min, y_max, x_max, score]
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

// ParseNmsByScore parses NMS output in BY_SCORE format
// Format: [total_count, det0, det1, ...]
// Detection: [y_min, x_min, y_max, x_max, score, class_id]
func ParseNmsByScore(data []float32) []Detection {
	if len(data) == 0 {
		return nil
	}

	count := int(data[0])
	var detections []Detection
	idx := 1

	for i := 0; i < count && idx+6 <= len(data); i++ {
		d := Detection{
			BBox: BBox{
				YMin: data[idx],
				XMin: data[idx+1],
				YMax: data[idx+2],
				XMax: data[idx+3],
			},
			Score:   data[idx+4],
			ClassId: int(data[idx+5]),
		}
		detections = append(detections, d)
		idx += 6
	}

	return detections
}

// SortDetectionsByScore sorts detections by score in descending order
func SortDetectionsByScore(detections []Detection) []Detection {
	sorted := make([]Detection, len(detections))
	copy(sorted, detections)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Score > sorted[j].Score
	})
	return sorted
}

// FilterByScore filters detections by minimum score threshold
func FilterByScore(detections []Detection, threshold float32) []Detection {
	var filtered []Detection
	for _, d := range detections {
		if d.Score >= threshold {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

// ApplyNms applies non-maximum suppression
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

// ApplyNmsAllClasses applies non-maximum suppression across all classes
func ApplyNmsAllClasses(detections []Detection, iouThreshold float32) []Detection {
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
			if CalculateIou(d.BBox, sorted[j].BBox) > iouThreshold {
				suppressed[j] = true
			}
		}
	}

	return kept
}

// LimitPerClass limits the number of detections per class
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

// CalculateIou calculates the Intersection over Union of two bounding boxes
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

// ScaleDetections scales detection coordinates from normalized to pixel coordinates
func ScaleDetections(detections []Detection, imgWidth, imgHeight int) []Detection {
	scaled := make([]Detection, len(detections))
	w := float32(imgWidth)
	h := float32(imgHeight)

	for i, d := range detections {
		scaled[i] = Detection{
			BBox: BBox{
				YMin: d.BBox.YMin * h,
				XMin: d.BBox.XMin * w,
				YMax: d.BBox.YMax * h,
				XMax: d.BBox.XMax * w,
			},
			Score:   d.Score,
			ClassId: d.ClassId,
			Mask:    d.Mask,
		}
	}

	return scaled
}
