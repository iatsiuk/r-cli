//go:build integration

package integration

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"r-cli/internal/conn"
	"r-cli/internal/reql"
	"r-cli/internal/response"
)

func TestQueryNonExistentTable(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	// do NOT create a table - query it directly to get a runtime error

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("no_such_table"), nil)
	closeCursor(cur)
	if err == nil {
		t.Fatal("expected error for non-existent table, got nil")
	}

	var runtimeErr *response.ReqlRuntimeError
	var nonExistErr *response.ReqlNonExistenceError
	if !errors.As(err, &runtimeErr) && !errors.As(err, &nonExistErr) {
		t.Errorf("expected ReqlRuntimeError or ReqlNonExistenceError, got %T: %v", err, err)
	}
}

func TestQueryNonExistentDatabase(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()

	_, cur, err := exec.Run(ctx, reql.DB("nonexistent_db_xyz_12345").Table("docs"), nil)
	closeCursor(cur)
	if err == nil {
		t.Fatal("expected error for non-existent database, got nil")
	}

	var runtimeErr *response.ReqlRuntimeError
	var nonExistErr *response.ReqlNonExistenceError
	if !errors.As(err, &runtimeErr) && !errors.As(err, &nonExistErr) {
		t.Errorf("expected ReqlRuntimeError or ReqlNonExistenceError, got %T: %v", err, err)
	}
}

func TestMalformedReQL(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// connect directly to send a raw invalid ReQL term
	addr := fmt.Sprintf("%s:%d", containerHost, containerPort)
	c, err := conn.Dial(ctx, addr, defaultCfg(), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = c.Close() }()

	// [1, [15, []], {}] = START query with TABLE term (type 15) and 0 args
	// TABLE requires 1-2 args; 0 args triggers COMPILE_ERROR
	payload := []byte(`[1,[15,[]],{}]`)
	raw, err := c.Send(ctx, c.NextToken(), payload)
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("parse response: %v", err)
	}

	rerr := response.MapError(resp)
	if rerr == nil {
		t.Fatal("expected error for malformed ReQL, got success")
	}

	var compileErr *response.ReqlCompileError
	if !errors.As(rerr, &compileErr) {
		t.Errorf("expected ReqlCompileError, got %T: %v", rerr, rerr)
	}
}

func TestTypeMismatch(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	// r.expr(1).add("world") -> type mismatch: cannot add number and string
	_, cur, err := exec.Run(ctx, reql.Datum(1).Add(reql.Datum("world")), nil)
	closeCursor(cur)
	if err == nil {
		t.Fatal("expected error for type mismatch, got nil")
	}

	var runtimeErr *response.ReqlRuntimeError
	if !errors.As(err, &runtimeErr) {
		t.Errorf("expected ReqlRuntimeError, got %T: %v", err, err)
	}
}

func TestContextTimeout(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	// warm up: establish the underlying connection
	ctx0 := context.Background()
	_, warmup, err := exec.Run(ctx0, reql.DBList(), nil)
	closeCursor(warmup)
	if err != nil {
		t.Fatalf("warmup: %v", err)
	}

	// use an already-expired context so Send returns DeadlineExceeded before
	// the response can arrive from the server (network roundtrip takes longer)
	expired, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	_, _, err = exec.Run(expired, reql.DBList(), nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded, got %v (type %T)", err, err)
	}

	// connection must still be usable after the timeout
	_, cur, err := exec.Run(ctx0, reql.DBList(), nil)
	closeCursor(cur)
	if err != nil {
		t.Errorf("connection should be usable after timeout: %v", err)
	}
}
