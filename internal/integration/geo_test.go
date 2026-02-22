//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"r-cli/internal/reql"
)

func TestGeoGetNearest(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "places")

	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("places").IndexCreate("location", reql.OptArgs{"geo": true}), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("index create: %v", err)
	}
	_, cur, err = exec.Run(ctx,
		reql.DB(dbName).Table("places").IndexWait("location"), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("index wait: %v", err)
	}

	// insert three points: times square (origin), central park (~2.1km), brooklyn (~8.5km)
	docs := []interface{}{
		map[string]interface{}{"id": "times_square", "location": reql.Point(-73.9857, 40.7484)},
		map[string]interface{}{"id": "central_park", "location": reql.Point(-73.9712, 40.7614)},
		map[string]interface{}{"id": "brooklyn", "location": reql.Point(-73.9442, 40.6782)},
	}
	_, cur, err = exec.Run(ctx,
		reql.DB(dbName).Table("places").Insert(reql.Array(docs...)), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("insert geo docs: %v", err)
	}

	_, cur, err = exec.Run(ctx,
		reql.DB(dbName).Table("places").GetNearest(
			reql.Point(-73.9857, 40.7484),
			reql.OptArgs{"index": "location", "max_results": 3},
		), nil)
	if err != nil {
		t.Fatalf("get nearest: %v", err)
	}
	defer closeCursor(cur)

	// GetNearest returns SUCCESS_ATOM: a single item containing the full result array
	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	type nearestItem struct {
		Dist float64         `json:"dist"`
		Doc  json.RawMessage `json:"doc"`
	}
	var items []nearestItem
	if err := json.Unmarshal(raw, &items); err != nil {
		t.Fatalf("unmarshal nearest result: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3", len(items))
	}

	// first result should be times_square with near-zero distance
	var firstDoc struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(items[0].Doc, &firstDoc); err != nil {
		t.Fatalf("unmarshal first doc: %v", err)
	}
	if firstDoc.ID != "times_square" {
		t.Errorf("nearest id=%q, want times_square", firstDoc.ID)
	}
	if items[0].Dist > 1.0 {
		t.Errorf("distance to self=%f, want ~0", items[0].Dist)
	}

	// verify distances are sorted ascending
	for i := 1; i < len(items); i++ {
		if items[i].Dist < items[i-1].Dist {
			t.Errorf("item %d: dist %f < prev %f (not sorted ascending)", i, items[i].Dist, items[i-1].Dist)
		}
	}
}

func TestGeoGetIntersecting(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "places")

	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("places").IndexCreate("location", reql.OptArgs{"geo": true}), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("index create: %v", err)
	}
	_, cur, err = exec.Run(ctx,
		reql.DB(dbName).Table("places").IndexWait("location"), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("index wait: %v", err)
	}

	// insert two points: one inside the test polygon, one outside
	docs := []interface{}{
		map[string]interface{}{"id": "inside", "location": reql.Point(-73.9857, 40.7484)},
		map[string]interface{}{"id": "outside", "location": reql.Point(-73.9442, 40.6782)},
	}
	_, cur, err = exec.Run(ctx,
		reql.DB(dbName).Table("places").Insert(reql.Array(docs...)), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("insert geo docs: %v", err)
	}

	// polygon covering midtown manhattan: lon [-74.01, -73.95], lat [40.73, 40.77]
	poly := reql.Polygon(
		reql.Point(-74.01, 40.73),
		reql.Point(-74.01, 40.77),
		reql.Point(-73.95, 40.77),
		reql.Point(-73.95, 40.73),
	)

	_, cur, err = exec.Run(ctx,
		reql.DB(dbName).Table("places").GetIntersecting(poly, reql.OptArgs{"index": "location"}), nil)
	if err != nil {
		t.Fatalf("get intersecting: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	var doc struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rows[0], &doc); err != nil {
		t.Fatalf("unmarshal doc: %v", err)
	}
	if doc.ID != "inside" {
		t.Errorf("intersecting doc id=%q, want inside", doc.ID)
	}
}

func TestGeoDistance(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()

	// distance between times square and central park (~2.1 km)
	timesSquare := reql.Point(-73.9857, 40.7484)
	centralPark := reql.Point(-73.9712, 40.7614)

	_, cur, err := exec.Run(ctx, timesSquare.Distance(centralPark), nil)
	if err != nil {
		t.Fatalf("distance: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var dist float64
	if err := json.Unmarshal(raw, &dist); err != nil {
		t.Fatalf("unmarshal distance: %v", err)
	}
	// roughly 2100 meters, allow generous bounds
	if dist < 1000 || dist > 5000 {
		t.Errorf("distance=%f meters, expected roughly 2100 (between 1000 and 5000)", dist)
	}
}
