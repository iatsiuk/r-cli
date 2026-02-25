//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"r-cli/internal/reql/parser"
)

func TestFunctionSyntaxFilter(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "routes")
	seedTable(t, exec, dbName, "routes", []map[string]interface{}{
		{"id": "1", "enabled": true},
		{"id": "2", "enabled": false},
		{"id": "3", "enabled": false},
	})

	expr := fmt.Sprintf(`r.db("%s").table("routes").filter(function(r){ return r("enabled").eq(false) })`, dbName)
	term, err := parser.Parse(expr)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, cur, err := exec.Run(ctx, term, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("got %d rows, want 2 (enabled=false)", len(rows))
	}
	for _, raw := range rows {
		var doc struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if doc.Enabled {
			t.Errorf("doc.enabled should be false: %s", string(raw))
		}
	}
}

func TestFunctionSyntaxArrowWithR(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "routes")
	seedTable(t, exec, dbName, "routes", []map[string]interface{}{
		{"id": "1", "enabled": true},
		{"id": "2", "enabled": false},
		{"id": "3", "enabled": false},
	})

	expr := fmt.Sprintf(`r.db("%s").table("routes").filter((r) => r("enabled").eq(false))`, dbName)
	term, err := parser.Parse(expr)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, cur, err := exec.Run(ctx, term, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("got %d rows, want 2 (enabled=false)", len(rows))
	}
	for _, raw := range rows {
		var doc struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if doc.Enabled {
			t.Errorf("doc.enabled should be false: %s", string(raw))
		}
	}
}

func TestFunctionSyntaxEquivalence(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "routes")
	seedTable(t, exec, dbName, "routes", []map[string]interface{}{
		{"id": "1", "enabled": true},
		{"id": "2", "enabled": false},
		{"id": "3", "enabled": false},
	})

	funcExpr := fmt.Sprintf(`r.db("%s").table("routes").filter(function(r){ return r("enabled").eq(false) })`, dbName)
	arrowExpr := fmt.Sprintf(`r.db("%s").table("routes").filter((r) => r("enabled").eq(false))`, dbName)

	funcTerm, err := parser.Parse(funcExpr)
	if err != nil {
		t.Fatalf("parse function expr: %v", err)
	}
	arrowTerm, err := parser.Parse(arrowExpr)
	if err != nil {
		t.Fatalf("parse arrow expr: %v", err)
	}

	_, cur1, err := exec.Run(ctx, funcTerm, nil)
	if err != nil {
		t.Fatalf("run function: %v", err)
	}
	defer closeCursor(cur1)

	_, cur2, err := exec.Run(ctx, arrowTerm, nil)
	if err != nil {
		t.Fatalf("run arrow: %v", err)
	}
	defer closeCursor(cur2)

	rows1, err := cur1.All()
	if err != nil {
		t.Fatalf("cursor1 all: %v", err)
	}
	rows2, err := cur2.All()
	if err != nil {
		t.Fatalf("cursor2 all: %v", err)
	}

	if len(rows1) != len(rows2) {
		t.Errorf("function syntax got %d rows, arrow got %d rows", len(rows1), len(rows2))
	}

	ids1 := extractRouteIDs(rows1)
	ids2 := extractRouteIDs(rows2)
	for id := range ids1 {
		if !ids2[id] {
			t.Errorf("id %q in function result but not in arrow result", id)
		}
	}
	for id := range ids2 {
		if !ids1[id] {
			t.Errorf("id %q in arrow result but not in function result", id)
		}
	}
}

func extractRouteIDs(rows []json.RawMessage) map[string]bool {
	ids := make(map[string]bool, len(rows))
	for _, raw := range rows {
		var doc struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(raw, &doc); err == nil && doc.ID != "" {
			ids[doc.ID] = true
		}
	}
	return ids
}

func TestFunctionSyntaxCLI(t *testing.T) {
	t.Parallel()
	qexec := newExecutor(t)

	dbName := sanitizeID(t.Name())
	setupTestDB(t, qexec, dbName)
	createTestTable(t, qexec, dbName, "routes")
	seedTable(t, qexec, dbName, "routes", []map[string]interface{}{
		{"id": "1", "enabled": true},
		{"id": "2", "enabled": false},
		{"id": "3", "enabled": false},
	})

	expr := fmt.Sprintf(`r.db("%s").table("routes").filter(function(r){ return r("enabled").eq(false) })`, dbName)
	stdout, _, code := cliRun(t, "", cliArgs("-f", "json", expr)...)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}

	var docs []map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &docs); err != nil {
		t.Fatalf("unmarshal output: %v\noutput: %q", err, stdout)
	}
	if len(docs) != 2 {
		t.Errorf("got %d docs, want 2 (enabled=false)", len(docs))
	}
	for _, doc := range docs {
		enabled, ok := doc["enabled"].(bool)
		if !ok {
			t.Errorf("doc.enabled missing or not bool: %v", doc)
			continue
		}
		if enabled {
			t.Errorf("doc.enabled should be false: %v", doc)
		}
	}
}

func TestFunctionSyntaxCLIArrowR(t *testing.T) {
	t.Parallel()
	qexec := newExecutor(t)

	dbName := sanitizeID(t.Name())
	setupTestDB(t, qexec, dbName)
	createTestTable(t, qexec, dbName, "routes")
	seedTable(t, qexec, dbName, "routes", []map[string]interface{}{
		{"id": "1", "enabled": true},
		{"id": "2", "enabled": false},
		{"id": "3", "enabled": false},
	})

	expr := fmt.Sprintf(`r.db("%s").table("routes").filter((r) => r("enabled").eq(false))`, dbName)
	stdout, _, code := cliRun(t, "", cliArgs("-f", "json", expr)...)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}

	var docs []map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &docs); err != nil {
		t.Fatalf("unmarshal output: %v\noutput: %q", err, stdout)
	}
	if len(docs) != 2 {
		t.Errorf("got %d docs, want 2 (enabled=false)", len(docs))
	}
	for _, doc := range docs {
		enabled, ok := doc["enabled"].(bool)
		if !ok {
			t.Errorf("doc.enabled missing or not bool: %v", doc)
			continue
		}
		if enabled {
			t.Errorf("doc.enabled should be false: %v", doc)
		}
	}
}
