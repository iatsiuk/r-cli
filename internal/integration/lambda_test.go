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

func TestLambdaFilter(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "age": 15},
		{"id": "2", "age": 25},
		{"id": "3", "age": 30},
		{"id": "4", "age": 10},
	})

	expr := fmt.Sprintf(`r.db("%s").table("docs").filter((x) => x("age").gt(21))`, dbName)
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
		t.Errorf("got %d rows for age>21, want 2", len(rows))
	}
	for _, raw := range rows {
		var doc struct {
			Age float64 `json:"age"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if doc.Age <= 21 {
			t.Errorf("age %v should be >21", doc.Age)
		}
	}
}

func TestLambdaReduce(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	// 2 docs so reduce((a,b) => a('val').add(b('val'))) works as a single pairwise sum
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "val": 10},
		{"id": "2", "val": 20},
	})

	expr := fmt.Sprintf(`r.db("%s").table("docs").reduce((a, b) => a("val").add(b("val")))`, dbName)
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
	var sum float64
	if err := json.Unmarshal(raw, &sum); err != nil {
		t.Fatalf("unmarshal sum: %v", err)
	}
	if sum != 30 {
		t.Errorf("sum=%v, want 30", sum)
	}
}

func TestLambdaMap(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "name": "alice"},
		{"id": "2", "name": "bob"},
		{"id": "3", "name": "carol"},
	})

	expr := fmt.Sprintf(`r.db("%s").table("docs").map((x) => x("name").upcase())`, dbName)
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
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}

	names := make(map[string]bool)
	for _, raw := range rows {
		var name string
		if err := json.Unmarshal(raw, &name); err != nil {
			t.Fatalf("unmarshal name: %v", err)
		}
		if name != strings.ToUpper(name) {
			t.Errorf("name %q is not uppercase", name)
		}
		names[name] = true
	}
	for _, want := range []string{"ALICE", "BOB", "CAROL"} {
		if !names[want] {
			t.Errorf("missing %q in map result", want)
		}
	}
}

func TestLambdaInnerJoin(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "users")
	createTestTable(t, exec, dbName, "orders")

	seedTable(t, exec, dbName, "users", []map[string]interface{}{
		{"id": "u1"},
		{"id": "u2"},
		{"id": "u3"},
	})
	seedTable(t, exec, dbName, "orders", []map[string]interface{}{
		{"id": "o1", "uid": "u1"},
		{"id": "o2", "uid": "u2"},
		{"id": "o3", "uid": "u2"},
	})

	// u1->o1 (1 pair), u2->o2,o3 (2 pairs), u3 has no match -> 3 total
	expr := fmt.Sprintf(
		`r.db("%s").table("users").innerJoin(r.db("%s").table("orders"), (left, right) => left("id").eq(right("uid")))`,
		dbName, dbName,
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
	if len(rows) != 3 {
		t.Errorf("innerJoin got %d rows, want 3", len(rows))
	}
	for _, raw := range rows {
		var pair struct {
			Left  map[string]interface{} `json:"left"`
			Right map[string]interface{} `json:"right"`
		}
		if err := json.Unmarshal(raw, &pair); err != nil {
			t.Fatalf("unmarshal join pair: %v", err)
		}
		if pair.Left == nil || pair.Right == nil {
			t.Errorf("join pair has nil side: %s", string(raw))
		}
		leftID, _ := pair.Left["id"].(string)
		rightUID, _ := pair.Right["uid"].(string)
		if leftID != rightUID {
			t.Errorf("join mismatch: user.id=%q != order.uid=%q", leftID, rightUID)
		}
	}
}

func TestLambdaCLI(t *testing.T) {
	t.Parallel()
	qexec := newExecutor(t)

	dbName := sanitizeID(t.Name())
	setupTestDB(t, qexec, dbName)
	createTestTable(t, qexec, dbName, "people")
	seedTable(t, qexec, dbName, "people", []map[string]interface{}{
		{"id": "1", "age": 15, "name": "alice"},
		{"id": "2", "age": 25, "name": "bob"},
		{"id": "3", "age": 30, "name": "carol"},
	})

	expr := fmt.Sprintf(`r.db("%s").table("people").filter((x) => x("age").gt(21))`, dbName)
	stdout, _, code := cliRun(t, "", cliArgs("-f", "json", expr)...)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}

	var docs []map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &docs); err != nil {
		t.Fatalf("unmarshal output: %v\noutput: %q", err, stdout)
	}
	if len(docs) != 2 {
		t.Errorf("got %d docs, want 2 (age>21)", len(docs))
	}
	for _, doc := range docs {
		age, _ := doc["age"].(float64)
		if age <= 21 {
			t.Errorf("doc age %v should be >21", age)
		}
	}
}
