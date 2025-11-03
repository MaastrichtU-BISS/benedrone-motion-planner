# Drone Motion Planner (PRM-based)

A Go-based HTTP server for drone path planning across the Netherlands using Probabilistic Roadmap (PRM) algorithm with no-fly zone avoidance.

## 🚀 Overview

This service pre-computes a roadmap of safe flight paths and uses it to quickly calculate optimal routes between any two points while respecting airspace restrictions.

## 📋 Features

- **PRM Graph Generation**: Pre-compute navigation graphs with configurable resolution
- **No-Fly Zone Support**: Automatically avoids restricted airspace
- **Fast Route Calculation**: Query routes in milliseconds using A* pathfinding
- **Graph Persistence**: Save/load graphs to avoid rebuilding
- **Auto-Connect**: Start and end points automatically connect to the nearest graph nodes
- **Configurable Resolution**: From 500 to 15,000+ sample points

## 🔧 Quick Start

### Build and Run

```bash
go run .
```

Server starts on `http://localhost:8080`

## API Endpoints

### `POST /route`
Calculate a route between two points.

**Request:**
```json
{
  "start": {"x": 4.9, "y": 52.4},     // Longitude, Latitude
  "end": {"x": 5.7, "y": 50.9},
}
```

**Response:**
```json
{
  "success": true,
  "path": [
    {"x": 4.9, "y": 52.4},
    {"x": 5.1, "y": 52.1},
    {"x": 5.5, "y": 51.5},
    {"x": 5.7, "y": 50.9}
  ],
  "distanceMeters": 145230.45
}
```

### `GET /getPRMGraphLines`
Get graph edges for visualization.

**Response:**
```json
{
  "success": true,
  "lines": [
    [{"x": 4.9, "y": 52.4}, {"x": 5.0, "y": 52.3}],
    [{"x": 5.0, "y": 52.3}, {"x": 5.1, "y": 52.2}]
  ],
  "numNodes": 5000,
  "numEdges": 25000
}
```

### `GET /health`
Check server status.

**Response:**
```json
{
  "status": "ready",
  "hasPRMGraph": true,
  "numNodes": 5000
}
```

## 🔄 Workflow

1. **Build Graph** (once): Pre-compute navigation roadmap
   - Randomly sample points across Netherlands
   - Connect nearby points
   - Avoid no-fly zones
   - Save to disk

2. **Query Routes** (many times): Fast path calculation
   - Load graph from memory
   - Connect start/end to graph
   - Run A* search
   - Return waypoint list

## 🌍 Coverage Area

- **Region**: Netherlands
- **Bounding Box**: 
  - Latitude: 50.75° to 53.55°
  - Longitude: 3.36° to 7.23°

## 🛠️ Technology

- **Language**: Go 1.20+
- **Algorithm**: Probabilistic Roadmap (PRM) + A*
- **Storage**: JSON file persistence
- **API**: REST HTTP with CORS support

