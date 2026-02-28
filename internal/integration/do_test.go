//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"r-cli/internal/reql"
	"r-cli/internal/reql/parser"
)

func TestDoTopLevel(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "a"}, {"id": "b"}, {"id": "c"},
	})

	// r.do(count, n => n.add(1)) -- count = 3, result = 4
	count := reql.DB(dbName).Table("docs").Count()
	addFn := reql.Func(reql.Var(1).Add(1), 1)
	_, cur, err := exec.Run(ctx, reql.Do(count, addFn), nil)
	if err != nil {
		t.Fatalf("do top-level: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	var got float64
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if int(got) != 4 {
		t.Errorf("do result = %v, want 4", got)
	}
}

func TestDoChainForm(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	// r.expr(5).do(n => n.mul(2)) -- result = 10
	base := reql.Datum(5)
	mulFn := reql.Func(reql.Var(1).Mul(2), 1)
	_, cur, err := exec.Run(ctx, base.Do(mulFn), nil)
	if err != nil {
		t.Fatalf("do chain: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	var got float64
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if int(got) != 10 {
		t.Errorf("do chain result = %v, want 10", got)
	}
}

func TestDoParserTopLevel(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1"}, {"id": "2"}, {"id": "3"},
	})

	// r.do(r.db(dbName).table("docs").count(), n => n.add(1)) -- count=3, result=4
	expr := `r.do(r.db("` + dbName + `").table("docs").count(), n => n.add(1))`
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
		t.Fatalf("next: %v", err)
	}
	var got float64
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if int(got) != 4 {
		t.Errorf("parser do result = %v, want 4", got)
	}
}

func TestDoParserChainForm(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	// r.expr(5).do(n => n.mul(2)) -- result = 10
	term, err := parser.Parse(`r.expr(5).do(n => n.mul(2))`)
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
		t.Fatalf("next: %v", err)
	}
	var got float64
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if int(got) != 10 {
		t.Errorf("parser do chain result = %v, want 10", got)
	}
}
