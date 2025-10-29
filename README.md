# Drone Motion Planner

A high-performance Go backend for computing collision-free drone routes using visibility graphs and A* pathfinding.

## Features

- **Visibility Graph Construction**: Automatically builds a graph from polygon obstacles
- **A* Pathfinding**: Efficient shortest path computation
- **Spatial Indexing**: R-tree based spatial index for handling large datasets (1600+ polygons, 600k+ vertices)
- **REST API**: Simple HTTP endpoints for route planning

## How It Works

### Architecture

1. **One-time Setup**: Load all no-fly zones into an R-tree spatial index
2. **Per-Query Optimization**: For each route request:
   - Query only relevant polygons within the route's bounding box (+ margin)
   - Build visibility graph from polygon vertices + start/end points
   - Run A* to find shortest collision-free path

### Why This is Fast

With 1600 polygons and 600k vertices total:
- **Without spatial index**: Would need to check all 600k vertices for every route
- **With R-tree index**: Only queries ~hundreds of vertices in the relevant area
- **Typical speedup**: 100-1000x faster for sparse queries

## API Usage

### Step 1: Initialize Spatial Index (One Time)

Load all your no-fly zones once when the server starts or data changes:

```bash
curl -X POST http://localhost:8080/createSpatialIndex \
  -H "Content-Type: application/json" \
  -d '[
    {
      "vertices": [
        {"x": 100, "y": 100},
        {"x": 200, "y": 100},
        {"x": 200, "y": 200},
        {"x": 100, "y": 200}
      ]
    }
  ]'
```

**Response:**
```json
{
  "success": true,
  "indexed": 1600
}
```

### Step 2: Query Routes (Multiple Times)

Each route query only needs start and end points:

```bash
curl -X POST http://localhost:8080/route \
  -H "Content-Type: application/json" \
  -d '{
    "start": {"x": 0, "y": 0},
    "end": {"x": 1000, "y": 1000},
    "margin": 1000
  }'
```

**Response:**
```json
{
  "path": [
    {"x": 0, "y": 0},
    {"x": 150, "y": 200},
    {"x": 1000, "y": 1000}
  ],
  "success": true,
  "polygonsQueried": 47
}
```

### Parameters

#### `/route` endpoint:
- `start` (required): Starting point `{x, y}`
- `end` (required): Ending point `{x, y}`
- `margin` (optional): Distance in map units to query around the route (default: 1000)
  - Larger margin = more polygons checked = slower but safer
  - Smaller margin = faster but might miss obstacles

## Installation and Running

```bash
# Install dependencies
go mod download

# Build
go build

# Run server
./motion-planner

# Or run directly
go run .
```

Server starts on `http://localhost:8080`

### Logging

The server provides detailed logging for debugging:

```
========================================
📍 Route request received
   Start: (0.00, 0.00)
   End:   (1000.00, 1000.00)
   Margin: 1000.00
   Query bbox: (-1000.00, -1000.00) to (2000.00, 2000.00)
   Polygons queried: 47
🔗 Building visibility graph...
   Graph nodes: 234
   Graph edges: 1567
🔍 Running A* pathfinding...
✅ Path found with 5 waypoints
   Waypoints:
      0: (0.00, 0.00)
      1: (100.00, 150.00)
      2: (300.00, 400.00)
      3: (800.00, 900.00)
      4: (1000.00, 1000.00)
========================================
```

### CORS Support

CORS is enabled for all origins, allowing your frontend to connect from any domain.

## Example Usage

See `example_usage.sh` for a complete example:

```bash
./example_usage.sh
```

### Quick Test with HTML Interface

Open `test.html` in your browser for an interactive test interface:

```bash
# Start the server
go run .

# Open test.html in your browser
open test.html  # macOS
# or just drag test.html into your browser
```

The test page allows you to:
- Load GeoJSON no-fly zones
- Compute routes with custom start/end points
- View results in real-time

### Frontend Integration

See `FRONTEND_INTEGRATION.md` for detailed examples with:
- GeoJSON conversion
- React components
- Vue components
- Mapbox/Leaflet integration

## Algorithm Details

### Visibility Graph
- Nodes: Start point + end point + all polygon vertices
- Edges: Direct lines between nodes that don't intersect any obstacle interior
- **Edge routing allowed**: Drones can fly along no-fly zone boundaries (not through them)
- Only edges within the query region are considered (spatial optimization)

### A* Pathfinding
- Heuristic: Euclidean distance to goal
- Cost: Actual distance traveled
- Guarantees shortest collision-free path

### Path Clearance Rules
- ✅ **Allowed**: Flying along polygon edges (on the boundary)
- ✅ **Allowed**: Flying outside all no-fly zones
- ❌ **Blocked**: Flying through polygon interior
- ❌ **Blocked**: Crossing polygon boundaries at non-vertex points

This means the drone can "hug" the edges of no-fly zones to find the shortest path.

### Spatial Optimization
- R-tree indexes polygons by bounding box
- Route queries only load polygons intersecting: `bbox(start, end) + margin`
- Typical reduction: 600k vertices → few hundred vertices per query

## Performance Tips

1. **Margin tuning**: Start with default (1000), adjust based on your obstacle density
2. **Coordinate system**: Use consistent units (meters, pixels, etc.)
3. **Index updates**: Re-call `/createSpatialIndex` only when obstacles change
4. **Concurrent requests**: The spatial index is thread-safe (uses RWMutex)

## Data Format

### Polygon Format
```json
{
  "vertices": [
    {"x": 0, "y": 0},
    {"x": 100, "y": 0},
    {"x": 100, "y": 100},
    {"x": 0, "y": 100}
  ]
}
```
- Vertices should form a closed polygon
- Order: clockwise or counter-clockwise
- No need to repeat the first vertex at the end

## Project Structure

```
motion-planner/
├── main.go              # HTTP server and handlers
├── astar.go             # A* pathfinding algorithm
├── geometry.go          # Geometry utilities (intersections, collisions)
├── visibility_graph.go  # Visibility graph construction
├── spatial_index.go     # R-tree spatial indexing
├── example_usage.sh     # Example API calls
├── go.mod              # Dependencies
└── README.md           # This file
```

## License

MIT
