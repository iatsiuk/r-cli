//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"r-cli/internal/query"
	"r-cli/internal/reql"
)

// waitForIndex creates a secondary index on tableName and blocks until it is ready.
func waitForIndex(t *testing.T, exec *query.Executor, dbName, tableName, indexName string) {
	t.Helper()
	ctx := context.Background()
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table(tableName).IndexCreate(indexName), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("indexCreate %s: %v", indexName, err)
	}
	_, cur, err = exec.Run(ctx, reql.DB(dbName).Table(tableName).IndexWait(indexName), nil)
	if err != nil {
		t.Fatalf("indexWait %s: %v", indexName, err)
	}
	closeCursor(cur)
}

func TestIndexCreate(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "items")

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("items").IndexCreate("score"), nil)
	if err != nil {
		t.Fatalf("index create: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var result struct {
		Created int `json:"created"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Created != 1 {
		t.Errorf("created=%d, want 1", result.Created)
	}
}

func TestIndexList(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "items")
	waitForIndex(t, exec, dbName, "items", "score")

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("items").IndexList(), nil)
	if err != nil {
		t.Fatalf("index list: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var indexes []string
	if err := json.Unmarshal(raw, &indexes); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	found := false
	for _, idx := range indexes {
		if idx == "score" {
			found = true
		}
	}
	if !found {
		t.Errorf("index 'score' not in list %v", indexes)
	}
}

func TestIndexWait(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "items")

	// create index
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("items").IndexCreate("score"), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("index create: %v", err)
	}

	// wait for index readiness
	_, cur, err = exec.Run(ctx, reql.DB(dbName).Table("items").IndexWait("score"), nil)
	if err != nil {
		t.Fatalf("index wait: %v", err)
	}
	if cur == nil {
		t.Fatal("expected cursor from IndexWait")
	}
	raw, err := cur.Next()
	_ = cur.Close()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}

	// IndexWait returns SUCCESS_ATOM: the value is an array of status objects
	var statuses []struct {
		Index string `json:"index"`
		Ready bool   `json:"ready"`
	}
	if err := json.Unmarshal(raw, &statuses); err != nil {
		t.Fatalf("unmarshal statuses: %v", err)
	}
	if len(statuses) == 0 {
		t.Fatal("indexWait returned empty array")
	}
	if statuses[0].Index != "score" {
		t.Errorf("index=%q, want score", statuses[0].Index)
	}
	if !statuses[0].Ready {
		t.Errorf("index not ready after IndexWait")
	}
}

func TestIndexStatus(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "items")
	waitForIndex(t, exec, dbName, "items", "score")

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("items").IndexStatus("score"), nil)
	if err != nil {
		t.Fatalf("index status: %v", err)
	}
	if cur == nil {
		t.Fatal("expected cursor from IndexStatus")
	}
	raw, err := cur.Next()
	_ = cur.Close()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}

	// IndexStatus returns SUCCESS_ATOM: the value is an array of status objects
	var statuses []struct {
		Index string `json:"index"`
		Ready bool   `json:"ready"`
	}
	if err := json.Unmarshal(raw, &statuses); err != nil {
		t.Fatalf("unmarshal statuses: %v", err)
	}
	if len(statuses) == 0 {
		t.Fatal("indexStatus returned empty array")
	}
	if statuses[0].Index != "score" {
		t.Errorf("index=%q, want score", statuses[0].Index)
	}
	if !statuses[0].Ready {
		t.Errorf("index ready=%v, want true", statuses[0].Ready)
	}
}

func TestGetAllWithIndex(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "items")
	waitForIndex(t, exec, dbName, "items", "score")
	seedTable(t, exec, dbName, "items", []map[string]interface{}{
		{"id": "a", "score": 10},
		{"id": "b", "score": 20},
		{"id": "c", "score": 10},
		{"id": "d", "score": 30},
	})

	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("items").GetAll(10, reql.OptArgs{"index": "score"}), nil)
	if err != nil {
		t.Fatalf("getAll with index: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("got %d rows, want 2 (docs with score=10)", len(rows))
	}
}

func TestBetweenWithIndex(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "items")
	waitForIndex(t, exec, dbName, "items", "score")
	seedTable(t, exec, dbName, "items", []map[string]interface{}{
		{"id": "a", "score": 10},
		{"id": "b", "score": 20},
		{"id": "c", "score": 30},
		{"id": "d", "score": 40},
		{"id": "e", "score": 50},
	})

	// between(15, 45) with index: returns score=20, 30, 40 (lower inclusive, upper exclusive)
	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("items").Between(15, 45, reql.OptArgs{"index": "score"}), nil)
	if err != nil {
		t.Fatalf("between with index: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 3 {
		t.Errorf("got %d rows, want 3 (score=20,30,40)", len(rows))
	}
}

func TestIndexDrop(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "items")
	waitForIndex(t, exec, dbName, "items", "score")

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("items").IndexDrop("score"), nil)
	if err != nil {
		t.Fatalf("index drop: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var result struct {
		Dropped int `json:"dropped"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Dropped != 1 {
		t.Errorf("dropped=%d, want 1", result.Dropped)
	}

	// verify index is gone
	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("items").IndexList(), nil)
	if err != nil {
		t.Fatalf("index list: %v", err)
	}
	defer closeCursor(cur2)

	raw2, err := cur2.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var indexes []string
	if err := json.Unmarshal(raw2, &indexes); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, idx := range indexes {
		if idx == "score" {
			t.Errorf("index 'score' still present after drop")
		}
	}
}

func TestIndexRename(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "items")
	waitForIndex(t, exec, dbName, "items", "tmp_score")

	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("items").IndexRename("tmp_score", "score_v2"), nil)
	if err != nil {
		t.Fatalf("index rename: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var result struct {
		Renamed int `json:"renamed"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Renamed != 1 {
		t.Errorf("renamed=%d, want 1", result.Renamed)
	}

	// verify new name exists, old does not
	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("items").IndexList(), nil)
	if err != nil {
		t.Fatalf("index list: %v", err)
	}
	defer closeCursor(cur2)

	raw2, err := cur2.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var indexes []string
	if err := json.Unmarshal(raw2, &indexes); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	foundNew, foundOld := false, false
	for _, idx := range indexes {
		if idx == "score_v2" {
			foundNew = true
		}
		if idx == "tmp_score" {
			foundOld = true
		}
	}
	if !foundNew {
		t.Errorf("index 'score_v2' not found after rename")
	}
	if foundOld {
		t.Errorf("index 'tmp_score' still present after rename")
	}
}
