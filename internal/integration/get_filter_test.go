//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"r-cli/internal/query"
	"r-cli/internal/reql"
)

// seedTable inserts documents into dbName.tableName.
func seedTable(t *testing.T, exec *query.Executor, dbName, tableName string, docs []map[string]interface{}) {
	t.Helper()
	ctx := context.Background()
	args := make([]interface{}, len(docs))
	for i, d := range docs {
		args[i] = d
	}
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table(tableName).Insert(reql.Array(args...)), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("seed table: %v", err)
	}
}

func TestGetExisting(t *testing.T) {
	t.Parallel()
	exec, cleanup := newExecutor()
	defer cleanup()

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "u1", "name": "alice"},
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Get("u1"), nil)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}

	var doc struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if doc.ID != "u1" || doc.Name != "alice" {
		t.Errorf("got %+v, want {id:u1 name:alice}", doc)
	}
}

func TestGetNonexistent(t *testing.T) {
	t.Parallel()
	exec, cleanup := newExecutor()
	defer cleanup()

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Get("no-such-key"), nil)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	if string(raw) != "null" {
		t.Errorf("expected null for missing key, got %s", raw)
	}
}

func TestGetAll(t *testing.T) {
	t.Parallel()
	exec, cleanup := newExecutor()
	defer cleanup()

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "a", "v": 1},
		{"id": "b", "v": 2},
		{"id": "c", "v": 3},
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").GetAll("a", "c"), nil)
	if err != nil {
		t.Fatalf("get all: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("got %d rows, want 2", len(rows))
	}
}

func TestGetAllSecondaryIndex(t *testing.T) {
	t.Parallel()
	exec, cleanup := newExecutor()
	defer cleanup()

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "dept": "eng"},
		{"id": "2", "dept": "eng"},
		{"id": "3", "dept": "hr"},
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").IndexCreate("dept"), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("index create: %v", err)
	}

	_, wCur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").IndexWait("dept"), nil)
	closeCursor(wCur)
	if err != nil {
		t.Fatalf("index wait: %v", err)
	}

	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("docs").GetAll("eng", reql.OptArgs{"index": "dept"}), nil)
	if err != nil {
		t.Fatalf("get all by index: %v", err)
	}
	defer closeCursor(cur2)

	rows, err := cur2.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("got %d rows for dept=eng, want 2", len(rows))
	}
}

func TestGetAllNoMatches(t *testing.T) {
	t.Parallel()
	exec, cleanup := newExecutor()
	defer cleanup()

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").GetAll("missing"), nil)
	if err != nil {
		t.Fatalf("get all: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("got %d rows, want 0", len(rows))
	}
}

func TestFilterExact(t *testing.T) {
	t.Parallel()
	exec, cleanup := newExecutor()
	defer cleanup()

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "role": "admin"},
		{"id": "2", "role": "user"},
		{"id": "3", "role": "admin"},
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Filter(map[string]interface{}{"role": "admin"}), nil)
	if err != nil {
		t.Fatalf("filter: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("got %d rows for role=admin, want 2", len(rows))
	}
}

func TestFilterGT(t *testing.T) {
	t.Parallel()
	exec, cleanup := newExecutor()
	defer cleanup()

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "score": 10},
		{"id": "2", "score": 50},
		{"id": "3", "score": 80},
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Filter(
		reql.Row().GetField("score").Gt(40),
	), nil)
	if err != nil {
		t.Fatalf("filter: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("got %d rows for score>40, want 2", len(rows))
	}
}

func TestFilterAnd(t *testing.T) {
	t.Parallel()
	exec, cleanup := newExecutor()
	defer cleanup()

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "age": 30, "active": true},
		{"id": "2", "age": 20, "active": true},
		{"id": "3", "age": 35, "active": false},
	})

	pred := reql.Row().GetField("age").Gt(25).And(reql.Row().GetField("active").Eq(true))
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Filter(pred), nil)
	if err != nil {
		t.Fatalf("filter: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("got %d rows for age>25 AND active=true, want 1", len(rows))
	}
}

func TestFilterEmpty(t *testing.T) {
	t.Parallel()
	exec, cleanup := newExecutor()
	defer cleanup()

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "v": 1},
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Filter(
		reql.Row().GetField("v").Gt(9999),
	), nil)
	if err != nil {
		t.Fatalf("filter: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("got %d rows, want 0", len(rows))
	}
}

func TestFilterNested(t *testing.T) {
	t.Parallel()
	exec, cleanup := newExecutor()
	defer cleanup()

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "info": map[string]interface{}{"level": 5}},
		{"id": "2", "info": map[string]interface{}{"level": 10}},
		{"id": "3", "info": map[string]interface{}{"level": 3}},
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Filter(
		reql.Row().GetField("info").GetField("level").Gt(4),
	), nil)
	if err != nil {
		t.Fatalf("filter: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("got %d rows for info.level>4, want 2", len(rows))
	}
}
