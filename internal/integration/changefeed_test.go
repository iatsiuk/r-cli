//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"r-cli/internal/reql"
)

// changeDoc represents a RethinkDB changefeed event.
type changeDoc struct {
	OldVal json.RawMessage `json:"old_val"`
	NewVal json.RawMessage `json:"new_val"`
}

// isNullRaw reports whether a RawMessage is absent or JSON null.
func isNullRaw(raw json.RawMessage) bool {
	return raw == nil || string(raw) == "null"
}

func TestChangefeedInsert(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Changes(), nil)
	if err != nil {
		t.Fatalf("start changefeed: %v", err)
	}
	defer closeCursor(cur)

	// insert a doc after changefeed is listening
	go func() {
		time.Sleep(100 * time.Millisecond)
		bgCtx := context.Background()
		_, c, _ := exec.Run(bgCtx, reql.DB(dbName).Table("docs").Insert(
			map[string]interface{}{"id": "d1", "v": 1},
		), nil)
		closeCursor(c)
	}()

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}

	var ch changeDoc
	if err := json.Unmarshal(raw, &ch); err != nil {
		t.Fatalf("unmarshal change: %v", err)
	}
	if !isNullRaw(ch.OldVal) {
		t.Errorf("old_val should be null for insert, got %s", ch.OldVal)
	}
	if isNullRaw(ch.NewVal) {
		t.Errorf("new_val should not be null for insert, got %s", ch.NewVal)
	}
	var newDoc map[string]interface{}
	if err := json.Unmarshal(ch.NewVal, &newDoc); err != nil {
		t.Fatalf("unmarshal new_val: %v", err)
	}
	if newDoc["id"] != "d1" {
		t.Errorf("new_val.id=%v, want d1", newDoc["id"])
	}
}

func TestChangefeedUpdate(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	// pre-insert before changefeed so we don't see the insert event
	_, c, err := exec.Run(bgCtx, reql.DB(dbName).Table("docs").Insert(
		map[string]interface{}{"id": "d1", "v": 1},
	), nil)
	closeCursor(c)
	if err != nil {
		t.Fatalf("pre-insert: %v", err)
	}

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Changes(), nil)
	if err != nil {
		t.Fatalf("start changefeed: %v", err)
	}
	defer closeCursor(cur)

	go func() {
		time.Sleep(100 * time.Millisecond)
		_, c2, _ := exec.Run(bgCtx, reql.DB(dbName).Table("docs").Get("d1").Update(
			map[string]interface{}{"v": 2},
		), nil)
		closeCursor(c2)
	}()

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}

	var ch changeDoc
	if err := json.Unmarshal(raw, &ch); err != nil {
		t.Fatalf("unmarshal change: %v", err)
	}
	if isNullRaw(ch.OldVal) {
		t.Errorf("old_val should not be null for update")
	}
	if isNullRaw(ch.NewVal) {
		t.Errorf("new_val should not be null for update")
	}
	var newDoc map[string]interface{}
	if err := json.Unmarshal(ch.NewVal, &newDoc); err != nil {
		t.Fatalf("unmarshal new_val: %v", err)
	}
	if v, ok := newDoc["v"].(float64); !ok || v != 2 {
		t.Errorf("new_val.v=%v, want 2", newDoc["v"])
	}
}

func TestChangefeedDelete(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	// pre-insert before changefeed so we don't see the insert event
	_, c, err := exec.Run(bgCtx, reql.DB(dbName).Table("docs").Insert(
		map[string]interface{}{"id": "d1", "v": 1},
	), nil)
	closeCursor(c)
	if err != nil {
		t.Fatalf("pre-insert: %v", err)
	}

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Changes(), nil)
	if err != nil {
		t.Fatalf("start changefeed: %v", err)
	}
	defer closeCursor(cur)

	go func() {
		time.Sleep(100 * time.Millisecond)
		_, c2, _ := exec.Run(bgCtx, reql.DB(dbName).Table("docs").Get("d1").Delete(), nil)
		closeCursor(c2)
	}()

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}

	var ch changeDoc
	if err := json.Unmarshal(raw, &ch); err != nil {
		t.Fatalf("unmarshal change: %v", err)
	}
	if isNullRaw(ch.OldVal) {
		t.Errorf("old_val should not be null for delete")
	}
	if !isNullRaw(ch.NewVal) {
		t.Errorf("new_val should be null for delete, got %s", ch.NewVal)
	}
}

func TestChangefeedClose(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Changes(), nil)
	if err != nil {
		t.Fatalf("start changefeed: %v", err)
	}

	if err := cur.Close(); err != nil {
		t.Errorf("close changefeed returned error: %v", err)
	}
}

func TestChangefeedIncludeInitial(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	// pre-insert 2 docs before starting the changefeed
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "a", "v": 1},
		{"id": "b", "v": 2},
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Changes(reql.OptArgs{"include_initial": true}), nil)
	if err != nil {
		t.Fatalf("start changefeed with include_initial: %v", err)
	}
	defer closeCursor(cur)

	// read 2 initial documents; they should have new_val set (old_val=null)
	for i := range 2 {
		raw, err := cur.Next()
		if err != nil {
			t.Fatalf("cursor next [%d]: %v", i, err)
		}
		var ch changeDoc
		if err := json.Unmarshal(raw, &ch); err != nil {
			t.Fatalf("unmarshal change [%d]: %v", i, err)
		}
		if isNullRaw(ch.NewVal) {
			t.Errorf("change[%d]: new_val should not be null for initial doc", i)
		}
	}
}
