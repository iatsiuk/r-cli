//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"r-cli/internal/reql"
)

func TestDuringBasicFilter(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "events")
	// seed: 3 events at known epoch timestamps
	// e1: 2023-01-01 = 1672531200
	// e2: 2023-06-01 = 1685577600
	// e3: 2024-01-01 = 1704067200
	seedTable(t, exec, dbName, "events", []map[string]interface{}{
		{"id": "e1", "ts": reql.EpochTime(1672531200)},
		{"id": "e2", "ts": reql.EpochTime(1685577600)},
		{"id": "e3", "ts": reql.EpochTime(1704067200)},
	})

	// during [2023-02-01, 2023-12-31) -- only e2 falls in this range
	start := reql.EpochTime(1675209600) // 2023-02-01
	end := reql.EpochTime(1703980800)   // 2023-12-31
	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("events").Filter(
			reql.Func(reql.Var(1).GetField("ts").During(start, end), 1),
		), nil)
	if err != nil {
		t.Fatalf("filter during: %v", err)
	}
	rows, err := cur.All()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("during filter got %d rows, want 1", len(rows))
	}
	var doc struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rows[0], &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if doc.ID != "e2" {
		t.Errorf("during filter returned id=%q, want e2", doc.ID)
	}
}

func TestDuringBoundaryBehavior(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "events")
	// e1 is exactly at start, e2 is in middle, e3 is exactly at end
	seedTable(t, exec, dbName, "events", []map[string]interface{}{
		{"id": "e1", "ts": reql.EpochTime(1672531200)}, // exactly at start
		{"id": "e2", "ts": reql.EpochTime(1685577600)}, // in range
		{"id": "e3", "ts": reql.EpochTime(1704067200)}, // exactly at end
	})

	// default bounds: left=closed, right=open => [start, end)
	// e1 at start is included, e3 at end is excluded
	start := reql.EpochTime(1672531200)
	end := reql.EpochTime(1704067200)
	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("events").Filter(
			reql.Func(reql.Var(1).GetField("ts").During(start, end), 1),
		), nil)
	if err != nil {
		t.Fatalf("during boundary: %v", err)
	}
	rows, err := cur.All()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	// [start, end) includes e1 and e2 but not e3
	if len(rows) != 2 {
		t.Fatalf("during [start, end) got %d rows, want 2", len(rows))
	}
	ids := make(map[string]bool)
	for _, r := range rows {
		var doc struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(r, &doc); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		ids[doc.ID] = true
	}
	if !ids["e1"] || !ids["e2"] || ids["e3"] {
		t.Errorf("during [start, end) returned ids=%v, want e1 and e2 only", ids)
	}
}
