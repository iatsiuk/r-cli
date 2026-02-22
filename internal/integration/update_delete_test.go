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

func TestUpdateSingle(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "u1", "name": "alice", "score": 10},
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Get("u1").Update(map[string]interface{}{"score": 99}), nil)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	r := parseWriteResult(t, cur)

	if r.Replaced != 1 {
		t.Errorf("replaced=%d, want 1", r.Replaced)
	}

	// verify field changed
	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Get("u1"), nil)
	if err != nil {
		t.Fatalf("get after update: %v", err)
	}
	defer closeCursor(cur2)

	raw, err := cur2.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var doc struct {
		Score int `json:"score"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if doc.Score != 99 {
		t.Errorf("score=%d after update, want 99", doc.Score)
	}
}

func TestUpdateAll(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "v": 1},
		{"id": "2", "v": 2},
		{"id": "3", "v": 3},
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Update(map[string]interface{}{"v": 0}), nil)
	if err != nil {
		t.Fatalf("update all: %v", err)
	}
	r := parseWriteResult(t, cur)

	if r.Replaced != 3 {
		t.Errorf("replaced=%d, want 3", r.Replaced)
	}
}

func TestUpdateMerge(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "u1", "name": "alice"},
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Get("u1").Update(map[string]interface{}{"email": "alice@example.com"}), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Get("u1"), nil)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer closeCursor(cur2)

	raw, err := cur2.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := doc["email"]; !ok {
		t.Error("email field missing after merge update")
	}
	if _, ok := doc["name"]; !ok {
		t.Error("existing name field missing after merge update")
	}
}

func TestUpdateNonexistent(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Get("no-such").Update(map[string]interface{}{"x": 1}), nil)
	if err != nil {
		t.Fatalf("update nonexistent: %v", err)
	}
	r := parseWriteResult(t, cur)

	if r.Skipped != 1 {
		t.Errorf("skipped=%d, want 1", r.Skipped)
	}
}

func TestUpdateReturnChanges(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "u1", "v": 1},
	})

	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").Get("u1").Update(
			map[string]interface{}{"v": 2},
			reql.OptArgs{"return_changes": true},
		), nil)
	if err != nil {
		t.Fatalf("update with return_changes: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}

	var result struct {
		Replaced int `json:"replaced"`
		Changes  []struct {
			OldVal map[string]interface{} `json:"old_val"`
			NewVal map[string]interface{} `json:"new_val"`
		} `json:"changes"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Replaced != 1 {
		t.Errorf("replaced=%d, want 1", result.Replaced)
	}
	if len(result.Changes) != 1 {
		t.Fatalf("changes len=%d, want 1", len(result.Changes))
	}
	if result.Changes[0].OldVal == nil {
		t.Error("old_val is nil")
	}
	if result.Changes[0].NewVal == nil {
		t.Error("new_val is nil")
	}
}

func TestReplaceByGet(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "u1", "name": "alice", "extra": "field"},
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Get("u1").Replace(map[string]interface{}{"id": "u1", "name": "bob"}), nil)
	if err != nil {
		t.Fatalf("replace: %v", err)
	}
	r := parseWriteResult(t, cur)

	if r.Replaced != 1 {
		t.Errorf("replaced=%d, want 1", r.Replaced)
	}

	// verify old fields are gone
	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Get("u1"), nil)
	if err != nil {
		t.Fatalf("get after replace: %v", err)
	}
	defer closeCursor(cur2)

	raw, err := cur2.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := doc["extra"]; ok {
		t.Error("old 'extra' field still present after replace")
	}
	if doc["name"] != "bob" {
		t.Errorf("name=%v, want bob", doc["name"])
	}
}

func TestReplaceMissingPrimaryKey(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "u1", "name": "alice"},
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Get("u1").Replace(map[string]interface{}{"name": "bob"}), nil)
	// RethinkDB reports this as a write error (errors=1 in the result) rather than
	// a runtime error response, so check both paths.
	if err != nil {
		var runtimeErr *response.ReqlRuntimeError
		if !errors.As(err, &runtimeErr) {
			t.Errorf("expected ReqlRuntimeError, got %T: %v", err, err)
		}
		closeCursor(cur)
		return
	}
	r := parseWriteResult(t, cur)
	if r.Errors == 0 {
		t.Error("expected errors>0 for replace without primary key")
	}
}

func TestDeleteSingle(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "u1"},
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Get("u1").Delete(), nil)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	r := parseWriteResult(t, cur)

	if r.Deleted != 1 {
		t.Errorf("deleted=%d, want 1", r.Deleted)
	}
}

func TestDeleteWithFilter(t *testing.T) {
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
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Filter(map[string]interface{}{"active": true}).Delete(), nil)
	if err != nil {
		t.Fatalf("delete with filter: %v", err)
	}
	r := parseWriteResult(t, cur)

	if r.Deleted != 2 {
		t.Errorf("deleted=%d, want 2", r.Deleted)
	}
}

func TestDeleteAll(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1"},
		{"id": "2"},
		{"id": "3"},
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Delete(), nil)
	if err != nil {
		t.Fatalf("delete all: %v", err)
	}
	r := parseWriteResult(t, cur)

	if r.Deleted != 3 {
		t.Errorf("deleted=%d, want 3", r.Deleted)
	}
}

func TestDeleteNonexistent(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Get("no-such").Delete(), nil)
	if err != nil {
		t.Fatalf("delete nonexistent: %v", err)
	}
	r := parseWriteResult(t, cur)

	if r.Deleted != 0 {
		t.Errorf("deleted=%d, want 0", r.Deleted)
	}
}
