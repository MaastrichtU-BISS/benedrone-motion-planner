package main

import (
	"math"
)

// SimplifyPolygon reduces polygon complexity using Douglas-Peucker algorithm
// epsilon is the tolerance - larger values = more simplification
func SimplifyPolygon(polygon Polygon, epsilon float64) Polygon {
	if len(polygon.Vertices) <= 3 {
		return polygon // Already minimal
	}

	simplified := douglasPeucker(polygon.Vertices, epsilon)
	return Polygon{Vertices: simplified}
}

// SimplifyPolygons simplifies multiple polygons
func SimplifyPolygons(polygons []Polygon, epsilon float64) []Polygon {
	simplified := make([]Polygon, len(polygons))
	for i, poly := range polygons {
		simplified[i] = SimplifyPolygon(poly, epsilon)
	}
	return simplified
}

// douglasPeucker implements the Douglas-Peucker line simplification algorithm
func douglasPeucker(points []Point, epsilon float64) []Point {
	if len(points) <= 2 {
		return points
	}

	// Find the point with maximum distance from line between first and last
	dmax := 0.0
	index := 0
	end := len(points) - 1

	for i := 1; i < end; i++ {
		d := perpendicularDistance(points[i], points[0], points[end])
		if d > dmax {
			index = i
			dmax = d
		}
	}

	// If max distance is greater than epsilon, recursively simplify
	if dmax > epsilon {
		// Recursive call on both parts
		left := douglasPeucker(points[0:index+1], epsilon)
		right := douglasPeucker(points[index:], epsilon)

		// Combine results (removing duplicate point at index)
		result := make([]Point, 0, len(left)+len(right)-1)
		result = append(result, left[:len(left)-1]...)
		result = append(result, right...)
		return result
	}

	// All points in between can be discarded
	return []Point{points[0], points[end]}
}

// perpendicularDistance calculates perpendicular distance from point to line
func perpendicularDistance(point, lineStart, lineEnd Point) float64 {
	dx := lineEnd.X - lineStart.X
	dy := lineEnd.Y - lineStart.Y

	// Normalize
	mag := math.Sqrt(dx*dx + dy*dy)
	if mag > 0 {
		dx /= mag
		dy /= mag
	}

	pvx := point.X - lineStart.X
	pvy := point.Y - lineStart.Y

	// Get dot product (project pv onto normalized direction)
	pvdot := dx*pvx + dy*pvy

	// Scale by length to get actual distance
	ax := pvx - pvdot*dx
	ay := pvy - pvdot*dy

	return math.Sqrt(ax*ax + ay*ay)
}

// EstimateSimplificationEpsilon suggests epsilon based on coordinate system and vertex count
func EstimateSimplificationEpsilon(polygons []Polygon, currentVertexCount int) float64 {
	if len(polygons) == 0 {
		return 0.0001 // Default for lat/lng
	}

	// Sample some points to detect coordinate system
	samplePoint := polygons[0].Vertices[0]

	// Check if lat/lng (values between -180 and 180)
	if samplePoint.X >= -180 && samplePoint.X <= 180 &&
		samplePoint.Y >= -90 && samplePoint.Y <= 90 {
		// Lat/lng coordinates: use adaptive epsilon based on vertex count
		// Base epsilon: 0.0001 degrees ≈ 11 meters
		// Target: reduce to max 400-500 vertices for reasonable performance
		baseEpsilon := 0.0001

		if currentVertexCount > 30000 {
			return baseEpsilon * 20.0 // 0.002 degrees ≈ 220 meters
		} else if currentVertexCount > 20000 {
			return baseEpsilon * 15.0 // 0.0015 degrees ≈ 165 meters
		} else if currentVertexCount > 10000 {
			return baseEpsilon * 10.0 // 0.001 degrees ≈ 110 meters
		} else if currentVertexCount > 5000 {
			return baseEpsilon * 5.0 // 0.0005 degrees ≈ 55 meters
		} else if currentVertexCount > 2000 {
			return baseEpsilon * 3.0 // 0.0003 degrees ≈ 33 meters
		} else if currentVertexCount > 1000 {
			return baseEpsilon * 2.0 // 0.0002 degrees ≈ 22 meters
		}
		return baseEpsilon
	}

	// Projected/planar coordinates: use larger epsilon
	if currentVertexCount > 30000 {
		return 20.0
	} else if currentVertexCount > 20000 {
		return 15.0
	} else if currentVertexCount > 10000 {
		return 10.0
	} else if currentVertexCount > 5000 {
		return 5.0
	} else if currentVertexCount > 2000 {
		return 3.0
	} else if currentVertexCount > 1000 {
		return 2.0
	}
	return 1.0
}
