//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"r-cli/internal/reql"
)

func TestInfoMethod(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	tableName := "info_tbl"
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, tableName)

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table(tableName).Info(), nil)
	if err != nil {
		t.Fatalf("info: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["name"] != tableName {
		t.Errorf("info name=%v, want %q", got["name"], tableName)
	}
	if got["type"] != "TABLE" {
		t.Errorf("info type=%v, want TABLE", got["type"])
	}
}

func TestOffsetsOfMethod(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	term := reql.Array(reql.Datum("a"), reql.Datum("b"), reql.Datum("c"), reql.Datum("b")).OffsetsOf(reql.Datum("b"))
	_, cur, err := exec.Run(ctx, term, nil)
	if err != nil {
		t.Fatalf("offsetsOf: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	var got []interface{}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("offsetsOf got %d results, want 2", len(got))
	}
	if int(got[0].(float64)) != 1 || int(got[1].(float64)) != 3 {
		t.Errorf("offsetsOf got %v, want [1, 3]", got)
	}
}
