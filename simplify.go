package main

import (
	"math"
)

// SimplifyPolygon reduces polygon complexity using Douglas-Peucker algorithm
// For closed polygons, uses topology-preserving approach to avoid expansion
func SimplifyPolygon(polygon Polygon, epsilon float64) Polygon {
	if len(polygon.Vertices) <= 3 {
		return polygon
	}

	n := len(polygon.Vertices)
	first := polygon.Vertices[0]
	last := polygon.Vertices[n-1]
	const closeThreshold = 1e-9
	isClosed := (math.Abs(first.X-last.X) < closeThreshold && math.Abs(first.Y-last.Y) < closeThreshold)

	var simplified []Point
	if isClosed {
		// Remove duplicate closing point, simplify, then re-close
		openPolygon := polygon.Vertices[:n-1]
		simplified = douglasPeucker(append(openPolygon, openPolygon[0]), epsilon)

		// Remove the duplicate we added and re-close properly
		if len(simplified) > 1 {
			simplified = simplified[:len(simplified)-1]
			if len(simplified) >= 3 {
				simplified = append(simplified, simplified[0])
			} else {
				return polygon // Failed to simplify adequately
			}
		}
	} else {
		simplified = douglasPeucker(polygon.Vertices, epsilon)
	}

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
// Uses conservative values to preserve polygon topology
func EstimateSimplificationEpsilon(polygons []Polygon, currentVertexCount int) float64 {
	if len(polygons) == 0 {
		return 0.0001
	}

	samplePoint := polygons[0].Vertices[0]

	// Check if lat/lng coordinates
	if samplePoint.X >= -180 && samplePoint.X <= 180 &&
		samplePoint.Y >= -90 && samplePoint.Y <= 90 {
		// Base: 0.00002 degrees ≈ 2.2 meters
		baseEpsilon := 0.00002

		if currentVertexCount > 50000 {
			return baseEpsilon * 10.0
		} else if currentVertexCount > 30000 {
			return baseEpsilon * 7.0
		} else if currentVertexCount > 20000 {
			return baseEpsilon * 5.0
		} else if currentVertexCount > 10000 {
			return baseEpsilon * 4.0
		} else if currentVertexCount > 5000 {
			return baseEpsilon * 3.0
		} else if currentVertexCount > 2000 {
			return baseEpsilon * 2.0
		} else if currentVertexCount > 1000 {
			return baseEpsilon * 1.5
		}
		return baseEpsilon
	}

	// Projected/planar coordinates
	if currentVertexCount > 50000 {
		return 20.0
	} else if currentVertexCount > 30000 {
		return 15.0
	} else if currentVertexCount > 20000 {
		return 10.0
	} else if currentVertexCount > 10000 {
		return 7.0
	} else if currentVertexCount > 5000 {
		return 5.0
	} else if currentVertexCount > 2000 {
		return 3.0
	} else if currentVertexCount > 1000 {
		return 2.0
	}
	return 1.0
}
