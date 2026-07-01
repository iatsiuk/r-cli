//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"r-cli/internal/connmgr"
	"r-cli/internal/query"
	"r-cli/internal/reql"
)

// newReadOnlyExecutor creates an Executor with read-only mode enabled, backed by
// the shared test container. Mirrors newExecutor(t).
func newReadOnlyExecutor(t *testing.T) *query.Executor {
	t.Helper()
	mgr := connmgr.NewFromConfig(defaultCfg(), nil)
	t.Cleanup(func() { _ = mgr.Close() })
	return query.New(mgr, query.WithReadOnly(true))
}

// assertReadOnly runs term via the read-only executor and asserts it is rejected
// with query.ErrReadOnly and no cursor.
func assertReadOnly(t *testing.T, ro *query.Executor, name string, term reql.Term) {
	t.Helper()
	_, cur, err := ro.Run(context.Background(), term, nil)
	closeCursor(cur)
	if !errors.Is(err, query.ErrReadOnly) {
		t.Errorf("%s: got err %v, want ErrReadOnly", name, err)
	}
	if cur != nil {
		t.Errorf("%s: got non-nil cursor, want nil", name)
	}
}

func TestReadOnlyRejectsDocumentWrites(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ro := newReadOnlyExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "u1", "name": "alice"},
	})

	writes := []struct {
		name string
		term reql.Term
	}{
		{"insert", reql.DB(dbName).Table("docs").Insert(map[string]interface{}{"id": "u2", "name": "bob"})},
		{"update", reql.DB(dbName).Table("docs").Get("u1").Update(map[string]interface{}{"name": "changed"})},
		{"replace", reql.DB(dbName).Table("docs").Get("u1").Replace(map[string]interface{}{"id": "u1", "name": "replaced"})},
		{"delete", reql.DB(dbName).Table("docs").Get("u1").Delete()},
	}
	for _, w := range writes {
		assertReadOnly(t, ro, w.name, w.term)
	}

	// re-read with a normal executor: seeded data must be unchanged
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Get("u1"), nil)
	if err != nil {
		t.Fatalf("re-read get: %v", err)
	}
	raw, err := cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("re-read cursor next: %v", err)
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

	_, cur, err = exec.Run(ctx, reql.DB(dbName).Table("docs").Count(), nil)
	if err != nil {
		t.Fatalf("re-read count: %v", err)
	}
	raw, err = cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("count cursor next: %v", err)
	}
	var count int
	if err := json.Unmarshal(raw, &count); err != nil {
		t.Fatalf("unmarshal count: %v", err)
	}
	if count != 1 {
		t.Errorf("row count=%d, want 1 (no rows added or removed)", count)
	}
}

func TestReadOnlyRejectsDDL(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ro := newReadOnlyExecutor(t)

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	ddl := []struct {
		name string
		term reql.Term
	}{
		{"dbCreate", reql.DBCreate(dbName + "_ro")},
		{"tableCreate", reql.DB(dbName).TableCreate("newtbl")},
		{"tableDrop", reql.DB(dbName).TableDrop("docs")},
		{"indexCreate", reql.DB(dbName).Table("docs").IndexCreate("name")},
	}
	for _, d := range ddl {
		assertReadOnly(t, ro, d.name, d.term)
	}
}

func TestReadOnlyRejectsAdminWrites(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ro := newReadOnlyExecutor(t)

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	admin := []struct {
		name string
		term reql.Term
	}{
		{"reconfigure", reql.DB(dbName).Table("docs").Reconfigure(reql.OptArgs{"shards": 1, "replicas": 1})},
		{"rebalance", reql.DB(dbName).Table("docs").Rebalance()},
		{"sync", reql.DB(dbName).Table("docs").Sync()},
		{"grant", reql.Grant("ro_grant_user", map[string]interface{}{"read": true})},
	}
	for _, a := range admin {
		assertReadOnly(t, ro, a.name, a.term)
	}
}

func TestReadOnlyAllowsReads(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ro := newReadOnlyExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "u1", "name": "alice"},
		{"id": "u2", "name": "bob"},
	})

	// table list
	_, cur, err := ro.Run(ctx, reql.DB(dbName).TableList(), nil)
	if err != nil {
		t.Fatalf("table list: %v", err)
	}
	raw, err := cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("table list next: %v", err)
	}
	var tables []string
	if err := json.Unmarshal(raw, &tables); err != nil {
		t.Fatalf("unmarshal table list: %v", err)
	}
	if !strContains(tables, "docs") {
		t.Errorf("table list %v, want to contain docs", tables)
	}

	// get
	_, cur, err = ro.Run(ctx, reql.DB(dbName).Table("docs").Get("u1"), nil)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	raw, err = cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("get next: %v", err)
	}
	var doc struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal get: %v", err)
	}
	if doc.ID != "u1" || doc.Name != "alice" {
		t.Errorf("got %+v, want {id:u1 name:alice}", doc)
	}

	// filter
	_, cur, err = ro.Run(ctx, reql.DB(dbName).Table("docs").Filter(map[string]interface{}{"name": "bob"}), nil)
	if err != nil {
		t.Fatalf("filter: %v", err)
	}
	defer closeCursor(cur)
	rows, err := cur.All()
	if err != nil {
		t.Fatalf("filter all: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("filter returned %d rows, want 1", len(rows))
	}
}
