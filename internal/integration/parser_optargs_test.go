//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"r-cli/internal/reql/parser"
)

func TestParserGetAllWithSecondaryIndexOptArgs(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "dept": "eng"},
		{"id": "2", "dept": "eng"},
		{"id": "3", "dept": "hr"},
	})

	waitForIndex(t, exec, dbName, "docs", "dept")

	expr := fmt.Sprintf(`r.db("%s").table("docs").getAll("eng", {index: "dept"})`, dbName)
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
		t.Errorf("getAll with secondary index: got %d rows, want 2", len(rows))
	}
}

func TestParserBetweenWithIndexOptArgs(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "score": 5},
		{"id": "2", "score": 50},
		{"id": "3", "score": 80},
		{"id": "4", "score": 15},
	})

	waitForIndex(t, exec, dbName, "docs", "score")

	// between 10 and 60 on secondary index "score" should match score=50 and score=15
	expr := fmt.Sprintf(`r.db("%s").table("docs").between(10, 60, {index: "score"})`, dbName)
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
		t.Errorf("between with index: got %d rows, want 2", len(rows))
	}
}

func TestParserOptArgsCamelCaseConversion(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "a", "score": 5},
		{"id": "b", "score": 15},
		{"id": "c", "score": 50},
		{"id": "d", "score": 80},
	})
	waitForIndex(t, exec, dbName, "docs", "score")

	t.Run("between_camelCase_bounds", func(t *testing.T) {
		// leftBound and rightBound are camelCase; should be auto-converted to snake_case
		expr := fmt.Sprintf(
			`r.db("%s").table("docs").between(10, 60, {index: "score", leftBound: "closed", rightBound: "closed"})`,
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
		// score=15 and score=50 are within [10,60] closed on both sides
		if len(rows) != 2 {
			t.Errorf("between with camelCase bounds: got %d rows, want 2", len(rows))
		}
	})

	t.Run("insert_returnChanges", func(t *testing.T) {
		// returnChanges is camelCase; should be converted to return_changes
		expr := fmt.Sprintf(
			`r.db("%s").table("docs").insert({id: "new1", score: 99}, {returnChanges: true})`,
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
		raw, err := cur.Next()
		if err != nil {
			t.Fatalf("cursor next: %v", err)
		}
		var result struct {
			Inserted int               `json:"inserted"`
			Changes  []json.RawMessage `json:"changes"`
		}
		if err := json.Unmarshal(raw, &result); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if result.Inserted != 1 {
			t.Errorf("inserted=%d, want 1", result.Inserted)
		}
		if len(result.Changes) != 1 {
			t.Errorf("changes len=%d, want 1 (return_changes not applied if 0)", len(result.Changes))
		}
	})

	t.Run("getAll_snake_case_index_still_works", func(t *testing.T) {
		// index is already snake_case; verify no regression
		expr := fmt.Sprintf(
			`r.db("%s").table("docs").getAll(15, {index: "score"})`,
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
			t.Errorf("getAll with snake_case index: got %d rows, want 1", len(rows))
		}
	})
}

func TestParserEqJoinWithIndexOptArgs(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "users")
	createTestTable(t, exec, dbName, "orders")

	seedTable(t, exec, dbName, "users", []map[string]interface{}{
		{"id": "u1", "name": "alice"},
		{"id": "u2", "name": "bob"},
	})
	seedTable(t, exec, dbName, "orders", []map[string]interface{}{
		{"id": "o1", "user_id": "u1", "amount": 100},
		{"id": "o2", "user_id": "u2", "amount": 200},
		{"id": "o3", "user_id": "u1", "amount": 150},
	})

	waitForIndex(t, exec, dbName, "orders", "user_id")

	// eqJoin users.id -> orders via secondary index user_id
	expr := fmt.Sprintf(
		`r.db("%s").table("users").eqJoin("id", r.db("%s").table("orders"), {index: "user_id"})`,
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
	// u1 matches o1 and o3, u2 matches o2: 3 pairs total
	if len(rows) != 3 {
		t.Errorf("eqJoin with index: got %d rows, want 3", len(rows))
	}
}
