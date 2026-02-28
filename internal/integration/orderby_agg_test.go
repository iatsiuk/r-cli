//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"r-cli/internal/cursor"
	"r-cli/internal/reql"
)

// atomRows reads a SUCCESS_ATOM cursor that contains a JSON array and returns
// each element. orderBy/skip/distinct without an index return SUCCESS_ATOM.
func atomRows(t *testing.T, cur cursor.Cursor) []json.RawMessage {
	t.Helper()
	defer closeCursor(cur)
	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatalf("unmarshal atom array: %v", err)
	}
	return rows
}

func TestOrderByAscending(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "c", "score": 30},
		{"id": "a", "score": 10},
		{"id": "b", "score": 20},
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").OrderBy(reql.Asc("score")), nil)
	if err != nil {
		t.Fatalf("order by asc: %v", err)
	}

	rows := atomRows(t, cur)
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}

	var scores [3]int
	for i, raw := range rows {
		var doc struct {
			Score int `json:"score"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal row %d: %v", i, err)
		}
		scores[i] = doc.Score
	}
	if scores[0] != 10 || scores[1] != 20 || scores[2] != 30 {
		t.Errorf("scores not ascending: %v", scores)
	}
}

func TestOrderByDescending(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "c", "score": 30},
		{"id": "a", "score": 10},
		{"id": "b", "score": 20},
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").OrderBy(reql.Desc("score")), nil)
	if err != nil {
		t.Fatalf("order by desc: %v", err)
	}

	rows := atomRows(t, cur)
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}

	var scores [3]int
	for i, raw := range rows {
		var doc struct {
			Score int `json:"score"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal row %d: %v", i, err)
		}
		scores[i] = doc.Score
	}
	if scores[0] != 30 || scores[1] != 20 || scores[2] != 10 {
		t.Errorf("scores not descending: %v", scores)
	}
}

func TestLimit(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	docs := make([]map[string]interface{}, 20)
	for i := range docs {
		docs[i] = map[string]interface{}{"id": i, "n": i}
	}
	seedTable(t, exec, dbName, "docs", docs)

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Limit(5), nil)
	if err != nil {
		t.Fatalf("limit: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 5 {
		t.Errorf("got %d rows, want 5", len(rows))
	}
}

func TestSkip(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	docs := make([]map[string]interface{}, 20)
	for i := range docs {
		docs[i] = map[string]interface{}{"id": i, "n": i}
	}
	seedTable(t, exec, dbName, "docs", docs)

	// orderBy without index returns SUCCESS_ATOM; skip() on table stream
	// also becomes an atom after orderBy materializes it.
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").OrderBy(reql.Asc("n")).Skip(10), nil)
	if err != nil {
		t.Fatalf("skip: %v", err)
	}

	rows := atomRows(t, cur)
	if len(rows) != 10 {
		t.Errorf("got %d rows after skip(10) on 20 docs, want 10", len(rows))
	}
}

func TestSkipLimit(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	docs := make([]map[string]interface{}, 20)
	for i := range docs {
		docs[i] = map[string]interface{}{"id": i, "n": i}
	}
	seedTable(t, exec, dbName, "docs", docs)

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").OrderBy(reql.Asc("n")).Skip(5).Limit(5), nil)
	if err != nil {
		t.Fatalf("skip+limit: %v", err)
	}

	rows := atomRows(t, cur)
	if len(rows) != 5 {
		t.Fatalf("got %d rows, want 5", len(rows))
	}

	// docs are ordered 0..19; skip(5).limit(5) should return docs with n=5..9
	for i, raw := range rows {
		var doc struct {
			N int `json:"n"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal row %d: %v", i, err)
		}
		if doc.N != 5+i {
			t.Errorf("row[%d].n=%d, want %d", i, doc.N, 5+i)
		}
	}
}

func TestCountFiltered(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "active": true},
		{"id": "2", "active": false},
		{"id": "3", "active": true},
		{"id": "4", "active": true},
	})

	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").Filter(map[string]interface{}{"active": true}).Count(), nil)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var n int
	if err := json.Unmarshal(raw, &n); err != nil {
		t.Fatalf("unmarshal count: %v", err)
	}
	if n != 3 {
		t.Errorf("count=%d, want 3", n)
	}
}

func TestDistinct(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "color": "red"},
		{"id": "2", "color": "blue"},
		{"id": "3", "color": "red"},
		{"id": "4", "color": "green"},
		{"id": "5", "color": "blue"},
	})

	// r.map(r.row.getField("color")).distinct() - use explicit FUNC to extract color
	colorFn := reql.Func(reql.Var(1).GetField("color"), 1)
	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").Map(colorFn).Distinct(), nil)
	if err != nil {
		t.Fatalf("distinct: %v", err)
	}

	rows := atomRows(t, cur)
	if len(rows) != 3 {
		t.Errorf("distinct colors=%d, want 3 (red, blue, green)", len(rows))
	}
}

func TestSum(t *testing.T) {
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

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Sum("score"), nil)
	if err != nil {
		t.Fatalf("sum: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var total float64
	if err := json.Unmarshal(raw, &total); err != nil {
		t.Fatalf("unmarshal sum: %v", err)
	}
	if total != 60 {
		t.Errorf("sum=%v, want 60", total)
	}
}

func TestAvg(t *testing.T) {
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

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Avg("score"), nil)
	if err != nil {
		t.Fatalf("avg: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var avg float64
	if err := json.Unmarshal(raw, &avg); err != nil {
		t.Fatalf("unmarshal avg: %v", err)
	}
	if avg != 20 {
		t.Errorf("avg=%v, want 20", avg)
	}
}

func TestMin(t *testing.T) {
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

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Min("score"), nil)
	if err != nil {
		t.Fatalf("min: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var doc struct {
		Score int `json:"score"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal min doc: %v", err)
	}
	if doc.Score != 10 {
		t.Errorf("min score=%d, want 10", doc.Score)
	}
}

func TestMax(t *testing.T) {
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

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Max("score"), nil)
	if err != nil {
		t.Fatalf("max: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var doc struct {
		Score int `json:"score"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal max doc: %v", err)
	}
	if doc.Score != 30 {
		t.Errorf("max score=%d, want 30", doc.Score)
	}
}
