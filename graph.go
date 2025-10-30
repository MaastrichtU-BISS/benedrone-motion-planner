package main

// Graph represents a graph for pathfinding (used by both visibility graph and PRM)
type Graph struct {
	Nodes map[int]Point
	Edges map[int][]Edge
}

// Edge represents a connection between two nodes with a cost
type Edge struct {
	To   int     // Index of the destination node
	Cost float64 // Distance cost
}
