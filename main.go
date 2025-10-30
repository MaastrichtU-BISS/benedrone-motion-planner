package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
)

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type BoundingBox struct {
	MinX float64 `json:"minX"`
	MinY float64 `json:"minY"`
	MaxX float64 `json:"maxX"`
	MaxY float64 `json:"maxY"`
}

type RouteRequest struct {
	Start      Point     `json:"start"`
	End        Point     `json:"end"`
	NoFlyZones []Polygon `json:"noFlyZones,omitempty"` // Optional: for checking start/end connections
}

type RouteResponse struct {
	Path           []Point `json:"path"`
	Success        bool    `json:"success"`
	Message        string  `json:"message,omitempty"`
	DistanceMeters float64 `json:"distanceMeters,omitempty"`
}

var (
	globalPRMGraph *PRMGraph
	prmMutex       sync.RWMutex
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

	log.Printf("   Start: (%.6f, %.6f)\n", req.Start.X, req.Start.Y)
	log.Printf("   End:   (%.6f, %.6f)\n", req.End.X, req.End.Y)

	// Check if PRM graph is available
	prmMutex.RLock()
	prmGraph := globalPRMGraph
	prmMutex.RUnlock()

	if prmGraph == nil {
		log.Println("❌ PRM graph not available")
		http.Error(w, "PRM graph not built. Call /buildPRMGraph first", http.StatusBadRequest)
		log.Println("========================================")
		return
	}

	// Create a temporary graph with start and end points connected
	log.Println("🔗 Connecting start and end points to graph...")
	tempGraph, startNodeID, endNodeID := prmGraph.CreateGraphWithStartEnd(req.Start, req.End, req.NoFlyZones)

	if startNodeID == -1 || endNodeID == -1 {
		log.Println("❌ Could not connect start or end point to graph")
		response := RouteResponse{
			Success: false,
			Message: "Could not connect start or end point to the graph (possibly blocked by no-fly zones)",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		log.Println("========================================")
		return
	}

	log.Printf("   ✅ Start connected as node %d\n", startNodeID)
	log.Printf("   ✅ End connected as node %d\n", endNodeID)

	// Convert to standard graph format
	graph := tempGraph.ConvertToGraph()

	// Run A* on the graph with start and end
	log.Println("🔍 Running A* on PRM graph...")
	path, success := AStarPathOnGraph(graph, startNodeID, endNodeID)

	// Calculate distance
	var distanceMeters float64
	if success && len(path) > 1 {
		for i := 0; i < len(path)-1; i++ {
			distanceMeters += path[i].DistanceMeters(path[i+1])
		}
	}

	response := RouteResponse{
		Path:           path,
		Success:        success,
		DistanceMeters: distanceMeters,
	}

	if !success {
		log.Println("❌ No path found on PRM graph")
		response.Message = "No path found on PRM graph"
	} else {
		log.Printf("✅ Path found with %d waypoints\n", len(path))
		log.Printf("   Distance: %.2f meters (%.2f km)\n", distanceMeters, distanceMeters/1000)
		log.Println("   Path preview (first/last 3 waypoints):")
		for i := 0; i < len(path) && i < 3; i++ {
			log.Printf("      %d: (%.6f, %.6f)\n", i, path[i].X, path[i].Y)
		}
		if len(path) > 6 {
			log.Printf("      ... (%d intermediate waypoints)\n", len(path)-6)
			startIdx := len(path) - 3
			for i := startIdx; i < len(path); i++ {
				log.Printf("      %d: (%.6f, %.6f)\n", i, path[i].X, path[i].Y)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Println("========================================")
}

// GET /health - Health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	prmMutex.RLock()
	hasPRMGraph := globalPRMGraph != nil
	numNodes := 0
	if globalPRMGraph != nil {
		numNodes = len(globalPRMGraph.Nodes)
	}
	prmMutex.RUnlock()

	status := "ready"
	if !hasPRMGraph {
		status = "waiting for PRM graph"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      status,
		"hasPRMGraph": hasPRMGraph,
		"numNodes":    numNodes,
	})
}

// POST /buildPRMGraph - Build a probabilistic roadmap for the Netherlands
func buildPRMGraphHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("========================================")
	log.Println("🗺️  Build PRM Graph request received")

	if r.Method != http.MethodPost {
		log.Printf("❌ Method not allowed: %s\n", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type BuildPRMRequest struct {
		NumSamples       int       `json:"numSamples"`       // Number of random samples
		ConnectionRadius float64   `json:"connectionRadius"` // Connection radius in degrees
		SaveToFile       bool      `json:"saveToFile"`       // Whether to save to disk
		Force            bool      `json:"force,omitempty"`  // Set to true to force rebuild
		NoFlyZones       []Polygon `json:"noFlyZones"`       // No-fly zone polygons
	}

	var req BuildPRMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Invalid request body: %v\n", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check if PRM graph already exists
	prmMutex.RLock()
	alreadyExists := globalPRMGraph != nil
	prmMutex.RUnlock()

	if alreadyExists && !req.Force {
		log.Println("⚠️  PRM graph already exists")
		log.Println("   To rebuild, set force:true in request or restart the server")
		log.Println("========================================")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "PRM graph already exists",
			"message": "Graph is already built. Set 'force: true' to rebuild, or restart the server.",
		})
		return
	}

	if alreadyExists && req.Force {
		log.Println("🔄 Force rebuild requested - recreating PRM graph...")
	}

	// Set defaults
	if req.NumSamples == 0 {
		req.NumSamples = 500 // Low precision default
	}
	if req.ConnectionRadius == 0 {
		req.ConnectionRadius = 0.1 // ~11 km
	}

	log.Printf("   Samples: %d\n", req.NumSamples)
	log.Printf("   Connection radius: %.4f degrees\n", req.ConnectionRadius)
	log.Printf("   No-fly zones: %d polygons\n", len(req.NoFlyZones))

	// Build the graph
	graph := BuildPRMGraph(req.NumSamples, req.ConnectionRadius, req.NoFlyZones)

	// Save to global variable
	prmMutex.Lock()
	globalPRMGraph = graph
	prmMutex.Unlock()

	// Optionally save to file
	if req.SaveToFile {
		if err := SavePRMGraph(graph, "prm_graph.json"); err != nil {
			log.Printf("⚠️  Failed to save graph: %v\n", err)
		}
	}

	log.Printf("✅ PRM graph built and stored in memory\n")
	log.Println("========================================")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"numNodes":   len(graph.Nodes),
		"numSamples": req.NumSamples,
		"boundingBox": map[string]float64{
			"minLat": graph.BoundingBox.MinLat,
			"maxLat": graph.BoundingBox.MaxLat,
			"minLon": graph.BoundingBox.MinLon,
			"maxLon": graph.BoundingBox.MaxLon,
		},
	})
}

// GET /getPRMGraphLines - Get graph edges as line strings for visualization
func getPRMGraphLinesHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("========================================")
	log.Println("📊 Get PRM Graph Lines request received")

	if r.Method != http.MethodGet {
		log.Printf("❌ Method not allowed: %s\n", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	prmMutex.RLock()
	graph := globalPRMGraph
	prmMutex.RUnlock()

	if graph == nil {
		log.Println("❌ PRM graph not built")
		http.Error(w, "PRM graph not built. Call /buildPRMGraph first", http.StatusBadRequest)
		log.Println("========================================")
		return
	}

	lines := graph.GetGraphAsLineStrings()

	log.Printf("   Returning %d line segments\n", len(lines))
	log.Println("========================================")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"lines":    lines,
		"numNodes": len(graph.Nodes),
		"numEdges": len(lines),
	})
}

func main() {
	// Try to load existing PRM graph from file on startup
	log.Println("========================================")
	log.Println("🚀 Drone Motion Planner Server (PRM-based)")
	log.Println("========================================")
	log.Println("Checking for existing PRM graph file...")

	if graph, err := LoadPRMGraph("prm_graph.json"); err == nil {
		prmMutex.Lock()
		globalPRMGraph = graph
		prmMutex.Unlock()
		log.Printf("✅ Loaded existing PRM graph from file\n")
		log.Printf("   Nodes: %d\n", len(graph.Nodes))
		log.Printf("   Bounding box: (%.2f, %.2f) to (%.2f, %.2f)\n",
			graph.BoundingBox.MinLon, graph.BoundingBox.MinLat,
			graph.BoundingBox.MaxLon, graph.BoundingBox.MaxLat)
	} else {
		log.Println("ℹ️  No existing graph found (this is normal on first run)")
		log.Println("   Call /buildPRMGraph to create a new graph")
	}
	log.Println("")

	http.HandleFunc("/route", corsMiddleware(routeHandler))
	http.HandleFunc("/buildPRMGraph", corsMiddleware(buildPRMGraphHandler))
	http.HandleFunc("/getPRMGraphLines", corsMiddleware(getPRMGraphLinesHandler))
	http.HandleFunc("/health", corsMiddleware(healthHandler))

	log.Println("Server starting on :8080")
	log.Println("")
	log.Println("Endpoints:")
	log.Println("  POST /buildPRMGraph      - Build probabilistic roadmap (PRM)")
	log.Println("  GET  /getPRMGraphLines   - Get PRM graph edges for visualization")
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
