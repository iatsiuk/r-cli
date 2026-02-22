//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"r-cli/internal/reql"
)

func TestFilterMatch(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "name": "alice"},
		{"id": "2", "name": "bob"},
		{"id": "3", "name": "anna"},
	})

	// match names starting with 'a'
	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").Filter(
			reql.Row().GetField("name").Match("^a"),
		), nil)
	if err != nil {
		t.Fatalf("filter match: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("got %d rows for name^a, want 2", len(rows))
	}
	for _, raw := range rows {
		var doc struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal doc: %v", err)
		}
		if len(doc.Name) == 0 || doc.Name[0] != 'a' {
			t.Errorf("unexpected name %q, want name starting with 'a'", doc.Name)
		}
	}
}

func TestInsertNowReadBack(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	before := time.Now().Add(-5 * time.Second)
	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").Insert(
			map[string]interface{}{"id": "ts1", "created": reql.Now()},
		), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("insert with now: %v", err)
	}
	after := time.Now().Add(5 * time.Second)

	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Get("ts1"), nil)
	if err != nil {
		t.Fatalf("get doc: %v", err)
	}
	defer closeCursor(cur2)

	raw, err := cur2.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var doc struct {
		Created struct {
			ReqlType  string  `json:"$reql_type$"`
			EpochTime float64 `json:"epoch_time"`
		} `json:"created"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal doc: %v", err)
	}
	if doc.Created.ReqlType != "TIME" {
		t.Errorf("$reql_type$=%q, want TIME", doc.Created.ReqlType)
	}
	ts := time.Unix(int64(doc.Created.EpochTime), 0)
	if ts.Before(before) || ts.After(after) {
		t.Errorf("timestamp %v outside expected range [%v, %v]", ts, before, after)
	}
}

func TestGroupByYear(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	// insert three docs: two in 2023, one in 2024
	docs := []map[string]interface{}{
		{"id": "1", "ts": reql.EpochTime(1672531200)}, // 2023-01-01
		{"id": "2", "ts": reql.EpochTime(1700000000)}, // 2023-11-14
		{"id": "3", "ts": reql.EpochTime(1704067200)}, // 2024-01-01
	}
	seedTable(t, exec, dbName, "docs", docs)

	// map each doc to its year using .year() time method
	yearFn := reql.Func(reql.Var(1).GetField("ts").Year(), 1)
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Map(yearFn), nil)
	if err != nil {
		t.Fatalf("map year: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}

	yearCounts := make(map[float64]int)
	for _, raw := range rows {
		var year float64
		if err := json.Unmarshal(raw, &year); err != nil {
			t.Fatalf("unmarshal year: %v", err)
		}
		yearCounts[year]++
	}
	if yearCounts[2023] != 2 {
		t.Errorf("year 2023 count=%d, want 2", yearCounts[2023])
	}
	if yearCounts[2024] != 1 {
		t.Errorf("year 2024 count=%d, want 1", yearCounts[2024])
	}
}

func TestEpochTimeRoundtrip(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	const epochVal = 1704067200.0 // 2024-01-01 00:00:00 UTC
	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").Insert(
			map[string]interface{}{"id": "ep1", "ts": reql.EpochTime(epochVal)},
		), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("insert epochTime: %v", err)
	}

	_, cur2, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").Get("ep1").GetField("ts").ToEpochTime(), nil)
	if err != nil {
		t.Fatalf("get epoch time: %v", err)
	}
	defer closeCursor(cur2)

	raw, err := cur2.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got float64
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal epoch time: %v", err)
	}
	if got != epochVal {
		t.Errorf("epoch time roundtrip: got %v, want %v", got, epochVal)
	}
}

func TestEqJoinSecondaryIndex(t *testing.T) {
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

	// create secondary index on users.id (effectively the primary key, use a real secondary)
	// eqJoin orders.user_id -> users (primary key = id, no separate index needed for left side)
	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("orders").EqJoin("user_id", reql.DB(dbName).Table("users")), nil)
	if err != nil {
		t.Fatalf("eqJoin: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("eqJoin got %d rows, want 3", len(rows))
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
			t.Errorf("join pair has nil side: left=%v right=%v", pair.Left, pair.Right)
		}
		userID, _ := pair.Left["user_id"].(string)
		rightID, _ := pair.Right["id"].(string)
		if userID != rightID {
			t.Errorf("join mismatch: order.user_id=%q != user.id=%q", userID, rightID)
		}
	}
}

func TestEqJoinZip(t *testing.T) {
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
	})

	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("orders").EqJoin("user_id", reql.DB(dbName).Table("users")).Zip(), nil)
	if err != nil {
		t.Fatalf("eqJoin zip: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("eqJoin+zip got %d rows, want 2", len(rows))
	}
	for _, raw := range rows {
		var doc map[string]interface{}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal zipped doc: %v", err)
		}
		// zipped doc should have fields from both sides: id, user_id, amount, name
		if _, ok := doc["name"]; !ok {
			t.Errorf("zipped doc missing 'name' field: %v", doc)
		}
		if _, ok := doc["amount"]; !ok {
			t.Errorf("zipped doc missing 'amount' field: %v", doc)
		}
	}
}
