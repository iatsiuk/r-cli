//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"r-cli/internal/conn"
	"r-cli/internal/connmgr"
	"r-cli/internal/query"
	"r-cli/internal/reql"
	"r-cli/internal/response"
)

// execAs creates a query.Executor authenticated as the given user.
func execAs(t *testing.T, host string, port int, user, password string) *query.Executor {
	t.Helper()
	cfg := conn.Config{Host: host, Port: port, User: user, Password: password}
	mgr := connmgr.NewFromConfig(cfg, nil)
	t.Cleanup(func() { _ = mgr.Close() })
	return query.New(mgr)
}

// createUser inserts a user into rethinkdb.users and registers cleanup to delete them.
func createUser(t *testing.T, exec *query.Executor, username, password string) {
	t.Helper()
	ctx := context.Background()
	_, cur, err := exec.Run(ctx,
		reql.DB("rethinkdb").Table("users").Insert(
			map[string]interface{}{"id": username, "password": password},
		),
		nil,
	)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("create user %s: %v", username, err)
	}
	t.Cleanup(func() {
		_, c, _ := exec.Run(context.Background(),
			reql.DB("rethinkdb").Table("users").Get(username).Delete(), nil)
		closeCursor(c)
	})
}

// isPermissionError reports whether err is (or wraps) a *response.ReqlPermissionError.
func isPermissionError(err error) bool {
	var e *response.ReqlPermissionError
	return errors.As(err, &e)
}

func TestGlobalPermissions(t *testing.T) {
	host, port := startRethinkDBWithPassword(t, "testpass")
	admin := adminExecAt(t, host, port, "testpass")
	ctx := context.Background()

	const (
		testDB    = "globalperm_db"
		testTable = "items"
	)

	_, cur, err := admin.Run(ctx, reql.DBCreate(testDB), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	t.Cleanup(func() {
		_, c, _ := admin.Run(context.Background(), reql.DBDrop(testDB), nil)
		closeCursor(c)
	})
	_, cur, err = admin.Run(ctx, reql.DB(testDB).TableCreate(testTable), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	t.Run("NoPermissionsPermissionError", func(t *testing.T) {
		t.Parallel()
		createUser(t, admin, "gp_noperm", "pass")
		userExec := execAs(t, host, port, "gp_noperm", "pass")
		qCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, c, err := userExec.Run(qCtx, reql.DB(testDB).Table(testTable).Count(), nil)
		closeCursor(c)
		if !isPermissionError(err) {
			t.Errorf("expected ReqlPermissionError, got %v (type %T)", err, err)
		}
	})

	t.Run("GlobalReadAllowsDBListAndCount", func(t *testing.T) {
		t.Parallel()
		createUser(t, admin, "gp_reader", "pass")
		_, c, err := admin.Run(ctx, reql.Grant("gp_reader", map[string]interface{}{"read": true}), nil)
		closeCursor(c)
		if err != nil {
			t.Fatalf("grant read: %v", err)
		}
		userExec := execAs(t, host, port, "gp_reader", "pass")
		qCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, c, err = userExec.Run(qCtx, reql.DBList(), nil)
		closeCursor(c)
		if err != nil {
			t.Errorf("expected dbList success with global read, got: %v", err)
		}

		_, c, err = userExec.Run(qCtx, reql.DB(testDB).Table(testTable).Count(), nil)
		closeCursor(c)
		if err != nil {
			t.Errorf("expected count success with global read, got: %v", err)
		}
	})

	t.Run("GlobalReadNoWriteInsertFails", func(t *testing.T) {
		t.Parallel()
		createUser(t, admin, "gp_readnowrite", "pass")
		_, c, err := admin.Run(ctx, reql.Grant("gp_readnowrite", map[string]interface{}{"read": true}), nil)
		closeCursor(c)
		if err != nil {
			t.Fatalf("grant read: %v", err)
		}
		userExec := execAs(t, host, port, "gp_readnowrite", "pass")
		qCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, c, err = userExec.Run(qCtx,
			reql.DB(testDB).Table(testTable).Insert(map[string]interface{}{"v": 1}),
			nil,
		)
		closeCursor(c)
		if !isPermissionError(err) {
			t.Errorf("expected ReqlPermissionError on insert with read-only, got %v (type %T)", err, err)
		}
	})

	t.Run("GlobalReadWriteInsertSucceeds", func(t *testing.T) {
		t.Parallel()
		createUser(t, admin, "gp_readwrite", "pass")
		_, c, err := admin.Run(ctx,
			reql.Grant("gp_readwrite", map[string]interface{}{"read": true, "write": true}),
			nil,
		)
		closeCursor(c)
		if err != nil {
			t.Fatalf("grant read+write: %v", err)
		}
		userExec := execAs(t, host, port, "gp_readwrite", "pass")
		qCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, c, err = userExec.Run(qCtx,
			reql.DB(testDB).Table(testTable).Insert(map[string]interface{}{"v": 2}),
			nil,
		)
		closeCursor(c)
		if err != nil {
			t.Errorf("expected insert success with global read+write, got: %v", err)
		}
	})

	t.Run("GlobalWriteNoReadFails", func(t *testing.T) {
		t.Parallel()
		createUser(t, admin, "gp_writenoread", "pass")
		_, c, err := admin.Run(ctx, reql.Grant("gp_writenoread", map[string]interface{}{"write": true}), nil)
		closeCursor(c)
		if err != nil {
			t.Fatalf("grant write: %v", err)
		}
		userExec := execAs(t, host, port, "gp_writenoread", "pass")
		qCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, c, err = userExec.Run(qCtx, reql.DB(testDB).Table(testTable).Count(), nil)
		closeCursor(c)
		if !isPermissionError(err) {
			t.Errorf("expected ReqlPermissionError on count with write-only, got %v (type %T)", err, err)
		}
	})

	t.Run("RevokePermissions", func(t *testing.T) {
		t.Parallel()
		createUser(t, admin, "gp_revoke", "pass")
		_, c, err := admin.Run(ctx, reql.Grant("gp_revoke", map[string]interface{}{"read": true}), nil)
		closeCursor(c)
		if err != nil {
			t.Fatalf("grant read: %v", err)
		}
		userExec := execAs(t, host, port, "gp_revoke", "pass")
		qCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		_, c, err = userExec.Run(qCtx, reql.DB(testDB).Table(testTable).Count(), nil)
		closeCursor(c)
		if err != nil {
			t.Fatalf("expected count success before revoke, got: %v", err)
		}

		_, c, err = admin.Run(ctx, reql.Grant("gp_revoke", map[string]interface{}{"read": false}), nil)
		closeCursor(c)
		if err != nil {
			t.Fatalf("revoke read: %v", err)
		}

		// open a fresh connection to ensure the new permissions are applied
		userExec2 := execAs(t, host, port, "gp_revoke", "pass")
		_, c, err = userExec2.Run(qCtx, reql.DB(testDB).Table(testTable).Count(), nil)
		closeCursor(c)
		if !isPermissionError(err) {
			t.Errorf("expected ReqlPermissionError after revoke, got %v (type %T)", err, err)
		}
	})
}
