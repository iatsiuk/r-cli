//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"r-cli/internal/reql/parser"
)

func TestGeoParser_Line(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	term, err := parser.Parse(`r.line(r.point(-74.01, 40.73), r.point(-73.95, 40.77)).toGeoJSON()`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, cur, err := exec.Run(ctx, term, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	var gj struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &gj); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if gj.Type != "LineString" {
		t.Errorf("r.line toGeoJSON type=%q, want LineString", gj.Type)
	}
}

func TestGeoParser_Polygon(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	term, err := parser.Parse(`r.polygon(r.point(-74.01, 40.73), r.point(-74.01, 40.77), r.point(-73.95, 40.77)).toGeoJSON()`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, cur, err := exec.Run(ctx, term, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	var gj struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &gj); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if gj.Type != "Polygon" {
		t.Errorf("r.polygon toGeoJSON type=%q, want Polygon", gj.Type)
	}
}

func TestGeoParser_Circle(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	// circle returns a Polygon approximating the circle
	term, err := parser.Parse(`r.circle(r.point(-73.9857, 40.7484), 1000).toGeoJSON()`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, cur, err := exec.Run(ctx, term, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	var gj struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &gj); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if gj.Type != "Polygon" {
		t.Errorf("r.circle toGeoJSON type=%q, want Polygon", gj.Type)
	}
}
