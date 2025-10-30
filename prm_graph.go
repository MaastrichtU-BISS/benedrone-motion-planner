package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"time"
)

// PRMNode represents a node in the probabilistic roadmap
type PRMNode struct {
	ID    int   `json:"id"`
	Point Point `json:"point"`
	Edges []int `json:"edges"` // IDs of connected nodes
}

// PRMGraph represents a pre-computed probabilistic roadmap
type PRMGraph struct {
	Nodes       []PRMNode `json:"nodes"`
	BoundingBox struct {
		MinLat float64 `json:"minLat"`
		MaxLat float64 `json:"maxLat"`
		MinLon float64 `json:"minLon"`
		MaxLon float64 `json:"maxLon"`
	} `json:"boundingBox"`
	NumSamples       int     `json:"numSamples"`
	ConnectionRadius float64 `json:"connectionRadius"` // in degrees
}

// Netherlands bounding box (approximate)
const (
	NetherlandsMinLat = 50.75 // South (Limburg)
	NetherlandsMaxLat = 53.55 // North (Groningen)
	NetherlandsMinLon = 3.36  // West (North Sea coast)
	NetherlandsMaxLon = 7.23  // East (German border)
)

// BuildPRMGraph creates a probabilistic roadmap with random sampling
// Excludes edges that intersect with no-fly zone polygons
func BuildPRMGraph(numSamples int, connectionRadius float64, noFlyZones []Polygon) *PRMGraph {
	startTime := time.Now()
	log.Printf("🗺️  Building PRM graph with %d samples...\n", numSamples)
	log.Printf("   No-fly zones: %d polygons\n", len(noFlyZones))

	graph := &PRMGraph{
		Nodes:            make([]PRMNode, 0, numSamples),
		NumSamples:       numSamples,
		ConnectionRadius: connectionRadius,
	}

	// Set bounding box to Netherlands
	graph.BoundingBox.MinLat = NetherlandsMinLat
	graph.BoundingBox.MaxLat = NetherlandsMaxLat
	graph.BoundingBox.MinLon = NetherlandsMinLon
	graph.BoundingBox.MaxLon = NetherlandsMaxLon

	// Initialize random number generator
	rand.Seed(time.Now().UnixNano())

	// Step 1: Random sampling within bounding box (filter out points inside no-fly zones)
	log.Println("   Generating random samples...")
	validSamples := 0
	attempts := 0
	maxAttempts := numSamples * 10 // Try up to 10x the desired samples

	for validSamples < numSamples && attempts < maxAttempts {
		attempts++
		lat := NetherlandsMinLat + rand.Float64()*(NetherlandsMaxLat-NetherlandsMinLat)
		lon := NetherlandsMinLon + rand.Float64()*(NetherlandsMaxLon-NetherlandsMinLon)
		point := Point{X: lon, Y: lat}

		// Check if point is inside any no-fly zone
		insideNoFlyZone := false
		for _, polygon := range noFlyZones {
			if IsPointInPolygon(point, polygon) {
				insideNoFlyZone = true
				break
			}
		}

		if !insideNoFlyZone {
			node := PRMNode{
				ID:    validSamples,
				Point: point,
				Edges: make([]int, 0),
			}
			graph.Nodes = append(graph.Nodes, node)
			validSamples++
		}
	}

	if validSamples < numSamples {
		log.Printf("   ⚠️  Only generated %d valid samples (requested %d)\n", validSamples, numSamples)
	}

	// Step 2: Connect nearby nodes (only if edge doesn't intersect no-fly zones)
	log.Printf("   Connecting nodes (radius: %.4f degrees ≈ %.0f meters)...\n",
		connectionRadius, connectionRadius*111000)

	edgeCount := 0
	rejectedEdges := 0

	for i := 0; i < len(graph.Nodes); i++ {
		for j := i + 1; j < len(graph.Nodes); j++ {
			dist := distance(graph.Nodes[i].Point, graph.Nodes[j].Point)

			if dist <= connectionRadius {
				// Check if edge intersects any no-fly zone
				edgeClear := true
				for _, polygon := range noFlyZones {
					if DoesEdgeIntersectPolygon(graph.Nodes[i].Point, graph.Nodes[j].Point, polygon) {
						edgeClear = false
						rejectedEdges++
						break
					}
				}

				if edgeClear {
					// Add bidirectional edge
					graph.Nodes[i].Edges = append(graph.Nodes[i].Edges, j)
					graph.Nodes[j].Edges = append(graph.Nodes[j].Edges, i)
					edgeCount++
				}
			}
		}
	}

	elapsed := time.Since(startTime)
	log.Printf("   ✅ PRM graph built: %d nodes, %d edges\n", len(graph.Nodes), edgeCount)
	if rejectedEdges > 0 {
		log.Printf("   ℹ️  Rejected %d edges due to no-fly zone intersections\n", rejectedEdges)
	}
	log.Printf("   ⏱️  Build time: %.2f seconds\n", elapsed.Seconds())

	return graph
}

// distance calculates Euclidean distance in degrees (simple for connection check)
func distance(p1, p2 Point) float64 {
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// DoesEdgeIntersectPolygon checks if an edge between two points intersects a polygon
func DoesEdgeIntersectPolygon(p1, p2 Point, polygon Polygon) bool {
	seg := LineSegment{P1: p1, P2: p2}
	return DoesSegmentIntersectPolygon(seg, polygon)
}

// CreateGraphWithStartEnd creates a temporary graph with start and end points connected
// Returns the modified graph and the node IDs for start and end points
func (g *PRMGraph) CreateGraphWithStartEnd(start, end Point, noFlyZones []Polygon) (*PRMGraph, int, int) {
	// Create a copy of the graph with additional nodes for start and end
	tempGraph := &PRMGraph{
		BoundingBox:      g.BoundingBox,
		NumSamples:       g.NumSamples,
		ConnectionRadius: g.ConnectionRadius,
		Nodes:            make([]PRMNode, len(g.Nodes)+2), // +2 for start and end
	}

	// Copy all existing nodes
	copy(tempGraph.Nodes, g.Nodes)

	// Add start point as a new node
	startNodeID := len(g.Nodes)
	tempGraph.Nodes[startNodeID] = PRMNode{
		ID:    startNodeID,
		Point: start,
		Edges: make([]int, 0),
	}

	// Add end point as a new node
	endNodeID := len(g.Nodes) + 1
	tempGraph.Nodes[endNodeID] = PRMNode{
		ID:    endNodeID,
		Point: end,
		Edges: make([]int, 0),
	}

	// Connect start point to nearby nodes within connection radius
	startConnected := false
	for i := 0; i < len(g.Nodes); i++ {
		dist := distance(start, g.Nodes[i].Point)
		if dist <= g.ConnectionRadius {
			// Check if edge intersects any no-fly zone
			edgeClear := true
			for _, polygon := range noFlyZones {
				if DoesEdgeIntersectPolygon(start, g.Nodes[i].Point, polygon) {
					edgeClear = false
					break
				}
			}

			if edgeClear {
				// Add bidirectional edge
				tempGraph.Nodes[startNodeID].Edges = append(tempGraph.Nodes[startNodeID].Edges, i)
				tempGraph.Nodes[i].Edges = append(tempGraph.Nodes[i].Edges, startNodeID)
				startConnected = true
			}
		}
	}

	// Connect end point to nearby nodes within connection radius
	endConnected := false
	for i := 0; i < len(g.Nodes); i++ {
		dist := distance(end, g.Nodes[i].Point)
		if dist <= g.ConnectionRadius {
			// Check if edge intersects any no-fly zone
			edgeClear := true
			for _, polygon := range noFlyZones {
				if DoesEdgeIntersectPolygon(end, g.Nodes[i].Point, polygon) {
					edgeClear = false
					break
				}
			}

			if edgeClear {
				// Add bidirectional edge
				tempGraph.Nodes[endNodeID].Edges = append(tempGraph.Nodes[endNodeID].Edges, i)
				tempGraph.Nodes[i].Edges = append(tempGraph.Nodes[i].Edges, endNodeID)
				endConnected = true
			}
		}
	}

	// Return -1 for node IDs if connection failed
	if !startConnected {
		log.Println("   ⚠️  Could not connect start point to any graph node")
		return tempGraph, -1, endNodeID
	}
	if !endConnected {
		log.Println("   ⚠️  Could not connect end point to any graph node")
		return tempGraph, startNodeID, -1
	}

	return tempGraph, startNodeID, endNodeID
}

// SavePRMGraph serializes and saves the graph to a JSON file
func SavePRMGraph(graph *PRMGraph, filename string) error {
	log.Printf("💾 Saving PRM graph to %s...\n", filename)

	data, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal graph: %w", err)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	log.Printf("   ✅ Graph saved (%d bytes)\n", len(data))
	return nil
}

// LoadPRMGraph deserializes and loads the graph from a JSON file
func LoadPRMGraph(filename string) (*PRMGraph, error) {
	log.Printf("📂 Loading PRM graph from %s...\n", filename)

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var graph PRMGraph
	err = json.Unmarshal(data, &graph)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal graph: %w", err)
	}

	log.Printf("   ✅ Graph loaded: %d nodes\n", len(graph.Nodes))
	return &graph, nil
}

// GetGraphAsLineStrings returns the graph edges as line segments for visualization
func (g *PRMGraph) GetGraphAsLineStrings() [][]Point {
	lines := make([][]Point, 0)

	// Use a map to avoid duplicate edges (since edges are bidirectional)
	seen := make(map[string]bool)

	for _, node := range g.Nodes {
		for _, neighborID := range node.Edges {
			// Create a unique key for this edge (sorted IDs)
			var key string
			if node.ID < neighborID {
				key = fmt.Sprintf("%d-%d", node.ID, neighborID)
			} else {
				key = fmt.Sprintf("%d-%d", neighborID, node.ID)
			}

			if !seen[key] {
				seen[key] = true
				neighbor := g.Nodes[neighborID]
				lines = append(lines, []Point{node.Point, neighbor.Point})
			}
		}
	}

	return lines
}

// FindNearestNode finds the closest node to a given point
func (g *PRMGraph) FindNearestNode(point Point) (int, float64) {
	if len(g.Nodes) == 0 {
		return -1, math.MaxFloat64
	}

	nearestID := 0
	minDist := point.Distance(g.Nodes[0].Point)

	for i := 1; i < len(g.Nodes); i++ {
		dist := point.Distance(g.Nodes[i].Point)
		if dist < minDist {
			minDist = dist
			nearestID = i
		}
	}

	return nearestID, minDist
}

// ConvertToGraph converts PRM graph to the existing Graph structure for A*
func (g *PRMGraph) ConvertToGraph() *Graph {
	graph := &Graph{
		Nodes: make(map[int]Point),
		Edges: make(map[int][]Edge),
	}

	// Add all nodes
	for _, node := range g.Nodes {
		graph.Nodes[node.ID] = node.Point
	}

	// Add all edges
	for _, node := range g.Nodes {
		edges := make([]Edge, 0, len(node.Edges))
		for _, neighborID := range node.Edges {
			neighbor := g.Nodes[neighborID]
			cost := node.Point.Distance(neighbor.Point)
			edges = append(edges, Edge{
				To:   neighborID,
				Cost: cost,
			})
		}
		graph.Edges[node.ID] = edges
	}

	return graph
}
