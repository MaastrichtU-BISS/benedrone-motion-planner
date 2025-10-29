package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type RouteRequest struct {
	Start Point `json:"start"`
	End   Point `json:"end"`
	// Optional: margin in map units to query around the route (default: 1000)
	Margin float64 `json:"margin,omitempty"`
}

type CreateIndexRequest struct {
	Polygons []Polygon `json:"polygons"`
	Force    bool      `json:"force,omitempty"` // Set to true to force reload
}

type RouteResponse struct {
	Path                []Point   `json:"path"`
	Success             bool      `json:"success"`
	Message             string    `json:"message,omitempty"`
	PolygonsQueried     int       `json:"polygonsQueried,omitempty"`
	SimplifiedPolygons  []Polygon `json:"simplifiedPolygons,omitempty"`
	VerticesBeforeSimpl int       `json:"verticesBeforeSimplification,omitempty"`
	VerticesAfterSimpl  int       `json:"verticesAfterSimplification,omitempty"`
}

var (
	globalIndex *SpatialIndex
	indexMutex  sync.RWMutex
)

// corsMiddleware adds CORS headers to allow frontend requests
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func routeHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("========================================")
	log.Println("📍 Route request received")

	if r.Method != http.MethodPost {
		log.Printf("❌ Method not allowed: %s\n", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Invalid request body: %v\n", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("   Start: (%.2f, %.2f)\n", req.Start.X, req.Start.Y)
	log.Printf("   End:   (%.2f, %.2f)\n", req.End.X, req.End.Y)

	// Default margin if not specified (e.g., 1000 meters or map units)
	margin := req.Margin
	if margin == 0 {
		margin = 1000.0
	}
	log.Printf("   Margin: %.2f\n", margin)

	var noFlyZones []Polygon

	// Use spatial index to query relevant polygons
	if globalIndex == nil {
		log.Println("❌ Spatial index not initialized")
		http.Error(w, "Spatial index not initialized. Call /createSpatialIndex first", http.StatusBadRequest)
		return
	}

	indexMutex.RLock()
	minX, minY, maxX, maxY := GetRouteBoundingBox(req.Start, req.End, margin)
	log.Printf("   Query bbox: (%.2f, %.2f) to (%.2f, %.2f)\n", minX, minY, maxX, maxY)
	noFlyZones = globalIndex.QueryRegion(minX, minY, maxX, maxY)
	indexMutex.RUnlock()

	log.Printf("   Polygons queried: %d\n", len(noFlyZones))

	// Count vertices and simplify if needed
	totalVertices := 0
	for _, poly := range noFlyZones {
		totalVertices += len(poly.Vertices)
	}
	log.Printf("   Total vertices: %d\n", totalVertices)

	// Track simplification
	verticesBeforeSimplification := 0
	verticesAfterSimplification := 0
	var simplifiedPolygons []Polygon

	// Simplify polygons if too many vertices
	if totalVertices > 1000 {
		verticesBeforeSimplification = totalVertices
		epsilon := EstimateSimplificationEpsilon(noFlyZones, totalVertices)
		log.Printf("⚙️  Simplifying polygons (epsilon: %.6f)...\n", epsilon)

		noFlyZones = SimplifyPolygons(noFlyZones, epsilon)
		simplifiedPolygons = noFlyZones // Store simplified polygons for response

		// Count after simplification
		simplifiedVertices := 0
		for _, poly := range noFlyZones {
			simplifiedVertices += len(poly.Vertices)
		}
		verticesAfterSimplification = simplifiedVertices
		log.Printf("   Vertices after simplification: %d (%.1f%% reduction)\n",
			simplifiedVertices,
			100.0*float64(totalVertices-simplifiedVertices)/float64(totalVertices))
		totalVertices = simplifiedVertices
	}

	// Final check on vertex count
	if totalVertices > 5000 {
		errorMsg := fmt.Sprintf("Too many vertices (%d) even after simplification. Reduce margin or simplify polygons on client side.", totalVertices)
		log.Printf("❌ %s\n", errorMsg)
		http.Error(w, errorMsg, http.StatusBadRequest)
		log.Println("========================================")
		return
	}

	// Build visibility graph from no-fly zones
	log.Println("🔗 Building visibility graph...")
	graph := BuildVisibilityGraph(req.Start, req.End, noFlyZones)
	log.Printf("   Graph nodes: %d\n", len(graph.Nodes))
	edgeCount := 0
	for _, edges := range graph.Edges {
		edgeCount += len(edges)
	}
	log.Printf("   Graph edges: %d\n", edgeCount/2) // Divided by 2 because edges are bidirectional

	if len(graph.Nodes) > 500 {
		log.Printf("💡 TIP: Consider simplifying your polygons to reduce vertices\n")
		log.Printf("   See POLYGON_SIMPLIFICATION.md for guidance\n")
	}

	// Compute A* path on the visibility graph
	// Start is always node 0, end is always node 1 (see BuildVisibilityGraph)
	log.Println("🔍 Running A* pathfinding...")
	path, success := AStarPathOnGraph(graph, 0, 1)

	response := RouteResponse{
		Path:                path,
		Success:             success,
		PolygonsQueried:     len(noFlyZones),
		SimplifiedPolygons:  simplifiedPolygons,
		VerticesBeforeSimpl: verticesBeforeSimplification,
		VerticesAfterSimpl:  verticesAfterSimplification,
	}

	if !success {
		log.Println("❌ No path found")
		response.Message = "No path found"
	} else {
		log.Printf("✅ Path found with %d waypoints\n", len(path))
		log.Println("   Waypoints:")
		for i, p := range path {
			log.Printf("      %d: (%.2f, %.2f)\n", i, p.X, p.Y)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Println("========================================")
}

// POST /createSpatialIndex - Load polygons into spatial index (one-time setup)
func createSpatialIndexHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("========================================")
	log.Println("📦 Create spatial index request received")

	if r.Method != http.MethodPost {
		log.Printf("❌ Method not allowed: %s\n", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if spatial index already exists
	indexMutex.RLock()
	alreadyExists := globalIndex != nil
	indexMutex.RUnlock()

	var req CreateIndexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Invalid request body: %v\n", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if alreadyExists && !req.Force {
		log.Println("⚠️  Spatial index already exists")
		log.Println("   To reload, set force:true in request or restart the server")
		log.Println("========================================")
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Spatial index already exists",
			"message": "Index is already initialized. Set 'force: true' to reload, or restart the server.",
		})
		return
	}

	if alreadyExists && req.Force {
		log.Println("🔄 Force reload requested - recreating spatial index...")
	}

	polygons := req.Polygons
	log.Printf("   Received %d polygons\n", len(polygons))

	// Count total vertices before merging
	totalVertices := 0
	for _, poly := range polygons {
		totalVertices += len(poly.Vertices)
	}
	log.Printf("   Total vertices: %d\n", totalVertices)

	// Merge overlapping/contained polygons to reduce complexity
	if len(polygons) > 1 {
		log.Println("🔀 Merging overlapping polygons...")
		polygons = MergeOverlappingPolygons(polygons)
		log.Printf("   Polygons after merge: %d\n", len(polygons))

		// Count vertices after merging
		mergedVertices := 0
		for _, poly := range polygons {
			mergedVertices += len(poly.Vertices)
		}
		log.Printf("   Vertices after merge: %d\n", mergedVertices)
	}

	log.Println("🔨 Building spatial index...")
	indexMutex.Lock()
	globalIndex = NewSpatialIndex(polygons)
	indexMutex.Unlock()

	log.Printf("✅ Spatial index created successfully\n")
	log.Println("========================================")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"indexed":       len(polygons),
		"totalVertices": totalVertices,
	})
}

// GET /health - Health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	indexMutex.RLock()
	isIndexed := globalIndex != nil
	indexMutex.RUnlock()

	status := "ready"
	if !isIndexed {
		status = "waiting for spatial index"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  status,
		"indexed": isIndexed,
	})
}

func main() {
	http.HandleFunc("/route", corsMiddleware(routeHandler))
	http.HandleFunc("/createSpatialIndex", corsMiddleware(createSpatialIndexHandler))
	http.HandleFunc("/health", corsMiddleware(healthHandler))

	log.Println("========================================")
	log.Println("🚀 Drone Motion Planner Server")
	log.Println("========================================")
	log.Println("Server starting on :8080")
	log.Println("")
	log.Println("Endpoints:")
	log.Println("  POST /createSpatialIndex - Load all no-fly zones once (call this first)")
	log.Println("  POST /route              - Compute route with start and end points")
	log.Println("  GET  /health             - Check server status")
	log.Println("")
	log.Println("CORS enabled for all origins")
	log.Println("========================================")
	log.Println("")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
