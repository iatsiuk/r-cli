//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"r-cli/internal/conn"
	"r-cli/internal/connmgr"
	"r-cli/internal/proto"
	"r-cli/internal/query"
	"r-cli/internal/reql"
	"r-cli/internal/response"
)

// TestConnManager50Concurrent sends 50 queries concurrently through a single
// multiplexed connection to verify token-based dispatch works under load.
func TestConnManager50Concurrent(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	const n = 50
	errs := make([]error, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			_, cur, err := exec.Run(ctx, reql.DBList(), nil)
			closeCursor(cur)
			errs[i] = err
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: %v", i, err)
		}
	}
}

// startRethinkDBForRestart starts a fresh RethinkDB container for reconnect tests.
// Caller must terminate the returned container.
func startRethinkDBForRestart(ctx context.Context) (testcontainers.Container, error) {
	req := testcontainers.ContainerRequest{
		Image:        "rethinkdb:2.4",
		ExposedPorts: []string{"28015/tcp"},
		WaitingFor:   wait.ForLog("Server ready").WithStartupTimeout(2 * time.Minute),
	}
	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("start container: %w", err)
	}
	return ctr, nil
}

// containerDialFunc returns a DialFunc that queries the container for its
// current port mapping at each dial attempt. This handles port reassignment
// after a container stop+start cycle.
func containerDialFunc(ctr testcontainers.Container) connmgr.DialFunc {
	return func(ctx context.Context) (*conn.Conn, error) {
		host, err := ctr.Host(ctx)
		if err != nil {
			return nil, fmt.Errorf("container host: %w", err)
		}
		port, err := ctr.MappedPort(ctx, "28015")
		if err != nil {
			return nil, fmt.Errorf("container port: %w", err)
		}
		cfg := conn.Config{Host: host, Port: port.Int(), User: "admin", Password: ""}
		return conn.Dial(ctx, fmt.Sprintf("%s:%d", host, port.Int()), cfg, nil)
	}
}

// TestConnManagerReconnectAfterRestart stops a dedicated container, starts it
// again, and verifies the ConnManager re-establishes the connection.
// Docker may reassign the host port after stop+start, so the dial function
// queries the mapped port dynamically on each attempt.
func TestConnManagerReconnectAfterRestart(t *testing.T) {
	// not parallel: spawns its own container
	ctx := context.Background()
	ctr, err := startRethinkDBForRestart(ctx)
	if err != nil {
		t.Fatalf("start container: %v", err)
	}
	defer func() { _ = ctr.Terminate(ctx) }()

	mgr := connmgr.New(containerDialFunc(ctr))
	defer func() { _ = mgr.Close() }()
	exec := query.New(mgr)

	// verify initial connection works
	_, cur, err := exec.Run(ctx, reql.DBList(), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("initial query: %v", err)
	}

	// stop the container (nil = default stop timeout)
	if err := ctr.Stop(ctx, nil); err != nil {
		t.Fatalf("stop container: %v", err)
	}

	// start the container back up
	if err := ctr.Start(ctx); err != nil {
		t.Fatalf("start container: %v", err)
	}

	// poll until the server is ready; ConnManager re-dials with the new port
	deadline := time.Now().Add(2 * time.Minute)
	var reconnectErr error
	for time.Now().Before(deadline) {
		_, cur, reconnectErr = exec.Run(ctx, reql.DBList(), nil)
		closeCursor(cur)
		if reconnectErr == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if reconnectErr != nil {
		t.Fatalf("reconnect after restart: %v", reconnectErr)
	}
}

// TestConnManagerCloseWithActiveQuery verifies that closing the ConnManager
// while a changefeed is blocked in CONTINUE causes the cursor to return an error.
func TestConnManagerCloseWithActiveQuery(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupExec := newExecutor(t)
	setupTestDB(t, setupExec, dbName)
	createTestTable(t, setupExec, dbName, "docs")

	// separate mgr so we can close it without affecting other tests
	mgr := connmgr.NewFromConfig(defaultCfg(), nil)
	defer func() { _ = mgr.Close() }()
	exec := query.New(mgr)

	// changefeed on an empty table blocks immediately in CONTINUE
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Changes(), nil)
	if err != nil {
		t.Fatalf("start changefeed: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		_, err := cur.Next()
		errCh <- err
	}()

	// allow the reader goroutine to start and block in Send
	time.Sleep(200 * time.Millisecond)

	// close the manager: closes the conn, unblocks pending Send with an error
	_ = mgr.Close()

	select {
	case err := <-errCh:
		if err == nil {
			t.Error("expected error after manager close, got nil")
		}
	case <-time.After(5 * time.Second):
		_ = cur.Close()
		t.Fatal("reader goroutine did not unblock within timeout")
	}
}

// sendNoreplyWait sends NOREPLY_WAIT on the manager's connection and verifies
// the server responds with WAIT_COMPLETE.
func sendNoreplyWait(ctx context.Context, mgr *connmgr.ConnManager) error {
	c, err := mgr.Get(ctx)
	if err != nil {
		return fmt.Errorf("get conn: %w", err)
	}
	token := c.NextToken()
	raw, err := c.Send(ctx, token, []byte(`[4]`))
	if err != nil {
		return fmt.Errorf("send noreply_wait: %w", err)
	}
	resp, err := response.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	if resp.Type != proto.ResponseWaitComplete {
		return fmt.Errorf("expected WAIT_COMPLETE (%d), got %d", proto.ResponseWaitComplete, resp.Type)
	}
	return nil
}

// TestNoreplyInsert verifies that inserting with noreply=true returns no cursor
// and that the document is visible after NOREPLY_WAIT.
func TestNoreplyInsert(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupExec := newExecutor(t)
	setupTestDB(t, setupExec, dbName)
	createTestTable(t, setupExec, dbName, "docs")

	// dedicated mgr so we can issue NOREPLY_WAIT on the same connection
	mgr := connmgr.NewFromConfig(defaultCfg(), nil)
	defer func() { _ = mgr.Close() }()
	exec := query.New(mgr)

	doc := map[string]interface{}{"id": "noreply-1", "v": 42}
	profile, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").Insert(doc),
		reql.OptArgs{"noreply": true},
	)
	if err != nil {
		t.Fatalf("noreply insert: %v", err)
	}
	if cur != nil {
		_ = cur.Close()
		t.Error("noreply insert returned non-nil cursor")
	}
	if profile != nil {
		t.Error("noreply insert returned non-nil profile")
	}

	// wait for the write to be durable on the same connection
	if err := sendNoreplyWait(ctx, mgr); err != nil {
		t.Fatalf("noreply_wait: %v", err)
	}

	// verify the document is visible
	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Get("noreply-1"), nil)
	if err != nil {
		t.Fatalf("get after noreply: %v", err)
	}
	defer closeCursor(cur2)

	raw, err := cur2.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["id"] != "noreply-1" {
		t.Errorf("id=%v, want noreply-1", got["id"])
	}
}

// TestNoreplyWait sends multiple noreply inserts followed by NOREPLY_WAIT
// and verifies all writes are visible in the table.
func TestNoreplyWait(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupExec := newExecutor(t)
	setupTestDB(t, setupExec, dbName)
	createTestTable(t, setupExec, dbName, "docs")

	mgr := connmgr.NewFromConfig(defaultCfg(), nil)
	defer func() { _ = mgr.Close() }()
	exec := query.New(mgr)

	const numDocs = 5
	for i := range numDocs {
		doc := map[string]interface{}{"id": fmt.Sprintf("nrw-%d", i), "n": i}
		_, cur, err := exec.Run(ctx,
			reql.DB(dbName).Table("docs").Insert(doc),
			reql.OptArgs{"noreply": true},
		)
		closeCursor(cur)
		if err != nil {
			t.Fatalf("noreply insert %d: %v", i, err)
		}
	}

	if err := sendNoreplyWait(ctx, mgr); err != nil {
		t.Fatalf("noreply_wait: %v", err)
	}

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Count(), nil)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var count int
	if err := json.Unmarshal(raw, &count); err != nil {
		t.Fatalf("unmarshal count: %v", err)
	}
	if count != numDocs {
		t.Errorf("count=%d, want %d", count, numDocs)
	}
}
