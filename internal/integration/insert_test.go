//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"r-cli/internal/cursor"
	"r-cli/internal/reql"
)

// writeResult holds fields from a RethinkDB write operation response.
type writeResult struct {
	Inserted      int      `json:"inserted"`
	Replaced      int      `json:"replaced"`
	Deleted       int      `json:"deleted"`
	Skipped       int      `json:"skipped"`
	Unchanged     int      `json:"unchanged"`
	Errors        int      `json:"errors"`
	GeneratedKeys []string `json:"generated_keys"`
}

// parseWriteResult reads the next cursor item and unmarshals it as writeResult.
func parseWriteResult(t *testing.T, cur cursor.Cursor) writeResult {
	t.Helper()
	defer closeCursor(cur)
	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var r writeResult
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatalf("unmarshal insert result: %v", err)
	}
	return r
}

func TestInsertSingle(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Insert(map[string]interface{}{
		"name": "alice",
	}), nil)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	r := parseWriteResult(t, cur)

	if r.Inserted != 1 {
		t.Errorf("inserted=%d, want 1", r.Inserted)
	}
	if len(r.GeneratedKeys) != 1 {
		t.Errorf("generated_keys len=%d, want 1", len(r.GeneratedKeys))
	}
	if len(r.GeneratedKeys) > 0 && !uuidRe.MatchString(r.GeneratedKeys[0]) {
		t.Errorf("generated key %q is not a valid UUID", r.GeneratedKeys[0])
	}
}

func TestInsertExplicitID(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Insert(map[string]interface{}{
		"id":   "user-1",
		"name": "bob",
	}), nil)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	r := parseWriteResult(t, cur)

	if r.Inserted != 1 {
		t.Errorf("inserted=%d, want 1", r.Inserted)
	}
	if len(r.GeneratedKeys) != 0 {
		t.Errorf("generated_keys should be absent for explicit id, got %v", r.GeneratedKeys)
	}
}

func TestInsertDuplicateID(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	doc := map[string]interface{}{"id": "dup-1", "v": 1}

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Insert(doc), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}

	// second insert with same id - should return errors=1 (conflict response)
	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Insert(doc), nil)
	if err != nil {
		t.Fatalf("second insert: %v", err)
	}
	r := parseWriteResult(t, cur2)

	if r.Errors != 1 {
		t.Errorf("errors=%d, want 1 for duplicate id", r.Errors)
	}
}

func TestInsertConflictReplace(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	doc := map[string]interface{}{"id": "c-1", "v": 1}

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Insert(doc), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}

	doc["v"] = 2
	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Insert(doc, reql.OptArgs{"conflict": "replace"}), nil)
	if err != nil {
		t.Fatalf("insert with conflict=replace: %v", err)
	}
	r := parseWriteResult(t, cur2)

	if r.Replaced != 1 {
		t.Errorf("replaced=%d, want 1", r.Replaced)
	}
}

func TestInsertConflictUpdate(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	doc := map[string]interface{}{"id": "c-2", "v": 1}

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Insert(doc), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}

	// insert same doc again with conflict=update - same data -> unchanged
	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Insert(doc, reql.OptArgs{"conflict": "update"}), nil)
	if err != nil {
		t.Fatalf("insert with conflict=update: %v", err)
	}
	r := parseWriteResult(t, cur2)

	if r.Unchanged+r.Replaced != 1 {
		t.Errorf("unchanged=%d replaced=%d, want unchanged+replaced=1", r.Unchanged, r.Replaced)
	}
}

func TestInsertBulk(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	docs := make([]interface{}, 100)
	for i := range docs {
		docs[i] = map[string]interface{}{"n": i}
	}

	// use reql.Array to create MAKE_ARRAY term; raw []interface{} would be
	// misinterpreted by the server as a ReQL term array.
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Insert(reql.Array(docs...)), nil)
	if err != nil {
		t.Fatalf("bulk insert: %v", err)
	}
	r := parseWriteResult(t, cur)

	if r.Inserted != 100 {
		t.Errorf("inserted=%d, want 100", r.Inserted)
	}
	if len(r.GeneratedKeys) != 100 {
		t.Errorf("generated_keys len=%d, want 100", len(r.GeneratedKeys))
	}
}

func TestInsertEmptyObject(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Insert(map[string]interface{}{}), nil)
	if err != nil {
		t.Fatalf("insert empty object: %v", err)
	}
	r := parseWriteResult(t, cur)

	if r.Inserted != 1 {
		t.Errorf("inserted=%d, want 1", r.Inserted)
	}
	if len(r.GeneratedKeys) != 1 {
		t.Errorf("generated_keys len=%d, want 1 for auto-generated id", len(r.GeneratedKeys))
	}
}

func TestInsertNestedRoundtrip(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	// use reql.JSON to pass the document as a JSON string; raw Go slices would
	// be misinterpreted by the server as ReQL term arrays when embedded in a datum.
	doc := reql.JSON(`{"id":"nested-1","meta":{"score":42,"active":true},"tags":["a","b","c"]}`)

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Insert(doc), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Get("nested-1"), nil)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer closeCursor(cur2)

	raw, err := cur2.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}

	var got struct {
		ID   string `json:"id"`
		Meta struct {
			Score  int  `json:"score"`
			Active bool `json:"active"`
		} `json:"meta"`
		Tags []string `json:"tags"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.ID != "nested-1" {
		t.Errorf("id=%q, want %q", got.ID, "nested-1")
	}
	if got.Meta.Score != 42 {
		t.Errorf("meta.score=%d, want 42", got.Meta.Score)
	}
	if !got.Meta.Active {
		t.Error("meta.active should be true")
	}
	if len(got.Tags) != 3 || got.Tags[0] != "a" || got.Tags[2] != "c" {
		t.Errorf("tags=%v, want [a b c]", got.Tags)
	}
}
