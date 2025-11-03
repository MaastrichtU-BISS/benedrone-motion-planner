package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

// GeoJSON structures for parsing no-fly zone files
type GeoJSONFeature struct {
	Type       string                 `json:"type"`
	Geometry   GeoJSONGeometry        `json:"geometry"`
	Properties map[string]interface{} `json:"properties"`
}

type GeoJSONGeometry struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

type GeoJSONFeatureCollection struct {
	Type     string           `json:"type"`
	Features []GeoJSONFeature `json:"features"`
}

// loadNoFlyZonesFromFiles loads all GeoJSON files from the nfz-polygons directory
func loadNoFlyZonesFromFiles() ([]Polygon, error) {
	nfzDir := "nfz-polygons"
	var allPolygons []Polygon

	files, err := filepath.Glob(filepath.Join(nfzDir, "*.geojson"))
	if err != nil {
		return nil, err
	}

	log.Printf("Loading no-fly zones from %d GeoJSON files...\n", len(files))

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			log.Printf("⚠️  Failed to read %s: %v\n", file, err)
			continue
		}

		var featureCollection GeoJSONFeatureCollection
		if err := json.Unmarshal(data, &featureCollection); err != nil {
			log.Printf("⚠️  Failed to parse %s: %v\n", file, err)
			continue
		}

		polygonCount := 0
		for _, feature := range featureCollection.Features {
			polygons := parseGeoJSONGeometry(feature.Geometry)
			allPolygons = append(allPolygons, polygons...)
			polygonCount += len(polygons)
		}

		log.Printf("   ✅ Loaded %d polygons from %s\n", polygonCount, filepath.Base(file))
	}

	log.Printf("Total no-fly zones loaded: %d polygons\n", len(allPolygons))
	return allPolygons, nil
}

// parseGeoJSONGeometry converts GeoJSON geometry to our Polygon format
func parseGeoJSONGeometry(geometry GeoJSONGeometry) []Polygon {
	var polygons []Polygon

	switch geometry.Type {
	case "Polygon":
		var coords [][][]float64
		if err := json.Unmarshal(geometry.Coordinates, &coords); err != nil {
			log.Printf("⚠️  Failed to parse Polygon coordinates: %v\n", err)
			return polygons
		}
		// First ring is the outer boundary
		if len(coords) > 0 {
			polygon := Polygon{Vertices: make([]Point, 0, len(coords[0]))}
			for _, coord := range coords[0] {
				if len(coord) >= 2 {
					polygon.Vertices = append(polygon.Vertices, Point{X: coord[0], Y: coord[1]})
				}
			}
			polygons = append(polygons, polygon)
		}

	case "MultiPolygon":
		var coords [][][][]float64
		if err := json.Unmarshal(geometry.Coordinates, &coords); err != nil {
			log.Printf("⚠️  Failed to parse MultiPolygon coordinates: %v\n", err)
			return polygons
		}
		for _, polyCoords := range coords {
			if len(polyCoords) > 0 {
				polygon := Polygon{Vertices: make([]Point, 0, len(polyCoords[0]))}
				for _, coord := range polyCoords[0] {
					if len(coord) >= 2 {
						polygon.Vertices = append(polygon.Vertices, Point{X: coord[0], Y: coord[1]})
					}
				}
				polygons = append(polygons, polygon)
			}
		}
	}

	return polygons
}
