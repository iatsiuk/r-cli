//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"r-cli/internal/reql"
)

func TestMapField(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "score": 10},
		{"id": "2", "score": 20},
		{"id": "3", "score": 30},
	})

	fn := reql.Func(reql.Var(1).GetField("score"), 1)
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Map(fn), nil)
	if err != nil {
		t.Fatalf("map: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}

	scores := make(map[float64]bool)
	for _, raw := range rows {
		var v float64
		if err := json.Unmarshal(raw, &v); err != nil {
			t.Fatalf("unmarshal score: %v", err)
		}
		scores[v] = true
	}
	for _, want := range []float64{10, 20, 30} {
		if !scores[want] {
			t.Errorf("missing score %v in map result", want)
		}
	}
}

func TestReduceSum(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "score": 10},
		{"id": "2", "score": 20},
		{"id": "3", "score": 30},
	})

	// map to score then reduce with add: sum = 10+20+30 = 60
	mapFn := reql.Func(reql.Var(1).GetField("score"), 1)
	reduceFn := reql.Func(reql.Var(1).Add(reql.Var(2)), 1, 2)
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Map(mapFn).Reduce(reduceFn), nil)
	if err != nil {
		t.Fatalf("reduce: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var sum float64
	if err := json.Unmarshal(raw, &sum); err != nil {
		t.Fatalf("unmarshal sum: %v", err)
	}
	if sum != 60 {
		t.Errorf("sum=%v, want 60", sum)
	}
}

func TestGroupByField(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "category": "a", "val": 1},
		{"id": "2", "category": "b", "val": 2},
		{"id": "3", "category": "a", "val": 3},
		{"id": "4", "category": "b", "val": 4},
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Group("category"), nil)
	if err != nil {
		t.Fatalf("group: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}

	// GROUPED_DATA: {"$reql_type$": "GROUPED_DATA", "data": [["a", [doc1, doc2]], ["b", [doc3, doc4]]]}
	var gd struct {
		Data [][2]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &gd); err != nil {
		t.Fatalf("unmarshal grouped data: %v", err)
	}
	if len(gd.Data) != 2 {
		t.Fatalf("got %d groups, want 2", len(gd.Data))
	}

	groupSizes := make(map[string]int)
	for _, pair := range gd.Data {
		var key string
		if err := json.Unmarshal(pair[0], &key); err != nil {
			t.Fatalf("unmarshal group key: %v", err)
		}
		var docs []json.RawMessage
		if err := json.Unmarshal(pair[1], &docs); err != nil {
			t.Fatalf("unmarshal group docs: %v", err)
		}
		groupSizes[key] = len(docs)
	}
	if groupSizes["a"] != 2 {
		t.Errorf("group 'a' has %d docs, want 2", groupSizes["a"])
	}
	if groupSizes["b"] != 2 {
		t.Errorf("group 'b' has %d docs, want 2", groupSizes["b"])
	}
}

func TestGroupCount(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "type": "x"},
		{"id": "2", "type": "y"},
		{"id": "3", "type": "x"},
		{"id": "4", "type": "y"},
		{"id": "5", "type": "z"},
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Group("type").Count(), nil)
	if err != nil {
		t.Fatalf("group count: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}

	// GROUPED_DATA: {"$reql_type$": "GROUPED_DATA", "data": [["x", 2], ["y", 2], ["z", 1]]}
	var gd struct {
		Data [][2]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &gd); err != nil {
		t.Fatalf("unmarshal grouped data: %v", err)
	}
	if len(gd.Data) != 3 {
		t.Fatalf("got %d groups, want 3", len(gd.Data))
	}

	counts := make(map[string]float64)
	for _, pair := range gd.Data {
		var key string
		if err := json.Unmarshal(pair[0], &key); err != nil {
			t.Fatalf("unmarshal group key: %v", err)
		}
		var count float64
		if err := json.Unmarshal(pair[1], &count); err != nil {
			t.Fatalf("unmarshal count: %v", err)
		}
		counts[key] = count
	}
	if counts["x"] != 2 {
		t.Errorf("type 'x' count=%v, want 2", counts["x"])
	}
	if counts["y"] != 2 {
		t.Errorf("type 'y' count=%v, want 2", counts["y"])
	}
	if counts["z"] != 1 {
		t.Errorf("type 'z' count=%v, want 1", counts["z"])
	}
}

func TestUngroup(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "tag": "foo"},
		{"id": "2", "tag": "bar"},
		{"id": "3", "tag": "foo"},
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Group("tag").Count().Ungroup(), nil)
	if err != nil {
		t.Fatalf("ungroup: %v", err)
	}
	defer closeCursor(cur)

	// ungroup returns SUCCESS_ATOM containing an array of {group, reduction} objects
	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var rows []struct {
		Group     string  `json:"group"`
		Reduction float64 `json:"reduction"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatalf("unmarshal ungroup rows: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows after ungroup, want 2", len(rows))
	}
	for i, obj := range rows {
		if obj.Group == "" {
			t.Errorf("row %d: ungroup row missing group field", i)
		}
		if obj.Reduction <= 0 {
			t.Errorf("row %d: reduction=%v, want > 0", i, obj.Reduction)
		}
	}
}
