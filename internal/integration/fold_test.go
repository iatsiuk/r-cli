//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"r-cli/internal/reql"
)

func TestFoldMethod(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	// sum of [1,2,3] = 6
	arr := reql.Array(reql.Datum(1), reql.Datum(2), reql.Datum(3))
	fn := reql.Func(reql.Var(1).Add(reql.Var(2)), 1, 2)
	_, cur, err := exec.Run(ctx, arr.Fold(reql.Datum(0), fn), nil)
	if err != nil {
		t.Fatalf("fold: %v", err)
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
	if int(got) != 6 {
		t.Errorf("fold sum = %v, want 6", got)
	}
}

func TestFoldOnTable(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	tableName := "fold_tbl"
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, tableName)

	docs := []map[string]interface{}{
		{"id": "a", "val": 10},
		{"id": "b", "val": 20},
		{"id": "c", "val": 30},
	}
	seedTable(t, exec, dbName, tableName, docs)

	// fold to sum the "val" field: 10 + 20 + 30 = 60
	table := reql.DB(dbName).Table(tableName)
	fn := reql.Func(reql.Var(1).Add(reql.Var(2).GetField("val")), 1, 2)
	_, cur, err := exec.Run(ctx, table.Fold(reql.Datum(0), fn), nil)
	if err != nil {
		t.Fatalf("fold on table: %v", err)
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
	if int(got) != 60 {
		t.Errorf("fold table sum = %v, want 60", got)
	}
}
