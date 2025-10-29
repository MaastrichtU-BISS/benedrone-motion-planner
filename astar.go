package main

import (
	"container/heap"
)

// Node represents a node in the A* search for visibility graph
type Node struct {
	NodeID int     // ID of the node in the graph
	G      float64 // Cost from start to this node
	H      float64 // Heuristic cost from this node to end
	F      float64 // Total cost (G + H)
	Parent *Node
	Index  int // Index in the heap
}

// PriorityQueue implements heap.Interface for A* algorithm
type PriorityQueue []*Node

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].F < pq[j].F
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	node := x.(*Node)
	node.Index = n
	*pq = append(*pq, node)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	node := old[n-1]
	old[n-1] = nil
	node.Index = -1
	*pq = old[0 : n-1]
	return node
}

// AStarPathOnGraph computes the shortest path using A* on a visibility graph
func AStarPathOnGraph(graph *Graph, startIdx, endIdx int) ([]Point, bool) {
	if graph == nil || len(graph.Nodes) == 0 {
		return []Point{}, false
	}

	startPoint := graph.Nodes[startIdx]
	endPoint := graph.Nodes[endIdx]

	openSet := &PriorityQueue{}
	heap.Init(openSet)

	startNode := &Node{
		NodeID: startIdx,
		G:      0,
		H:      startPoint.Distance(endPoint),
		F:      startPoint.Distance(endPoint),
	}
	heap.Push(openSet, startNode)

	closedSet := make(map[int]bool)
	openSetMap := make(map[int]*Node)
	openSetMap[startIdx] = startNode

	nodesExplored := 0

	for openSet.Len() > 0 {
		current := heap.Pop(openSet).(*Node)
		delete(openSetMap, current.NodeID)
		nodesExplored++

		// Check if we reached the goal
		if current.NodeID == endIdx {
			// Reconstruct path
			path := []Point{}
			for node := current; node != nil; node = node.Parent {
				path = append([]Point{graph.Nodes[node.NodeID]}, path...)
			}
			return path, true
		}

		closedSet[current.NodeID] = true

		// Explore neighbors
		for _, edge := range graph.Edges[current.NodeID] {
			neighborID := edge.To

			if closedSet[neighborID] {
				continue
			}

			// Calculate costs
			tentativeG := current.G + edge.Cost

			neighbor, exists := openSetMap[neighborID]
			if !exists {
				neighborPoint := graph.Nodes[neighborID]
				neighbor = &Node{
					NodeID: neighborID,
					G:      tentativeG,
					H:      neighborPoint.Distance(endPoint),
					Parent: current,
				}
				neighbor.F = neighbor.G + neighbor.H
				heap.Push(openSet, neighbor)
				openSetMap[neighborID] = neighbor
			} else if tentativeG < neighbor.G {
				// Found a better path to this neighbor
				neighbor.G = tentativeG
				neighbor.F = neighbor.G + neighbor.H
				neighbor.Parent = current
				heap.Fix(openSet, neighbor.Index)
			}
		}
	}

	// No path found
	return []Point{}, false
}
