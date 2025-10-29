package main

import (
	"github.com/dhconnelly/rtreego"
)

// PolygonEntry wraps a polygon for R-tree storage
type PolygonEntry struct {
	Polygon Polygon
	BBox    rtreego.Rect
}

// Bounds implements rtreego.Spatial interface
func (p *PolygonEntry) Bounds() rtreego.Rect {
	return p.BBox
}

// SpatialIndex manages polygon spatial queries
type SpatialIndex struct {
	tree *rtreego.Rtree
}

// NewSpatialIndex creates a new spatial index
func NewSpatialIndex(polygons []Polygon) *SpatialIndex {
	tree := rtreego.NewTree(2, 25, 50) // 2D, min 25, max 50 entries per node

	for _, polygon := range polygons {
		bbox, err := calculateBoundingBox(polygon)
		if err == nil {
			entry := &PolygonEntry{
				Polygon: polygon,
				BBox:    bbox,
			}
			tree.Insert(entry)
		}
	}

	return &SpatialIndex{tree: tree}
}

// QueryRegion returns polygons that intersect with the given bounding box
func (si *SpatialIndex) QueryRegion(minX, minY, maxX, maxY float64) []Polygon {
	bbox, err := rtreego.NewRect(
		rtreego.Point{minX, minY},
		[]float64{maxX - minX, maxY - minY},
	)
	if err != nil {
		return []Polygon{}
	}

	results := si.tree.SearchIntersect(bbox)
	polygons := make([]Polygon, 0, len(results))

	for _, item := range results {
		entry := item.(*PolygonEntry)
		polygons = append(polygons, entry.Polygon)
	}

	return polygons
}

// calculateBoundingBox computes the axis-aligned bounding box for a polygon
func calculateBoundingBox(polygon Polygon) (rtreego.Rect, error) {
	if len(polygon.Vertices) == 0 {
		return rtreego.Rect{}, nil
	}

	minX, minY := polygon.Vertices[0].X, polygon.Vertices[0].Y
	maxX, maxY := polygon.Vertices[0].X, polygon.Vertices[0].Y

	for _, v := range polygon.Vertices[1:] {
		if v.X < minX {
			minX = v.X
		}
		if v.X > maxX {
			maxX = v.X
		}
		if v.Y < minY {
			minY = v.Y
		}
		if v.Y > maxY {
			maxY = v.Y
		}
	}

	return rtreego.NewRect(
		rtreego.Point{minX, minY},
		[]float64{maxX - minX, maxY - minY},
	)
}

// GetRouteBoundingBox calculates the bounding box for a route with margin
// Uses default expansion factor of 1.0 (no expansion)
func GetRouteBoundingBox(start, end Point, margin float64) (minX, minY, maxX, maxY float64) {
	return GetRouteBoundingBoxWithFactor(start, end, margin, 1.0)
}

// GetRouteBoundingBoxWithFactor calculates the bounding box with a custom expansion factor
func GetRouteBoundingBoxWithFactor(start, end Point, margin float64, expansionFactor float64) (minX, minY, maxX, maxY float64) {
	baseMinX := min(start.X, end.X)
	baseMaxX := max(start.X, end.X)
	baseMinY := min(start.Y, end.Y)
	baseMaxY := max(start.Y, end.Y)

	// Calculate the route dimensions
	width := baseMaxX - baseMinX
	height := baseMaxY - baseMinY

	// Combine both additive margin and multiplicative expansion
	extraX := max(margin, width*expansionFactor*0.5)
	extraY := max(margin, height*expansionFactor*0.5)

	minX = baseMinX - extraX
	maxX = baseMaxX + extraX
	minY = baseMinY - extraY
	maxY = baseMaxY + extraY

	return
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
