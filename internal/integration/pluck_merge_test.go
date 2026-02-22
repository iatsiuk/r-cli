//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"r-cli/internal/reql"
)

func TestPluck(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "u1", "name": "alice", "age": 30, "email": "alice@example.com"},
		{"id": "u2", "name": "bob", "age": 25, "email": "bob@example.com"},
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Pluck("name"), nil)
	if err != nil {
		t.Fatalf("pluck: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}

	for _, raw := range rows {
		var doc map[string]json.RawMessage
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		// pluck("name") preserves id and name; age and email must be absent
		if _, ok := doc["age"]; ok {
			t.Error("pluck result should not contain 'age'")
		}
		if _, ok := doc["email"]; ok {
			t.Error("pluck result should not contain 'email'")
		}
		if _, ok := doc["name"]; !ok {
			t.Error("pluck result must contain 'name'")
		}
	}
}

func TestWithout(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "u1", "name": "alice", "password": "secret1"},
		{"id": "u2", "name": "bob", "password": "secret2"},
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Without("password"), nil)
	if err != nil {
		t.Fatalf("without: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}

	for _, raw := range rows {
		var doc map[string]json.RawMessage
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if _, ok := doc["password"]; ok {
			t.Error("without result should not contain 'password'")
		}
		if _, ok := doc["name"]; !ok {
			t.Error("without result must contain 'name'")
		}
	}
}

func TestMerge(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "u1", "name": "alice"},
		{"id": "u2", "name": "bob"},
	})

	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").Merge(map[string]interface{}{"status": "active"}), nil)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}

	for _, raw := range rows {
		var doc struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if doc.Status != "active" {
			t.Errorf("merged status=%q, want 'active'", doc.Status)
		}
		if doc.Name == "" {
			t.Error("merge should preserve existing 'name' field")
		}
	}
}

func TestHasFields(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "u1", "name": "alice", "email": "alice@example.com"},
		{"id": "u2", "name": "bob"},
		{"id": "u3", "name": "carol", "email": "carol@example.com"},
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").HasFields("email"), nil)
	if err != nil {
		t.Fatalf("has fields: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("got %d rows with email field, want 2", len(rows))
	}
}
