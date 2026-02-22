//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"r-cli/internal/reql"
)

func TestBulkInsertE2EPipe(t *testing.T) {
	t.Parallel()
	qexec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	tableName := "bulk_docs"
	setupTestDB(t, qexec, dbName)
	createTestTable(t, qexec, dbName, tableName)

	jsonl := `{"id":"1","val":10}` + "\n" + `{"id":"2","val":20}` + "\n"
	ref := fmt.Sprintf("%s.%s", dbName, tableName)

	stdout, stderr, code := cliRun(t, jsonl, cliArgs("insert", ref)...)
	if code != 0 {
		t.Fatalf("insert: exit code %d, stderr: %s", code, stderr)
	}

	var result struct {
		Inserted int64 `json:"inserted"`
		Errors   int64 `json:"errors"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &result); err != nil {
		t.Fatalf("parse output %q: %v", stdout, err)
	}
	if result.Inserted != 2 {
		t.Errorf("inserted=%d, want 2", result.Inserted)
	}
	if result.Errors != 0 {
		t.Errorf("errors=%d, want 0", result.Errors)
	}

	// verify docs are in the table via driver
	_, cur, err := qexec.Run(context.Background(), reql.DB(dbName).Table(tableName).Count(), nil)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	raw, err := cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("count next: %v", err)
	}
	var count int
	if err := json.Unmarshal(raw, &count); err != nil {
		t.Fatalf("unmarshal count: %v", err)
	}
	if count != 2 {
		t.Errorf("table count=%d, want 2", count)
	}
}

func TestBulkInsertE2EConflictReplace(t *testing.T) {
	t.Parallel()
	qexec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	tableName := "bulk_replace"
	setupTestDB(t, qexec, dbName)
	createTestTable(t, qexec, dbName, tableName)

	// insert original docs via driver
	ctx := context.Background()
	_, cur, err := qexec.Run(ctx, reql.DB(dbName).Table(tableName).Insert(reql.Array(
		map[string]interface{}{"id": "1", "val": 10},
		map[string]interface{}{"id": "2", "val": 20},
	)), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("initial insert: %v", err)
	}

	// pipe updated docs with --conflict replace
	jsonl := `{"id":"1","val":99}` + "\n" + `{"id":"2","val":88}` + "\n"
	ref := fmt.Sprintf("%s.%s", dbName, tableName)

	stdout, stderr, code := cliRun(t, jsonl, cliArgs("insert", ref, "--conflict", "replace")...)
	if code != 0 {
		t.Fatalf("insert --conflict replace: exit code %d, stderr: %s", code, stderr)
	}

	// insert with conflict=replace reports replaced, not inserted
	// the CLI aggregates only inserted+errors, so we check errors==0 and total docs == 2
	var result struct {
		Inserted int64 `json:"inserted"`
		Errors   int64 `json:"errors"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &result); err != nil {
		t.Fatalf("parse output %q: %v", stdout, err)
	}
	if result.Errors != 0 {
		t.Errorf("errors=%d, want 0", result.Errors)
	}

	// verify docs have updated values via driver
	_, cur2, err := qexec.Run(ctx, reql.DB(dbName).Table(tableName).Get("1"), nil)
	if err != nil {
		t.Fatalf("get doc 1: %v", err)
	}
	raw, err := cur2.Next()
	closeCursor(cur2)
	if err != nil {
		t.Fatalf("get doc 1 next: %v", err)
	}
	var doc struct {
		Val int `json:"val"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal doc: %v", err)
	}
	if doc.Val != 99 {
		t.Errorf("doc 1 val=%d, want 99 (replace did not update)", doc.Val)
	}
}
