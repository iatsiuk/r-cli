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

func TestParserFixesBracketNumericIndex(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "items")
	seedTable(t, exec, dbName, "items", []map[string]interface{}{
		{"id": "1", "val": 10},
		{"id": "2", "val": 20},
		{"id": "3", "val": 30},
	})

	expr := fmt.Sprintf(`r.db("%s").table("items").orderBy("id").limit(1)(0)`, dbName)
	term, err := parser.Parse(expr)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, cur, err := exec.Run(ctx, term, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if doc["id"] != "1" {
		t.Errorf("got id=%v, want 1", doc["id"])
	}
}

func TestParserFixesSample(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "items")
	docs := make([]map[string]interface{}, 10)
	for i := range docs {
		docs[i] = map[string]interface{}{"id": fmt.Sprintf("%d", i+1), "val": i + 1}
	}
	seedTable(t, exec, dbName, "items", docs)

	expr := fmt.Sprintf(`r.db("%s").table("items").sample(3)`, dbName)
	term, err := parser.Parse(expr)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, cur, err := exec.Run(ctx, term, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	defer closeCursor(cur)

	// sample returns SUCCESS_ATOM with an array value; cur.Next() gives the whole array
	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var sampled []json.RawMessage
	if err := json.Unmarshal(raw, &sampled); err != nil {
		t.Fatalf("unmarshal sample array: %v", err)
	}
	if len(sampled) != 3 {
		t.Errorf("got %d docs in sample, want 3", len(sampled))
	}
}

func TestParserFixesNestedFunction(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	// seed using parser so nested arrays produce correct MAKE_ARRAY ReQL terms
	insertExpr := fmt.Sprintf(
		`r.db("%s").table("docs").insert([{id: "1", items: [{type: "a"}, {type: "b"}]}, {id: "2", items: [{type: "a"}, {type: "c"}]}])`,
		dbName,
	)
	insertTerm, err := parser.Parse(insertExpr)
	if err != nil {
		t.Fatalf("parse insert: %v", err)
	}
	_, cur, err := exec.Run(ctx, insertTerm, nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("insert docs: %v", err)
	}

	expr := fmt.Sprintf(
		`r.db("%s").table("docs").map(function(doc){ return doc("items").filter(function(i){ return i("type").eq("a") }) })`,
		dbName,
	)
	term, err := parser.Parse(expr)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, cur, err = exec.Run(ctx, term, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
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
		var items []map[string]interface{}
		if err := json.Unmarshal(raw, &items); err != nil {
			t.Fatalf("unmarshal items: %v", err)
		}
		if len(items) != 1 {
			t.Errorf("expected 1 filtered item, got %d: %s", len(items), string(raw))
			continue
		}
		if items[0]["type"] != "a" {
			t.Errorf("expected type=a, got %v", items[0]["type"])
		}
	}
}

func TestParserFixesNestedArrow(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	// seed using parser so nested arrays produce correct MAKE_ARRAY ReQL terms
	insertExpr := fmt.Sprintf(
		`r.db("%s").table("docs").insert([{id: "1", items: [{type: "a"}, {type: "b"}]}, {id: "2", items: [{type: "a"}, {type: "c"}]}])`,
		dbName,
	)
	insertTerm, err := parser.Parse(insertExpr)
	if err != nil {
		t.Fatalf("parse insert: %v", err)
	}
	_, cur, err := exec.Run(ctx, insertTerm, nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("insert docs: %v", err)
	}

	expr := fmt.Sprintf(
		`r.db("%s").table("docs").map((doc) => doc("items").filter((i) => i("type").eq("a")))`,
		dbName,
	)
	term, err := parser.Parse(expr)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, cur, err = exec.Run(ctx, term, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
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
		var items []map[string]interface{}
		if err := json.Unmarshal(raw, &items); err != nil {
			t.Fatalf("unmarshal items: %v", err)
		}
		if len(items) != 1 {
			t.Errorf("expected 1 filtered item, got %d: %s", len(items), string(raw))
			continue
		}
		if items[0]["type"] != "a" {
			t.Errorf("expected type=a, got %v", items[0]["type"])
		}
	}
}

func TestParserFixesInsertOptArgs(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "items")

	expr := fmt.Sprintf(`r.db("%s").table("items").insert({id: "new", val: 1}, {return_changes: true})`, dbName)
	term, err := parser.Parse(expr)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, cur, err := exec.Run(ctx, term, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result["inserted"] != float64(1) {
		t.Errorf("inserted=%v, want 1", result["inserted"])
	}
	changes, ok := result["changes"].([]interface{})
	if !ok {
		t.Fatalf("changes field missing or not array: %v", result)
	}
	if len(changes) != 1 {
		t.Errorf("changes has %d entries, want 1", len(changes))
	}
	change, ok := changes[0].(map[string]interface{})
	if !ok {
		t.Fatalf("changes[0] is not object: %v", changes[0])
	}
	if change["new_val"] == nil {
		t.Error("changes[0].new_val is nil")
	}
}

func TestParserFixesArrowParenObject(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "people")
	seedTable(t, exec, dbName, "people", []map[string]interface{}{
		{"id": "1", "first": "Alice", "last": "Smith"},
	})

	expr := fmt.Sprintf(
		`r.db("%s").table("people").map(row => ({full: row("first").add(" ").add(row("last"))}))`,
		dbName,
	)
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
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(rows[0], &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if doc["full"] != "Alice Smith" {
		t.Errorf("full=%v, want 'Alice Smith'", doc["full"])
	}
}

func TestParserFixesCLI(t *testing.T) {
	t.Parallel()
	qexec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	setupTestDB(t, qexec, dbName)
	createTestTable(t, qexec, dbName, "items")
	docs := make([]map[string]interface{}, 10)
	for i := range docs {
		docs[i] = map[string]interface{}{"id": fmt.Sprintf("%d", i+1), "val": i + 1}
	}
	seedTable(t, qexec, dbName, "items", docs)

	expr := fmt.Sprintf(`r.db("%s").table("items").sample(3)`, dbName)
	stdout, _, code := cliRun(t, "", cliArgs("-f", "json", expr)...)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	var result []interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &result); err != nil {
		t.Fatalf("unmarshal output: %v\noutput: %q", err, stdout)
	}
	if len(result) != 3 {
		t.Errorf("got %d items, want 3", len(result))
	}
}
