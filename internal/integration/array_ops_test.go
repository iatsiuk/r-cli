//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"r-cli/internal/reql"
)

func TestAppend(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	_, cur, err := exec.Run(ctx, reql.Array(1, 2).Append(3), nil)
	if err != nil {
		t.Fatalf("append: %v", err)
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
	want := []float64{1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("append got %v, want %v", got, want)
	}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("element %d = %v, want %v", i, got[i], v)
		}
	}
}

func TestPrepend(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	_, cur, err := exec.Run(ctx, reql.Array(2, 3).Prepend(1), nil)
	if err != nil {
		t.Fatalf("prepend: %v", err)
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
	want := []float64{1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("prepend got %v, want %v", got, want)
	}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("element %d = %v, want %v", i, got[i], v)
		}
	}
}

func TestSliceTwoArgs(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	_, cur, err := exec.Run(ctx, reql.Array(0, 1, 2, 3, 4).Slice(1, 3), nil)
	if err != nil {
		t.Fatalf("slice(1,3): %v", err)
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
	want := []float64{1, 2}
	if len(got) != len(want) {
		t.Fatalf("slice(1,3) got %v, want %v", got, want)
	}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("element %d = %v, want %v", i, got[i], v)
		}
	}
}

func TestSliceOneArg(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	_, cur, err := exec.Run(ctx, reql.Array(0, 1, 2, 3, 4).Slice(2, 5), nil)
	if err != nil {
		t.Fatalf("slice(2): %v", err)
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
	want := []float64{2, 3, 4}
	if len(got) != len(want) {
		t.Fatalf("slice(2) got %v, want %v", got, want)
	}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("element %d = %v, want %v", i, got[i], v)
		}
	}
}

func TestDifference(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	_, cur, err := exec.Run(ctx, reql.Array(1, 2, 3, 2).Difference(reql.Array(2)), nil)
	if err != nil {
		t.Fatalf("difference: %v", err)
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
	want := []float64{1, 3}
	if len(got) != len(want) {
		t.Fatalf("difference got %v, want %v", got, want)
	}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("element %d = %v, want %v", i, got[i], v)
		}
	}
}

func TestInsertAt(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	_, cur, err := exec.Run(ctx, reql.Array(0, 1, 3).InsertAt(2, 2), nil)
	if err != nil {
		t.Fatalf("insertAt: %v", err)
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
	want := []float64{0, 1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("insertAt got %v, want %v", got, want)
	}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("element %d = %v, want %v", i, got[i], v)
		}
	}
}

func TestDeleteAt(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	_, cur, err := exec.Run(ctx, reql.Array(0, 1, 2, 3).DeleteAt(1), nil)
	if err != nil {
		t.Fatalf("deleteAt: %v", err)
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
	want := []float64{0, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("deleteAt got %v, want %v", got, want)
	}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("element %d = %v, want %v", i, got[i], v)
		}
	}
}

func TestChangeAt(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	_, cur, err := exec.Run(ctx, reql.Array(0, 1, 2).ChangeAt(1, 9), nil)
	if err != nil {
		t.Fatalf("changeAt: %v", err)
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
	want := []float64{0, 9, 2}
	if len(got) != len(want) {
		t.Fatalf("changeAt got %v, want %v", got, want)
	}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("element %d = %v, want %v", i, got[i], v)
		}
	}
}

func TestSpliceAt(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	_, cur, err := exec.Run(ctx, reql.Array(0, 3, 4).SpliceAt(1, reql.Array(1, 2)), nil)
	if err != nil {
		t.Fatalf("spliceAt: %v", err)
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
	want := []float64{0, 1, 2, 3, 4}
	if len(got) != len(want) {
		t.Fatalf("spliceAt got %v, want %v", got, want)
	}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("element %d = %v, want %v", i, got[i], v)
		}
	}
}

func TestArrayOpsOnTableFields(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	_, cur0, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").Insert(reql.JSON(`{"id":"a","tags":["x","y"]}`)), nil)
	closeCursor(cur0)
	if err != nil {
		t.Fatalf("insert doc: %v", err)
	}

	// append "z" to tags field
	updateFn := reql.Func(
		reql.Datum(map[string]interface{}{"tags": reql.Var(1).GetField("tags").Append("z")}),
		1,
	)
	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").Map(updateFn), nil)
	if err != nil {
		t.Fatalf("map append: %v", err)
	}
	rows, err := cur.All()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(rows[0], &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	tags, ok := doc["tags"].([]interface{})
	if !ok {
		t.Fatalf("tags not array: %v", doc["tags"])
	}
	if len(tags) != 3 {
		t.Errorf("after append tags has %d elements, want 3: %v", len(tags), tags)
	}

	// prepend "w" to tags field
	prependFn := reql.Func(
		reql.Datum(map[string]interface{}{"tags": reql.Var(1).GetField("tags").Prepend("w")}),
		1,
	)
	_, cur2, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").Map(prependFn), nil)
	if err != nil {
		t.Fatalf("map prepend: %v", err)
	}
	rows2, err := cur2.All()
	closeCursor(cur2)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows2) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows2))
	}
	var doc2 map[string]interface{}
	if err := json.Unmarshal(rows2[0], &doc2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	tags2, ok := doc2["tags"].([]interface{})
	if !ok {
		t.Fatalf("tags not array: %v", doc2["tags"])
	}
	if len(tags2) != 3 {
		t.Errorf("after prepend tags has %d elements, want 3: %v", len(tags2), tags2)
	}
	if tags2[0] != "w" {
		t.Errorf("prepend: first element = %v, want w", tags2[0])
	}
}
