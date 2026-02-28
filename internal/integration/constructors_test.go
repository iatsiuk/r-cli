//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"r-cli/internal/reql"
	"r-cli/internal/response"
)

func TestMinValMaxValBetween(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "aaa", "v": 1},
		{"id": "bbb", "v": 2},
		{"id": "mmm", "v": 3},
		{"id": "zzz", "v": 4},
	})

	// between(minval, maxval) returns all docs
	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").Between(reql.MinVal(), reql.MaxVal()), nil)
	if err != nil {
		t.Fatalf("between(minval, maxval): %v", err)
	}
	rows, err := cur.All()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 4 {
		t.Errorf("between(minval, maxval) got %d rows, want 4", len(rows))
	}

	// between(minval, "m") returns docs with id < "m"
	_, cur2, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").Between(reql.MinVal(), "m"), nil)
	if err != nil {
		t.Fatalf("between(minval, 'm'): %v", err)
	}
	rows2, err := cur2.All()
	closeCursor(cur2)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	// "aaa" and "bbb" are < "m"; "mmm" >= "m"
	if len(rows2) != 2 {
		t.Errorf("between(minval, 'm') got %d rows, want 2", len(rows2))
	}

	// between("m", maxval) returns docs with id >= "m"
	_, cur3, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").Between("m", reql.MaxVal()), nil)
	if err != nil {
		t.Fatalf("between('m', maxval): %v", err)
	}
	rows3, err := cur3.All()
	closeCursor(cur3)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	// "mmm" and "zzz" are >= "m"
	if len(rows3) != 2 {
		t.Errorf("between('m', maxval) got %d rows, want 2", len(rows3))
	}
}

func TestErrorReturnsRuntimeError(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	// r.error("boom") should return a ReqlRuntimeError
	_, cur, err := exec.Run(ctx, reql.Error("boom"), nil)
	closeCursor(cur)
	if err == nil {
		t.Fatal("expected error from r.error(), got nil")
	}
	var runtimeErr *response.ReqlRuntimeError
	if !errors.As(err, &runtimeErr) {
		t.Errorf("expected ReqlRuntimeError, got %T: %v", err, err)
	}
}

func TestErrorInBranchFalsePath(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	// branch(false, "ok", r.error("bad")) triggers error on false branch
	_, cur, err := exec.Run(ctx,
		reql.Branch(reql.Datum(false), reql.Datum("ok"), reql.Error("bad")), nil)
	closeCursor(cur)
	if err == nil {
		t.Fatal("expected error from false branch, got nil")
	}
	var runtimeErr *response.ReqlRuntimeError
	if !errors.As(err, &runtimeErr) {
		t.Errorf("expected ReqlRuntimeError from false branch, got %T: %v", err, err)
	}

	// branch(true, "ok", r.error("bad")) returns "ok"
	_, cur2, err := exec.Run(ctx,
		reql.Branch(reql.Datum(true), reql.Datum("ok"), reql.Error("bad")), nil)
	if err != nil {
		t.Fatalf("expected ok from true branch: %v", err)
	}
	defer closeCursor(cur2)
	raw, err := cur2.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var result string
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result != "ok" {
		t.Errorf("branch true result=%q, want ok", result)
	}
}

func TestArgsWithGetAll(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "alpha", "v": 1},
		{"id": "beta", "v": 2},
		{"id": "gamma", "v": 3},
	})

	// r.args(["alpha","beta"]) spreads as arguments to getAll
	argsArray := reql.Array("alpha", "beta")
	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").GetAll(reql.Args(argsArray)), nil)
	if err != nil {
		t.Fatalf("getAll(r.args(...)): %v", err)
	}
	rows, err := cur.All()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("getAll with args got %d rows, want 2", len(rows))
	}

	// verify only "alpha" and "beta" are returned
	ids := make(map[string]bool)
	for _, raw := range rows {
		var doc struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		ids[doc.ID] = true
	}
	if !ids["alpha"] || !ids["beta"] {
		t.Errorf("getAll with args returned ids=%v, want alpha and beta", ids)
	}
}

func TestLiteralReplacesNestedObject(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	// insert doc with nested object {"meta": {"a": 1, "b": 2}}
	_, cur0, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").Insert(reql.JSON(`{"id":"lit1","meta":{"a":1,"b":2}}`)), nil)
	closeCursor(cur0)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	// normal merge update: {"meta": {"a": 99}} merges, keeps "b"
	_, cur1, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").Get("lit1").Update(
			map[string]interface{}{"meta": map[string]interface{}{"a": 99}},
		), nil)
	closeCursor(cur1)
	if err != nil {
		t.Fatalf("normal update: %v", err)
	}
	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Get("lit1"), nil)
	if err != nil {
		t.Fatalf("get after normal update: %v", err)
	}
	raw, err := cur2.Next()
	closeCursor(cur2)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var doc1 struct {
		Meta map[string]json.RawMessage `json:"meta"`
	}
	if err := json.Unmarshal(raw, &doc1); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// after normal merge, both "a" and "b" should exist
	if _, ok := doc1.Meta["b"]; !ok {
		t.Error("normal merge should keep field b, but it's missing")
	}

	// literal update: r.literal({"x": 9}) replaces meta entirely -- no "a" or "b"
	_, cur3, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").Get("lit1").Update(
			map[string]interface{}{"meta": reql.Literal(map[string]interface{}{"x": 9})},
		), nil)
	closeCursor(cur3)
	if err != nil {
		t.Fatalf("literal update: %v", err)
	}
	_, cur4, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Get("lit1"), nil)
	if err != nil {
		t.Fatalf("get after literal update: %v", err)
	}
	raw2, err := cur4.Next()
	closeCursor(cur4)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var doc2 struct {
		Meta map[string]json.RawMessage `json:"meta"`
	}
	if err := json.Unmarshal(raw2, &doc2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// after literal, only "x" should exist
	if _, ok := doc2.Meta["a"]; ok {
		t.Error("literal replace should remove field a")
	}
	if _, ok := doc2.Meta["b"]; ok {
		t.Error("literal replace should remove field b")
	}
	if _, ok := doc2.Meta["x"]; !ok {
		t.Error("literal replace should set field x")
	}
}

func TestGeoJSONCreateAndQuery(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "places")

	// create geo index
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

	// create a point via r.geoJSON and insert it; use reql.JSON to safely
	// encode the coordinates array as a JSON string (raw []interface{} slices
	// are interpreted as ReQL term arrays on the wire)
	_, cur, err = exec.Run(ctx,
		reql.DB(dbName).Table("places").Insert(map[string]interface{}{
			"id":       "ts",
			"location": reql.GeoJSON(reql.JSON(`{"type":"Point","coordinates":[-73.9857,40.7484]}`)),
		}), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("insert geojson point: %v", err)
	}

	// insert an outside point for contrast
	_, cur, err = exec.Run(ctx,
		reql.DB(dbName).Table("places").Insert(map[string]interface{}{
			"id":       "far",
			"location": reql.Point(-73.9442, 40.6782),
		}), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("insert far point: %v", err)
	}

	// query with a polygon covering the first point only
	poly := reql.Polygon(
		reql.Point(-74.01, 40.73),
		reql.Point(-74.01, 40.77),
		reql.Point(-73.95, 40.77),
		reql.Point(-73.95, 40.73),
	)
	_, cur2, err := exec.Run(ctx,
		reql.DB(dbName).Table("places").GetIntersecting(poly, reql.OptArgs{"index": "location"}), nil)
	if err != nil {
		t.Fatalf("getIntersecting: %v", err)
	}
	rows, err := cur2.All()
	closeCursor(cur2)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("getIntersecting got %d rows, want 1", len(rows))
	}
	var doc struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rows[0], &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if doc.ID != "ts" {
		t.Errorf("intersecting doc id=%q, want ts", doc.ID)
	}
}
