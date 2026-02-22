//go:build integration

package integration

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"

	"r-cli/internal/query"
	"r-cli/internal/reql"
)

// seedLargeTable inserts n documents into dbName.tableName in batches of batchSize.
func seedLargeTable(t *testing.T, exec *query.Executor, dbName, tableName string, n, batchSize int) {
	t.Helper()
	ctx := context.Background()
	for start := 0; start < n; start += batchSize {
		end := start + batchSize
		if end > n {
			end = n
		}
		args := make([]interface{}, end-start)
		for i := range args {
			args[i] = map[string]interface{}{"n": start + i}
		}
		_, cur, err := exec.Run(ctx, reql.DB(dbName).Table(tableName).Insert(reql.Array(args...)), nil)
		closeCursor(cur)
		if err != nil {
			t.Fatalf("seed large table batch %d: %v", start, err)
		}
	}
}

func TestStreamMultiBatch(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	const n = 1500
	seedLargeTable(t, exec, dbName, "docs", n, 200)

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs"), nil)
	if err != nil {
		t.Fatalf("table scan: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != n {
		t.Errorf("got %d rows, want %d", len(rows), n)
	}
}

func TestCursorNextOneByOne(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "n": 1},
		{"id": "2", "n": 2},
		{"id": "3", "n": 3},
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs"), nil)
	if err != nil {
		t.Fatalf("table scan: %v", err)
	}
	defer closeCursor(cur)

	count := 0
	for {
		_, err := cur.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("cursor next: %v", err)
		}
		count++
	}
	if count != 3 {
		t.Errorf("got %d docs via Next(), want 3", count)
	}
}

func TestCursorAll(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "a"}, {"id": "b"}, {"id": "c"}, {"id": "d"}, {"id": "e"},
	})

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs"), nil)
	if err != nil {
		t.Fatalf("table scan: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 5 {
		t.Errorf("got %d rows, want 5", len(rows))
	}
}

func TestCursorCloseEarly(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedLargeTable(t, exec, dbName, "docs", 1000, 200)

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs"), nil)
	if err != nil {
		t.Fatalf("table scan: %v", err)
	}

	// read a few items then close mid-stream
	for range 5 {
		if _, err := cur.Next(); err != nil {
			t.Fatalf("cursor next: %v", err)
		}
	}

	if err := cur.Close(); err != nil {
		t.Errorf("close mid-stream returned error: %v", err)
	}
}

func TestCursorContextCancel(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedLargeTable(t, exec, dbName, "docs", 1000, 200)

	ctx, cancel := context.WithCancel(context.Background())
	// force small batches so the cursor uses SUCCESS_PARTIAL (streaming mode),
	// which is required for context cancellation to propagate.
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs"), reql.OptArgs{"max_batch_rows": 1})
	if err != nil {
		cancel()
		t.Fatalf("table scan: %v", err)
	}
	defer closeCursor(cur)

	// read a few items then cancel context
	for range 3 {
		if _, err := cur.Next(); err != nil {
			cancel()
			t.Fatalf("cursor next before cancel: %v", err)
		}
	}
	cancel()

	// drain until error; context cancellation should propagate
	var lastErr error
	for {
		_, err := cur.Next()
		if err != nil {
			lastErr = err
			break
		}
	}
	if !errors.Is(lastErr, context.Canceled) {
		t.Errorf("expected context.Canceled after cancel, got %v (type %T)", lastErr, lastErr)
	}
}

func TestTwoConcurrentCursors(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	const n = 200
	seedLargeTable(t, exec, dbName, "docs", n, 200)

	_, cur1, err := exec.Run(ctx, reql.DB(dbName).Table("docs"), nil)
	if err != nil {
		t.Fatalf("cursor1 start: %v", err)
	}
	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("docs"), nil)
	if err != nil {
		closeCursor(cur1)
		t.Fatalf("cursor2 start: %v", err)
	}

	var wg sync.WaitGroup
	var count1, count2 int
	var err1, err2 error

	wg.Add(2)
	go func() {
		defer wg.Done()
		rows, e := cur1.All()
		count1 = len(rows)
		err1 = e
		closeCursor(cur1)
	}()
	go func() {
		defer wg.Done()
		rows, e := cur2.All()
		count2 = len(rows)
		err2 = e
		closeCursor(cur2)
	}()
	wg.Wait()

	if err1 != nil {
		t.Errorf("cursor1: %v", err1)
	}
	if err2 != nil {
		t.Errorf("cursor2: %v", err2)
	}
	if count1 != n {
		t.Errorf("cursor1 got %d rows, want %d", count1, n)
	}
	if count2 != n {
		t.Errorf("cursor2 got %d rows, want %d", count2, n)
	}
}
