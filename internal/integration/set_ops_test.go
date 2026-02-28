//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"sort"
	"testing"

	"r-cli/internal/reql"
)

func TestSetInsert(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	// inserting existing element -- no duplicate
	_, cur, err := exec.Run(ctx, reql.Array(1, 2, 3).SetInsert(2), nil)
	if err != nil {
		t.Fatalf("setInsert dup: %v", err)
	}
	raw, err := cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got []float64
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("setInsert dup got %v, want 3 elements", got)
	}

	// inserting new element -- appended
	_, cur2, err := exec.Run(ctx, reql.Array(1, 2, 3).SetInsert(4), nil)
	if err != nil {
		t.Fatalf("setInsert new: %v", err)
	}
	raw2, err := cur2.Next()
	closeCursor(cur2)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got2 []float64
	if err := json.Unmarshal(raw2, &got2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got2) != 4 {
		t.Errorf("setInsert new got %v, want 4 elements", got2)
	}
	sort.Float64s(got2)
	want2 := []float64{1, 2, 3, 4}
	for i, v := range want2 {
		if got2[i] != v {
			t.Errorf("setInsert new element %d = %v, want %v", i, got2[i], v)
		}
	}
}

func TestSetIntersection(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	_, cur, err := exec.Run(ctx,
		reql.Array(1, 2, 3).SetIntersection(reql.Array(2, 3, 4)), nil)
	if err != nil {
		t.Fatalf("setIntersection: %v", err)
	}
	raw, err := cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got []float64
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	sort.Float64s(got)
	if len(got) != 2 {
		t.Fatalf("setIntersection got %v, want 2 elements", got)
	}
	if got[0] != 2 || got[1] != 3 {
		t.Errorf("setIntersection = %v, want [2 3]", got)
	}
}

func TestSetUnion(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	_, cur, err := exec.Run(ctx,
		reql.Array(1, 2).SetUnion(reql.Array(2, 3)), nil)
	if err != nil {
		t.Fatalf("setUnion: %v", err)
	}
	raw, err := cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got []float64
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	sort.Float64s(got)
	if len(got) != 3 {
		t.Fatalf("setUnion got %v, want 3 elements", got)
	}
	want := []float64{1, 2, 3}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("setUnion element %d = %v, want %v", i, got[i], v)
		}
	}
}

func TestSetDifference(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	_, cur, err := exec.Run(ctx,
		reql.Array(1, 2, 3).SetDifference(reql.Array(2)), nil)
	if err != nil {
		t.Fatalf("setDifference: %v", err)
	}
	raw, err := cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got []float64
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	sort.Float64s(got)
	if len(got) != 2 {
		t.Fatalf("setDifference got %v, want 2 elements", got)
	}
	if got[0] != 1 || got[1] != 3 {
		t.Errorf("setDifference = %v, want [1 3]", got)
	}
}

func TestSetOpsOnTableFields(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	_, cur0, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").Insert(reql.JSON(`{"id":"a","tags":[1,2,3]}`)), nil)
	closeCursor(cur0)
	if err != nil {
		t.Fatalf("insert doc: %v", err)
	}

	// setInsert via map -- add 2 (dup) and 4 (new) to tags
	insertDupFn := reql.Func(
		reql.Datum(map[string]interface{}{
			"dupResult": reql.Var(1).GetField("tags").SetInsert(2),
			"newResult": reql.Var(1).GetField("tags").SetInsert(4),
		}),
		1,
	)
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Map(insertDupFn), nil)
	if err != nil {
		t.Fatalf("map setInsert: %v", err)
	}
	rows, err := cur.All()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(rows[0], &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var dupResult []float64
	if err := json.Unmarshal(doc["dupResult"], &dupResult); err != nil {
		t.Fatalf("unmarshal dupResult: %v", err)
	}
	if len(dupResult) != 3 {
		t.Errorf("setInsert dup on table field: got %d elements, want 3", len(dupResult))
	}
	var newResult []float64
	if err := json.Unmarshal(doc["newResult"], &newResult); err != nil {
		t.Fatalf("unmarshal newResult: %v", err)
	}
	if len(newResult) != 4 {
		t.Errorf("setInsert new on table field: got %d elements, want 4", len(newResult))
	}

	// setUnion -- union with [3,4,5] on [1,2,3] -> [1,2,3,4,5]
	unionFn := reql.Func(
		reql.Var(1).GetField("tags").SetUnion(reql.Array(3, 4, 5)),
		1,
	)
	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Map(unionFn), nil)
	if err != nil {
		t.Fatalf("map setUnion: %v", err)
	}
	raw2, err := cur2.Next()
	closeCursor(cur2)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var union []float64
	if err := json.Unmarshal(raw2, &union); err != nil {
		t.Fatalf("unmarshal union: %v", err)
	}
	sort.Float64s(union)
	if len(union) != 5 {
		t.Errorf("setUnion on table field: got %v, want 5 elements", union)
	}
	// verify no duplicates and all expected values present
	wantUnion := []float64{1, 2, 3, 4, 5}
	for i, v := range wantUnion {
		if union[i] != v {
			t.Errorf("setUnion element %d = %v, want %v", i, union[i], v)
		}
	}
}
