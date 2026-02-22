//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"r-cli/internal/conn"
	"r-cli/internal/connmgr"
	"r-cli/internal/cursor"
	"r-cli/internal/query"
	"r-cli/internal/reql"
)

var (
	containerHost string
	containerPort int
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "rethinkdb:2.4.4",
		ExposedPorts: []string{"28015/tcp"},
		WaitingFor:   wait.ForListeningPort("28015/tcp").WithStartupTimeout(2 * time.Minute),
	}

	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		if ctr != nil {
			_ = ctr.Terminate(ctx)
		}
		_, _ = fmt.Fprintf(os.Stderr, "start rethinkdb container: %v\n", err)
		os.Exit(1)
	}

	host, err := ctr.Host(ctx)
	if err != nil {
		_ = ctr.Terminate(ctx)
		_, _ = fmt.Fprintf(os.Stderr, "container host: %v\n", err)
		os.Exit(1)
	}

	port, err := ctr.MappedPort(ctx, "28015")
	if err != nil {
		_ = ctr.Terminate(ctx)
		_, _ = fmt.Fprintf(os.Stderr, "container port: %v\n", err)
		os.Exit(1)
	}

	containerHost = host
	containerPort = port.Int()

	code := m.Run()
	_ = ctr.Terminate(ctx)
	os.Exit(code)
}

// defaultCfg returns a Config pointing at the shared test container.
func defaultCfg() conn.Config {
	return conn.Config{
		Host:     containerHost,
		Port:     containerPort,
		User:     "admin",
		Password: "",
	}
}

// newExecutor creates an Executor backed by the shared test container.
// Cleanup is registered via t.Cleanup so it runs after any t.Cleanup
// callbacks registered by setupTestDB (LIFO order ensures DB drop before close).
func newExecutor(t *testing.T) *query.Executor {
	t.Helper()
	mgr := connmgr.NewFromConfig(defaultCfg(), nil)
	t.Cleanup(func() { _ = mgr.Close() })
	return query.New(mgr)
}

// closeCursor closes a cursor if non-nil, discarding errors.
func closeCursor(cur cursor.Cursor) {
	if cur != nil {
		_ = cur.Close()
	}
}

// setupTestDB creates a database and registers cleanup to drop it.
func setupTestDB(t *testing.T, exec *query.Executor, dbName string) {
	t.Helper()
	ctx := context.Background()
	_, cur, err := exec.Run(ctx, reql.DBCreate(dbName), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("setup db %s: %v", dbName, err)
	}
	t.Cleanup(func() {
		_, cur2, _ := exec.Run(context.Background(), reql.DBDrop(dbName), nil)
		closeCursor(cur2)
	})
}

// createTestTable creates a table inside dbName.
func createTestTable(t *testing.T, exec *query.Executor, dbName, tableName string) {
	t.Helper()
	ctx := context.Background()
	_, cur, err := exec.Run(ctx, reql.DB(dbName).TableCreate(tableName), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("setup table %s.%s: %v", dbName, tableName, err)
	}
}
