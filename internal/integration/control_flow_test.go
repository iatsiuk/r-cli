//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"r-cli/internal/reql"
)

func TestUpdateWithBranch(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "a", "score": 80},
		{"id": "b", "score": 30},
		{"id": "c", "score": 50},
	})

	// update each doc: status = branch(score > 50, "pass", "fail")
	updateFn := reql.Func(
		reql.Branch(
			reql.Var(1).GetField("score").Gt(50),
			map[string]interface{}{"status": "pass"},
			map[string]interface{}{"status": "fail"},
		), 1)
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Update(updateFn), nil)
	if err != nil {
		t.Fatalf("update with branch: %v", err)
	}
	r := parseWriteResult(t, cur)
	if r.Replaced != 3 {
		t.Errorf("replaced=%d, want 3", r.Replaced)
	}

	// verify conditional results; orderBy without index returns SUCCESS_ATOM (array)
	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("docs").OrderBy("id"), nil)
	if err != nil {
		t.Fatalf("get docs: %v", err)
	}

	rows := atomRows(t, cur2)
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}

	expected := map[string]string{
		"a": "pass", // score 80 > 50
		"b": "fail", // score 30 <= 50
		"c": "fail", // score 50 not > 50
	}
	for _, raw := range rows {
		var doc struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal doc: %v", err)
		}
		want, ok := expected[doc.ID]
		if !ok {
			t.Errorf("unexpected doc id %q", doc.ID)
			continue
		}
		if doc.Status != want {
			t.Errorf("doc %q: status=%q, want %q", doc.ID, doc.Status, want)
		}
	}
}

func TestForEachInsertIntoTable(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "src")
	createTestTable(t, exec, dbName, "dst")
	seedTable(t, exec, dbName, "src", []map[string]interface{}{
		{"id": "1", "val": 10},
		{"id": "2", "val": 20},
		{"id": "3", "val": 30},
	})

	// forEach: for each doc in src, insert {id, doubled: val*2} into dst
	insertFn := reql.Func(
		reql.DB(dbName).Table("dst").Insert(
			map[string]interface{}{
				"id":      reql.Var(1).GetField("id"),
				"doubled": reql.Var(1).GetField("val").Mul(2),
			},
		), 1)
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("src").ForEach(insertFn), nil)
	if err != nil {
		t.Fatalf("forEach: %v", err)
	}
	closeCursor(cur)

	// verify dst has 3 docs
	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("dst").Count(), nil)
	if err != nil {
		t.Fatalf("count dst: %v", err)
	}
	defer closeCursor(cur2)

	raw, err := cur2.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var count float64
	if err := json.Unmarshal(raw, &count); err != nil {
		t.Fatalf("unmarshal count: %v", err)
	}
	if int(count) != 3 {
		t.Errorf("dst count=%d, want 3", int(count))
	}

	// verify one doc has doubled value
	_, cur3, err := exec.Run(ctx, reql.DB(dbName).Table("dst").Get("1"), nil)
	if err != nil {
		t.Fatalf("get dst doc: %v", err)
	}
	defer closeCursor(cur3)

	raw2, err := cur3.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var doc struct {
		Doubled float64 `json:"doubled"`
	}
	if err := json.Unmarshal(raw2, &doc); err != nil {
		t.Fatalf("unmarshal dst doc: %v", err)
	}
	if doc.Doubled != 20 {
		t.Errorf("doubled=%v, want 20", doc.Doubled)
	}
}

func TestDefaultOnMissingField(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "color": "red"},
		{"id": "2"}, // no color field
		{"id": "3", "color": "blue"},
	})

	// map each doc to its color, defaulting to "unknown"
	mapFn := reql.Func(reql.Var(1).GetField("color").Default("unknown"), 1)
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Map(mapFn), nil)
	if err != nil {
		t.Fatalf("map with default: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}

	colors := make(map[string]int)
	for _, raw := range rows {
		var color string
		if err := json.Unmarshal(raw, &color); err != nil {
			t.Fatalf("unmarshal color: %v", err)
		}
		colors[color]++
	}
	if colors["unknown"] != 1 {
		t.Errorf("unknown count=%d, want 1", colors["unknown"])
	}
	if colors["red"] != 1 {
		t.Errorf("red count=%d, want 1", colors["red"])
	}
	if colors["blue"] != 1 {
		t.Errorf("blue count=%d, want 1", colors["blue"])
	}
}
