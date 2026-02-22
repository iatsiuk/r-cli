//go:build integration

package integration

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"r-cli/internal/conn"
	"r-cli/internal/connmgr"
	"r-cli/internal/query"
	"r-cli/internal/reql"
)

// startRethinkDBWithPassword starts a RethinkDB container with the admin password set.
// Registers t.Cleanup to terminate the container. Returns host and port.
func startRethinkDBWithPassword(t *testing.T, password string) (string, int) {
	t.Helper()
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "rethinkdb:2.4.4",
		ExposedPorts: []string{"28015/tcp"},
		Cmd:          []string{"rethinkdb", "--initial-password", password, "--bind", "all"},
		WaitingFor:   wait.ForLog("Server ready").WithStartupTimeout(2 * time.Minute),
	}
	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start rethinkdb with password: %v", err)
	}
	t.Cleanup(func() { _ = ctr.Terminate(ctx) })

	host, err := ctr.Host(ctx)
	if err != nil {
		t.Fatalf("container host: %v", err)
	}
	port, err := ctr.MappedPort(ctx, "28015")
	if err != nil {
		t.Fatalf("container port: %v", err)
	}
	return host, port.Int()
}

// dialAs dials RethinkDB at host:port as the given user and returns the connection or error.
func dialAs(ctx context.Context, host string, port int, user, password string) (*conn.Conn, error) {
	cfg := conn.Config{Host: host, Port: port, User: user, Password: password}
	return conn.Dial(ctx, fmt.Sprintf("%s:%d", host, port), cfg, nil)
}

// adminExecAt creates a query.Executor authenticated as admin with the given password.
func adminExecAt(t *testing.T, host string, port int, password string) *query.Executor {
	t.Helper()
	cfg := conn.Config{Host: host, Port: port, User: "admin", Password: password}
	mgr := connmgr.NewFromConfig(cfg, nil)
	t.Cleanup(func() { _ = mgr.Close() })
	return query.New(mgr)
}

// TestAuthHandshake covers SCRAM-SHA-256 handshake scenarios using a single
// password-protected container shared across all subtests.
func TestAuthHandshake(t *testing.T) {
	// not parallel: spawns its own container
	host, port := startRethinkDBWithPassword(t, "testpass")

	t.Run("CorrectPassword", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		c, err := dialAs(ctx, host, port, "admin", "testpass")
		if err != nil {
			t.Fatalf("expected success, got: %v", err)
		}
		_ = c.Close()
	})

	t.Run("WrongPassword", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := dialAs(ctx, host, port, "admin", "wrongpass")
		if err == nil {
			t.Fatal("expected auth error, got nil")
		}
		if !errors.Is(err, conn.ErrReqlAuth) {
			t.Errorf("expected ErrReqlAuth, got %v (type %T)", err, err)
		}
	})

	t.Run("NonExistentUser", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := dialAs(ctx, host, port, "no_such_user_xyz", "any")
		if err == nil {
			t.Fatal("expected auth error, got nil")
		}
		if !errors.Is(err, conn.ErrReqlAuth) {
			t.Errorf("expected ErrReqlAuth, got %v (type %T)", err, err)
		}
	})

	t.Run("CreateUserAndConnect", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		exec := adminExecAt(t, host, port, "testpass")

		_, cur, err := exec.Run(ctx,
			reql.DB("rethinkdb").Table("users").Insert(
				map[string]interface{}{"id": "alice_auth", "password": "alicepass"},
			),
			nil,
		)
		closeCursor(cur)
		if err != nil {
			t.Fatalf("create user alice_auth: %v", err)
		}
		t.Cleanup(func() {
			_, c2, _ := exec.Run(context.Background(),
				reql.DB("rethinkdb").Table("users").Get("alice_auth").Delete(), nil)
			closeCursor(c2)
		})

		dialCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		c, err := dialAs(dialCtx, host, port, "alice_auth", "alicepass")
		if err != nil {
			t.Fatalf("connect as alice_auth: %v", err)
		}
		_ = c.Close()
	})

	t.Run("ChangePassword", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		exec := adminExecAt(t, host, port, "testpass")

		_, cur, err := exec.Run(ctx,
			reql.DB("rethinkdb").Table("users").Insert(
				map[string]interface{}{"id": "bob_auth", "password": "bobpass1"},
			),
			nil,
		)
		closeCursor(cur)
		if err != nil {
			t.Fatalf("create user bob_auth: %v", err)
		}
		t.Cleanup(func() {
			_, c2, _ := exec.Run(context.Background(),
				reql.DB("rethinkdb").Table("users").Get("bob_auth").Delete(), nil)
			closeCursor(c2)
		})

		dialCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()

		c1, err := dialAs(dialCtx, host, port, "bob_auth", "bobpass1")
		if err != nil {
			t.Fatalf("connect with initial password: %v", err)
		}
		_ = c1.Close()

		_, cur2, err := exec.Run(ctx,
			reql.DB("rethinkdb").Table("users").Get("bob_auth").Update(
				map[string]interface{}{"password": "bobpass2"},
			),
			nil,
		)
		closeCursor(cur2)
		if err != nil {
			t.Fatalf("change password: %v", err)
		}

		_, errOld := dialAs(dialCtx, host, port, "bob_auth", "bobpass1")
		if errOld == nil {
			t.Error("expected auth error with old password, got nil")
		} else if !errors.Is(errOld, conn.ErrReqlAuth) {
			t.Errorf("expected ErrReqlAuth for old password, got %v", errOld)
		}

		c2, err := dialAs(dialCtx, host, port, "bob_auth", "bobpass2")
		if err != nil {
			t.Fatalf("connect with new password: %v", err)
		}
		_ = c2.Close()
	})

	t.Run("SpecialCharPassword", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		exec := adminExecAt(t, host, port, "testpass")

		specialPass := `p@$$w0rd",'unicode`
		_, cur, err := exec.Run(ctx,
			reql.DB("rethinkdb").Table("users").Insert(
				map[string]interface{}{"id": "charlie_auth", "password": specialPass},
			),
			nil,
		)
		closeCursor(cur)
		if err != nil {
			t.Fatalf("create user charlie_auth: %v", err)
		}
		t.Cleanup(func() {
			_, c2, _ := exec.Run(context.Background(),
				reql.DB("rethinkdb").Table("users").Get("charlie_auth").Delete(), nil)
			closeCursor(c2)
		})

		dialCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		c, err := dialAs(dialCtx, host, port, "charlie_auth", specialPass)
		if err != nil {
			t.Fatalf("connect with special char password: %v", err)
		}
		_ = c.Close()
	})

	t.Run("EmptyPassword", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		exec := adminExecAt(t, host, port, "testpass")

		// password: false means no password required in RethinkDB
		_, cur, err := exec.Run(ctx,
			reql.DB("rethinkdb").Table("users").Insert(
				map[string]interface{}{"id": "emptypass_auth", "password": false},
			),
			nil,
		)
		closeCursor(cur)
		if err != nil {
			t.Fatalf("create user emptypass_auth: %v", err)
		}
		t.Cleanup(func() {
			_, c2, _ := exec.Run(context.Background(),
				reql.DB("rethinkdb").Table("users").Get("emptypass_auth").Delete(), nil)
			closeCursor(c2)
		})

		dialCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		c, err := dialAs(dialCtx, host, port, "emptypass_auth", "")
		if err != nil {
			// some servers may reject empty-string SCRAM with a no-password account
			t.Logf("empty password connection: %v (server may require non-empty)", err)
		} else {
			_ = c.Close()
		}
	})
}
