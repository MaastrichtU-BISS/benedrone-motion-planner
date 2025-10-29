package main

import (
	"log"
	"math"
)

// MergeOverlappingPolygons merges polygons that overlap or are contained within each other
// This reduces the total number of vertices and simplifies the visibility graph
func MergeOverlappingPolygons(polygons []Polygon) []Polygon {
	if len(polygons) <= 1 {
		return polygons
	}

	// First, remove polygons that are fully contained within other polygons
	filtered := removeContainedPolygons(polygons)

	log.Printf("   Polygons after removing contained: %d (removed %d)\n",
		len(filtered), len(polygons)-len(filtered))

	return filtered
}

// removeContainedPolygons removes polygons that are fully contained within other polygons
func removeContainedPolygons(polygons []Polygon) []Polygon {
	if len(polygons) <= 1 {
		return polygons
	}

	result := make([]Polygon, 0, len(polygons))
	contained := make([]bool, len(polygons))

	// Check each polygon against all others
	for i := 0; i < len(polygons); i++ {
		if contained[i] {
			continue
		}

		for j := 0; j < len(polygons); j++ {
			if i == j || contained[j] {
				continue
			}

			// Check if polygon i is contained in polygon j
			if isPolygonContainedIn(polygons[i], polygons[j]) {
				contained[i] = true
				break
			}

			// Check if polygon j is contained in polygon i
			if isPolygonContainedIn(polygons[j], polygons[i]) {
				contained[j] = true
			}
		}
	}

	// Collect non-contained polygons
	for i := 0; i < len(polygons); i++ {
		if !contained[i] {
			result = append(result, polygons[i])
		}
	}

	return result
}

// isPolygonContainedIn checks if polygon A is fully contained within polygon B
func isPolygonContainedIn(a, b Polygon) bool {
	if len(a.Vertices) == 0 || len(b.Vertices) == 0 {
		return false
	}

	// Quick bounding box check first
	if !isBBoxContained(getBBox(a), getBBox(b)) {
		return false
	}

	// Check if all vertices of A are inside B
	for _, vertex := range a.Vertices {
		if !IsPointInPolygon(vertex, b) {
			return false
		}
	}

	return true
}

// BBox represents a bounding box
type BBox struct {
	MinX, MinY, MaxX, MaxY float64
}

// getBBox calculates the bounding box of a polygon
func getBBox(poly Polygon) BBox {
	if len(poly.Vertices) == 0 {
		return BBox{}
	}

	bbox := BBox{
		MinX: poly.Vertices[0].X,
		MinY: poly.Vertices[0].Y,
		MaxX: poly.Vertices[0].X,
		MaxY: poly.Vertices[0].Y,
	}

	for _, v := range poly.Vertices[1:] {
		bbox.MinX = math.Min(bbox.MinX, v.X)
		bbox.MinY = math.Min(bbox.MinY, v.Y)
		bbox.MaxX = math.Max(bbox.MaxX, v.X)
		bbox.MaxY = math.Max(bbox.MaxY, v.Y)
	}

	return bbox
}

// isBBoxContained checks if bounding box A is contained in bounding box B
func isBBoxContained(a, b BBox) bool {
	return a.MinX >= b.MinX && a.MaxX <= b.MaxX &&
		a.MinY >= b.MinY && a.MaxY <= b.MaxY
}

// MergeAdjacentPolygons attempts to merge polygons that share edges
// This is a simplified version - full polygon union is complex
func MergeAdjacentPolygons(polygons []Polygon, tolerance float64) []Polygon {
	if len(polygons) <= 1 {
		return polygons
	}

	// Build groups of polygons that should be merged
	merged := make([]bool, len(polygons))
	result := make([]Polygon, 0, len(polygons))

	for i := 0; i < len(polygons); i++ {
		if merged[i] {
			continue
		}

		// Find all polygons that share edges with polygon i
		group := []int{i}
		merged[i] = true

		for j := i + 1; j < len(polygons); j++ {
			if merged[j] {
				continue
			}

			// Check if polygons share any edges
			if shareEdge(polygons[i], polygons[j], tolerance) {
				group = append(group, j)
				merged[j] = true
			}
		}

		// If only one polygon in group, add it as-is
		if len(group) == 1 {
			result = append(result, polygons[i])
		} else {
			// For now, just use the convex hull of all vertices
			// This is a simplification - proper union is more complex
			allVertices := make([]Point, 0)
			for _, idx := range group {
				allVertices = append(allVertices, polygons[idx].Vertices...)
			}
			hull := convexHull(allVertices)
			result = append(result, Polygon{Vertices: hull})
		}
	}

	return result
}

// shareEdge checks if two polygons share a common edge
func shareEdge(a, b Polygon, tolerance float64) bool {
	// Check each edge of polygon A against each edge of polygon B
	for i := 0; i < len(a.Vertices); i++ {
		v1 := a.Vertices[i]
		v2 := a.Vertices[(i+1)%len(a.Vertices)]

		for j := 0; j < len(b.Vertices); j++ {
			v3 := b.Vertices[j]
			v4 := b.Vertices[(j+1)%len(b.Vertices)]

			// Check if edges are the same (or reversed)
			if (pointsEqual(v1, v3, tolerance) && pointsEqual(v2, v4, tolerance)) ||
				(pointsEqual(v1, v4, tolerance) && pointsEqual(v2, v3, tolerance)) {
				return true
			}
		}
	}
	return false
}

// pointsEqual checks if two points are equal within tolerance
func pointsEqual(a, b Point, tolerance float64) bool {
	return math.Abs(a.X-b.X) <= tolerance && math.Abs(a.Y-b.Y) <= tolerance
}

// convexHull computes the convex hull using Graham scan algorithm
func convexHull(points []Point) []Point {
	if len(points) < 3 {
		return points
	}

	// Find the point with lowest Y (and lowest X if tied)
	start := 0
	for i := 1; i < len(points); i++ {
		if points[i].Y < points[start].Y ||
			(points[i].Y == points[start].Y && points[i].X < points[start].X) {
			start = i
		}
	}

	// Swap start point to position 0
	points[0], points[start] = points[start], points[0]
	pivot := points[0]

	// Sort points by polar angle with respect to pivot
	sortedPoints := make([]Point, len(points)-1)
	copy(sortedPoints, points[1:])

	// Simple bubble sort by angle (good enough for small sets)
	for i := 0; i < len(sortedPoints)-1; i++ {
		for j := i + 1; j < len(sortedPoints); j++ {
			if polarAngle(pivot, sortedPoints[j]) < polarAngle(pivot, sortedPoints[i]) {
				sortedPoints[i], sortedPoints[j] = sortedPoints[j], sortedPoints[i]
			}
		}
	}

	// Build hull
	hull := []Point{pivot, sortedPoints[0]}

	for i := 1; i < len(sortedPoints); i++ {
		// Remove points that create right turn
		for len(hull) > 1 && crossProduct(hull[len(hull)-2], hull[len(hull)-1], sortedPoints[i]) <= 0 {
			hull = hull[:len(hull)-1]
		}
		hull = append(hull, sortedPoints[i])
	}

	return hull
}

// polarAngle calculates the polar angle from pivot to point
func polarAngle(pivot, point Point) float64 {
	return math.Atan2(point.Y-pivot.Y, point.X-pivot.X)
}

// crossProduct calculates the cross product of vectors (b-a) and (c-a)
func crossProduct(a, b, c Point) float64 {
	return (b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X)
}
