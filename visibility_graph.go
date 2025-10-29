package main

import "log"

// Graph represents a visibility graph for pathfinding
type Graph struct {
	Nodes map[int]Point
	Edges map[int][]Edge
}

// Edge represents a connection between two nodes with a cost
type Edge struct {
	To   int     // Index of the destination node
	Cost float64 // Euclidean distance
}

// BuildVisibilityGraph constructs a visibility graph from start, end, and no-fly zone polygons
func BuildVisibilityGraph(start, end Point, noFlyZones []Polygon) *Graph {
	graph := &Graph{
		Nodes: make(map[int]Point),
		Edges: make(map[int][]Edge),
	}

	nodeIndex := 0

	// Add start point
	startIdx := nodeIndex
	graph.Nodes[nodeIndex] = start
	nodeIndex++

	// Add end point
	endIdx := nodeIndex
	graph.Nodes[nodeIndex] = end
	nodeIndex++

	// Track vertex to node index mapping
	vertexToIdx := make(map[Point]int)
	vertexToIdx[start] = startIdx
	vertexToIdx[end] = endIdx

	// Count total vertices
	totalVertices := 0
	for _, zone := range noFlyZones {
		totalVertices += len(zone.Vertices)
	}
	log.Printf("   Total vertices in polygons: %d\n", totalVertices)

	// Add all polygon vertices as nodes
	for _, zone := range noFlyZones {
		for _, vertex := range zone.Vertices {
			// Skip if vertex is already added (e.g., shared vertices)
			if _, exists := vertexToIdx[vertex]; !exists {
				graph.Nodes[nodeIndex] = vertex
				vertexToIdx[vertex] = nodeIndex
				nodeIndex++
			}
		}
	}

	totalNodes := len(graph.Nodes)
	totalPossibleEdges := (totalNodes * (totalNodes - 1)) / 2
	log.Printf("   Unique nodes: %d\n", totalNodes)
	log.Printf("   Checking up to %d possible edges...\n", totalPossibleEdges)

	// Safety limit: prevent massive graphs
	if totalNodes > 1000 {
		log.Printf("❌ ERROR: Too many nodes (%d). Reduce margin or simplify polygons.\n", totalNodes)
		// Return minimal graph with just start and end
		graph.Edges[startIdx] = append(graph.Edges[startIdx], Edge{To: endIdx, Cost: start.Distance(end)})
		graph.Edges[endIdx] = append(graph.Edges[endIdx], Edge{To: startIdx, Cost: start.Distance(end)})
		return graph
	}

	if totalPossibleEdges > 100000 {
		log.Printf("⚠️  WARNING: %d edge checks may take 30+ seconds!\n", totalPossibleEdges)
	}

	// Build edges: connect nodes that have line-of-sight (no collision)
	edgesChecked := 0
	edgesAdded := 0

	for i, nodeI := range graph.Nodes {
		for j, nodeJ := range graph.Nodes {
			if i >= j {
				continue // Avoid duplicates and self-loops
			}

			edgesChecked++

			// Log progress for large graphs
			if edgesChecked%10000 == 0 {
				log.Printf("   Progress: %d/%d edges checked...\n", edgesChecked, totalPossibleEdges)
			}

			// Check if there's a clear path between the two nodes
			if IsPathClear(nodeI, nodeJ, noFlyZones) {
				distance := nodeI.Distance(nodeJ)

				// Add bidirectional edge
				graph.Edges[i] = append(graph.Edges[i], Edge{To: j, Cost: distance})
				graph.Edges[j] = append(graph.Edges[j], Edge{To: i, Cost: distance})
				edgesAdded++
			}
		}
	}

	log.Printf("   Edges added: %d\n", edgesAdded)

	return graph
}
