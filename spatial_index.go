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
func GetRouteBoundingBox(start, end Point, margin float64) (minX, minY, maxX, maxY float64) {
	minX = min(start.X, end.X) - margin
	maxX = max(start.X, end.X) + margin
	minY = min(start.Y, end.Y) - margin
	maxY = max(start.Y, end.Y) + margin
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
