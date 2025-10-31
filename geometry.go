package main

import "math"

// Polygon represents a no-fly zone as a list of vertices
type Polygon struct {
	Vertices []Point `json:"vertices"`
}

// Distance calculates Euclidean distance between two points
func (p Point) Distance(other Point) float64 {
	dx := p.X - other.X
	dy := p.Y - other.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// DistanceMeters calculates the distance in meters between two points in lat/lng coordinates
// Uses the Haversine formula for accurate distance calculation
func (p Point) DistanceMeters(other Point) float64 {
	const earthRadiusMeters = 6371000.0 // Earth's radius in meters

	// Convert degrees to radians
	lat1 := p.Y * math.Pi / 180.0
	lat2 := other.Y * math.Pi / 180.0
	deltaLat := (other.Y - p.Y) * math.Pi / 180.0
	deltaLon := (other.X - p.X) * math.Pi / 180.0

	// Haversine formula
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMeters * c
}

// LineSegment represents a line segment between two points
type LineSegment struct {
	P1, P2 Point
}

// DoSegmentsIntersect checks if two line segments intersect
func DoSegmentsIntersect(seg1, seg2 LineSegment) bool {
	p1, p2 := seg1.P1, seg1.P2
	p3, p4 := seg2.P1, seg2.P2

	// Check if the segments are the same or share endpoints
	if (p1 == p3 && p2 == p4) || (p1 == p4 && p2 == p3) {
		return false
	}
	if p1 == p3 || p1 == p4 || p2 == p3 || p2 == p4 {
		return false
	}

	d1 := direction(p3, p4, p1)
	d2 := direction(p3, p4, p2)
	d3 := direction(p1, p2, p3)
	d4 := direction(p1, p2, p4)

	if ((d1 > 0 && d2 < 0) || (d1 < 0 && d2 > 0)) &&
		((d3 > 0 && d4 < 0) || (d3 < 0 && d4 > 0)) {
		return true
	}

	// Check for collinear cases
	if d1 == 0 && onSegment(p3, p4, p1) {
		return true
	}
	if d2 == 0 && onSegment(p3, p4, p2) {
		return true
	}
	if d3 == 0 && onSegment(p1, p2, p3) {
		return true
	}
	if d4 == 0 && onSegment(p1, p2, p4) {
		return true
	}

	return false
}

// direction calculates the cross product to determine orientation
func direction(p1, p2, p3 Point) float64 {
	return (p3.X-p1.X)*(p2.Y-p1.Y) - (p2.X-p1.X)*(p3.Y-p1.Y)
}

// onSegment checks if point q lies on segment pr
func onSegment(p, r, q Point) bool {
	return q.X <= math.Max(p.X, r.X) && q.X >= math.Min(p.X, r.X) &&
		q.Y <= math.Max(p.Y, r.Y) && q.Y >= math.Min(p.Y, r.Y)
}

// IsPointInPolygon checks if a point is inside a polygon using ray casting
func IsPointInPolygon(point Point, polygon Polygon) bool {
	n := len(polygon.Vertices)
	if n < 3 {
		return false
	}

	count := 0
	for i := 0; i < n; i++ {
		v1 := polygon.Vertices[i]
		v2 := polygon.Vertices[(i+1)%n]

		// Check if the ray from point to the right intersects the edge
		if (v1.Y > point.Y) != (v2.Y > point.Y) {
			slope := (point.X-v1.X)*(v2.Y-v1.Y) - (v2.X-v1.X)*(point.Y-v1.Y)
			if v2.Y > v1.Y {
				if slope > 0 {
					count++
				}
			} else {
				if slope < 0 {
					count++
				}
			}
		}
	}

	return count%2 == 1
}

// DoesSegmentIntersectPolygon checks if a line segment intersects any edge of a polygon
func DoesSegmentIntersectPolygon(seg LineSegment, polygon Polygon) bool {
	n := len(polygon.Vertices)
	for i := 0; i < n; i++ {
		edge := LineSegment{
			P1: polygon.Vertices[i],
			P2: polygon.Vertices[(i+1)%n],
		}
		if DoSegmentsIntersect(seg, edge) {
			return true
		}
	}
	return false
}

// IsPathClear checks if a straight line path between two points is collision-free
func IsPathClear(p1, p2 Point, noFlyZones []Polygon) bool {
	segment := LineSegment{P1: p1, P2: p2}

	for _, zone := range noFlyZones {
		// Check if the segment intersects the polygon boundary
		if DoesSegmentIntersectPolygon(segment, zone) {
			return false
		}

		// Check if either endpoint is inside the polygon
		if IsPointInPolygon(p1, zone) || IsPointInPolygon(p2, zone) {
			return false
		}

		// Check if the midpoint is inside (handles case where segment is entirely inside)
		midpoint := Point{
			X: (p1.X + p2.X) / 2,
			Y: (p1.Y + p2.Y) / 2,
		}
		if IsPointInPolygon(midpoint, zone) {
			return false
		}
	}

	return true
}
