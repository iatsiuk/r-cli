//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"sort"
	"testing"

	"r-cli/internal/reql"
)

func TestIsEmptyArray(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	_, cur, err := exec.Run(ctx, reql.Array().IsEmpty(), nil)
	if err != nil {
		t.Fatalf("isEmpty empty array: %v", err)
	}
	raw, err := cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got bool
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !got {
		t.Error("isEmpty([]) = false, want true")
	}

	_, cur2, err := exec.Run(ctx, reql.Array(1).IsEmpty(), nil)
	if err != nil {
		t.Fatalf("isEmpty non-empty array: %v", err)
	}
	raw2, err := cur2.Next()
	closeCursor(cur2)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got2 bool
	if err := json.Unmarshal(raw2, &got2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got2 {
		t.Error("isEmpty([1]) = true, want false")
	}
}

func TestIsEmptyTable(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "items")

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("items").IsEmpty(), nil)
	if err != nil {
		t.Fatalf("isEmpty empty table: %v", err)
	}
	raw, err := cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got bool
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !got {
		t.Error("isEmpty(empty table) = false, want true")
	}

	seedTable(t, exec, dbName, "items", []map[string]interface{}{
		{"id": "x", "v": 1},
	})

	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("items").IsEmpty(), nil)
	if err != nil {
		t.Fatalf("isEmpty non-empty table: %v", err)
	}
	raw2, err := cur2.Next()
	closeCursor(cur2)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got2 bool
	if err := json.Unmarshal(raw2, &got2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got2 {
		t.Error("isEmpty(non-empty table) = true, want false")
	}

	// empty filter result
	_, cur3, err := exec.Run(ctx,
		reql.DB(dbName).Table("items").Filter(
			reql.Func(reql.Var(1).GetField("v").Eq(999), 1),
		).IsEmpty(), nil)
	if err != nil {
		t.Fatalf("isEmpty empty filter: %v", err)
	}
	raw3, err := cur3.Next()
	closeCursor(cur3)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got3 bool
	if err := json.Unmarshal(raw3, &got3); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !got3 {
		t.Error("isEmpty(empty filter result) = false, want true")
	}
}

func TestContainsArray(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	_, cur, err := exec.Run(ctx, reql.Array(1, 2, 3).Contains(2), nil)
	if err != nil {
		t.Fatalf("contains 2: %v", err)
	}
	raw, err := cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got bool
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !got {
		t.Error("contains([1,2,3], 2) = false, want true")
	}

	_, cur2, err := exec.Run(ctx, reql.Array(1, 2, 3).Contains(5), nil)
	if err != nil {
		t.Fatalf("contains 5: %v", err)
	}
	raw2, err := cur2.Next()
	closeCursor(cur2)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got2 bool
	if err := json.Unmarshal(raw2, &got2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got2 {
		t.Error("contains([1,2,3], 5) = true, want false")
	}
}

func TestContainsTablePredicate(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "items")
	seedTable(t, exec, dbName, "items", []map[string]interface{}{
		{"id": "a", "val": 10},
		{"id": "b", "val": 20},
	})

	// contains with predicate: any row with val == 10
	pred := reql.Func(reql.Var(1).GetField("val").Eq(10), 1)
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("items").Contains(pred), nil)
	if err != nil {
		t.Fatalf("contains predicate true: %v", err)
	}
	raw, err := cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got bool
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !got {
		t.Error("contains(table, val==10) = false, want true")
	}

	// no row with val == 99
	pred2 := reql.Func(reql.Var(1).GetField("val").Eq(99), 1)
	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("items").Contains(pred2), nil)
	if err != nil {
		t.Fatalf("contains predicate false: %v", err)
	}
	raw2, err := cur2.Next()
	closeCursor(cur2)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got2 bool
	if err := json.Unmarshal(raw2, &got2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got2 {
		t.Error("contains(table, val==99) = true, want false")
	}
}

func TestConcatMapArray(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	// concatMap identity on array of arrays -> flatten; returns ATOM (single array value)
	fn := reql.Func(reql.Var(1), 1)
	_, cur, err := exec.Run(ctx,
		reql.Array(reql.Array(1, 2), reql.Array(3, 4)).ConcatMap(fn), nil)
	if err != nil {
		t.Fatalf("concatMap flatten: %v", err)
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
	if len(got) != 4 {
		t.Fatalf("concatMap got %d elements, want 4: %v", len(got), got)
	}
	want := []float64{1, 2, 3, 4}
	for i, v := range got {
		if v != want[i] {
			t.Errorf("element %d = %v, want %v", i, v, want[i])
		}
	}
}

func TestConcatMapTable(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	// use reql.JSON for docs with array fields to avoid slice misinterpretation
	_, cur0, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").Insert(reql.Array(
			reql.JSON(`{"id":"a","tags":["x","y"]}`),
			reql.JSON(`{"id":"b","tags":["z"]}`),
		)), nil)
	closeCursor(cur0)
	if err != nil {
		t.Fatalf("insert docs with tags: %v", err)
	}

	// concatMap to extract tags from each doc
	fn := reql.Func(reql.Var(1).GetField("tags"), 1)
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").ConcatMap(fn), nil)
	if err != nil {
		t.Fatalf("concatMap table: %v", err)
	}
	rows, err := cur.All()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	// 2 tags from doc a + 1 from doc b = 3 total
	if len(rows) != 3 {
		t.Fatalf("concatMap table got %d elements, want 3", len(rows))
	}
}

func TestUnionArrays(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	// union of two in-memory arrays returns ATOM (single array value)
	_, cur, err := exec.Run(ctx,
		reql.Array(1, 2).Union(reql.Array(3, 4)), nil)
	if err != nil {
		t.Fatalf("union arrays: %v", err)
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
	if len(got) != 4 {
		t.Fatalf("union got %d elements, want 4: %v", len(got), got)
	}
	want := []float64{1, 2, 3, 4}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("element %d = %v, want %v", i, got[i], w)
		}
	}
}

func TestUnionTables(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "t1")
	createTestTable(t, exec, dbName, "t2")
	seedTable(t, exec, dbName, "t1", []map[string]interface{}{
		{"id": "a"}, {"id": "b"},
	})
	seedTable(t, exec, dbName, "t2", []map[string]interface{}{
		{"id": "c"}, {"id": "d"}, {"id": "e"},
	})

	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("t1").Union(reql.DB(dbName).Table("t2")), nil)
	if err != nil {
		t.Fatalf("union tables: %v", err)
	}
	rows, err := cur.All()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 5 {
		t.Fatalf("union tables got %d rows, want 5", len(rows))
	}
}

func TestWithFields(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	// doc a has both name and email; doc b has only name; doc c has both
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "a", "name": "alice", "email": "alice@example.com"},
		{"id": "b", "name": "bob"},
		{"id": "c", "name": "carol", "email": "carol@example.com"},
	})

	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").WithFields("name", "email"), nil)
	if err != nil {
		t.Fatalf("withFields: %v", err)
	}
	rows, err := cur.All()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	// only a and c have both fields
	if len(rows) != 2 {
		t.Fatalf("withFields got %d rows, want 2", len(rows))
	}
	for _, raw := range rows {
		var doc map[string]interface{}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal doc: %v", err)
		}
		if _, ok := doc["name"]; !ok {
			t.Errorf("withFields doc missing 'name': %v", doc)
		}
		if _, ok := doc["email"]; !ok {
			t.Errorf("withFields doc missing 'email': %v", doc)
		}
	}
}

func TestKeys(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	// keys() on an object returns ATOM (a single array value)
	_, cur, err := exec.Run(ctx,
		reql.Datum(map[string]interface{}{"a": 1, "b": 2}).Keys(), nil)
	if err != nil {
		t.Fatalf("keys: %v", err)
	}
	raw, err := cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got []string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal keys: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("keys got %d elements, want 2: %v", len(got), got)
	}
	sort.Strings(got)
	if got[0] != "a" || got[1] != "b" {
		t.Errorf("keys = %v, want [a b]", got)
	}
}

func TestKeysOnTableRow(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "x", "name": "alice", "score": 10},
	})

	fn := reql.Func(reql.Var(1).Keys(), 1)
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Map(fn), nil)
	if err != nil {
		t.Fatalf("keys on table row via map: %v", err)
	}
	rows, err := cur.All()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	var keys []string
	if err := json.Unmarshal(rows[0], &keys); err != nil {
		t.Fatalf("unmarshal keys: %v", err)
	}
	sort.Strings(keys)
	if len(keys) != 3 {
		t.Errorf("keys count = %d, want 3: %v", len(keys), keys)
	}
}

func TestValues(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	// values() on an object returns ATOM (a single array value)
	_, cur, err := exec.Run(ctx,
		reql.Datum(map[string]interface{}{"a": 1, "b": 2}).Values(), nil)
	if err != nil {
		t.Fatalf("values: %v", err)
	}
	raw, err := cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got []float64
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal values: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("values got %d elements, want 2: %v", len(got), got)
	}
	sort.Float64s(got)
	if got[0] != 1 || got[1] != 2 {
		t.Errorf("values = %v, want [1 2]", got)
	}
}

func TestValuesOnTableRow(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "x", "score": 42},
	})

	// get() returns ATOM; values() on an ATOM object also returns ATOM (the array)
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Get("x").Values(), nil)
	if err != nil {
		t.Fatalf("values on table row: %v", err)
	}
	raw, err := cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var vals []interface{}
	if err := json.Unmarshal(raw, &vals); err != nil {
		t.Fatalf("unmarshal values: %v", err)
	}
	// doc has 2 fields: id and score
	if len(vals) != 2 {
		t.Fatalf("values got %d elements, want 2: %v", len(vals), vals)
	}
}
