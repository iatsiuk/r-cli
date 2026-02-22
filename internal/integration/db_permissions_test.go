//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"r-cli/internal/reql"
)

func TestDBPermissions(t *testing.T) {
	host, port := startRethinkDBWithPassword(t, "testpass")
	admin := execAs(t, host, port, "admin", "testpass")
	ctx := context.Background()

	const (
		allowedDB    = "dbperm_allowed"
		restrictedDB = "dbperm_restricted"
		testTable    = "items"
	)

	for _, db := range []string{allowedDB, restrictedDB} {
		_, cur, err := admin.Run(ctx, reql.DBCreate(db), nil)
		closeCursor(cur)
		if err != nil {
			t.Fatalf("create db %s: %v", db, err)
		}
		t.Cleanup(func() {
			_, c, _ := admin.Run(context.Background(), reql.DBDrop(db), nil)
			closeCursor(c)
		})
		_, cur, err = admin.Run(ctx, reql.DB(db).TableCreate(testTable), nil)
		closeCursor(cur)
		if err != nil {
			t.Fatalf("create table %s.%s: %v", db, testTable, err)
		}
	}

	t.Run("DBReadAllowsQueryInGrantedDB", func(t *testing.T) {
		t.Parallel()
		createUser(t, admin, "dp_reader", "pass")
		_, c, err := admin.Run(ctx,
			reql.DB(allowedDB).Grant("dp_reader", map[string]interface{}{"read": true}),
			nil,
		)
		closeCursor(c)
		if err != nil {
			t.Fatalf("grant db read: %v", err)
		}
		userExec := execAs(t, host, port, "dp_reader", "pass")
		qCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, c, err = userExec.Run(qCtx, reql.DB(allowedDB).Table(testTable).Count(), nil)
		closeCursor(c)
		if err != nil {
			t.Errorf("expected count success in granted db, got: %v", err)
		}
	})

	t.Run("DBReadDeniesQueryInOtherDB", func(t *testing.T) {
		t.Parallel()
		createUser(t, admin, "dp_reader_other", "pass")
		_, c, err := admin.Run(ctx,
			reql.DB(allowedDB).Grant("dp_reader_other", map[string]interface{}{"read": true}),
			nil,
		)
		closeCursor(c)
		if err != nil {
			t.Fatalf("grant db read: %v", err)
		}
		userExec := execAs(t, host, port, "dp_reader_other", "pass")
		qCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, c, err = userExec.Run(qCtx, reql.DB(restrictedDB).Table(testTable).Count(), nil)
		closeCursor(c)
		if !isPermissionError(err) {
			t.Errorf("expected ReqlPermissionError for non-granted db, got %v (type %T)", err, err)
		}
	})

	t.Run("DBWriteAllowsInsertInGrantedDB", func(t *testing.T) {
		t.Parallel()
		createUser(t, admin, "dp_writer", "pass")
		_, c, err := admin.Run(ctx,
			reql.DB(allowedDB).Grant("dp_writer", map[string]interface{}{"read": true, "write": true}),
			nil,
		)
		closeCursor(c)
		if err != nil {
			t.Fatalf("grant db read+write: %v", err)
		}
		userExec := execAs(t, host, port, "dp_writer", "pass")
		qCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, c, err = userExec.Run(qCtx,
			reql.DB(allowedDB).Table(testTable).Insert(map[string]interface{}{"v": 1}),
			nil,
		)
		closeCursor(c)
		if err != nil {
			t.Errorf("expected insert success in granted db, got: %v", err)
		}
	})

	t.Run("DBWriteDeniesInsertInOtherDB", func(t *testing.T) {
		t.Parallel()
		createUser(t, admin, "dp_writer_other", "pass")
		_, c, err := admin.Run(ctx,
			reql.DB(allowedDB).Grant("dp_writer_other", map[string]interface{}{"read": true, "write": true}),
			nil,
		)
		closeCursor(c)
		if err != nil {
			t.Fatalf("grant db read+write: %v", err)
		}
		userExec := execAs(t, host, port, "dp_writer_other", "pass")
		qCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, c, err = userExec.Run(qCtx,
			reql.DB(restrictedDB).Table(testTable).Insert(map[string]interface{}{"v": 2}),
			nil,
		)
		closeCursor(c)
		if !isPermissionError(err) {
			t.Errorf("expected ReqlPermissionError for insert in non-granted db, got %v (type %T)", err, err)
		}
	})

	t.Run("DBConfigAllowsTableCreate", func(t *testing.T) {
		t.Parallel()
		createUser(t, admin, "dp_config", "pass")
		_, c, err := admin.Run(ctx,
			reql.DB(allowedDB).Grant("dp_config", map[string]interface{}{"config": true}),
			nil,
		)
		closeCursor(c)
		if err != nil {
			t.Fatalf("grant db config: %v", err)
		}
		userExec := execAs(t, host, port, "dp_config", "pass")
		qCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, c, err = userExec.Run(qCtx, reql.DB(allowedDB).TableCreate("config_created"), nil)
		closeCursor(c)
		if err != nil {
			t.Errorf("expected TableCreate success with config permission, got: %v", err)
		}
		t.Cleanup(func() {
			_, c2, _ := admin.Run(context.Background(), reql.DB(allowedDB).TableDrop("config_created"), nil)
			closeCursor(c2)
		})
	})

	t.Run("DBConfigFalseDeniesTableCreate", func(t *testing.T) {
		t.Parallel()
		createUser(t, admin, "dp_noconfig", "pass")
		// grant read but not config
		_, c, err := admin.Run(ctx,
			reql.DB(allowedDB).Grant("dp_noconfig", map[string]interface{}{"read": true}),
			nil,
		)
		closeCursor(c)
		if err != nil {
			t.Fatalf("grant db read: %v", err)
		}
		userExec := execAs(t, host, port, "dp_noconfig", "pass")
		qCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, c, err = userExec.Run(qCtx, reql.DB(allowedDB).TableCreate("noconfig_created"), nil)
		closeCursor(c)
		if !isPermissionError(err) {
			t.Errorf("expected ReqlPermissionError for TableCreate without config, got %v (type %T)", err, err)
		}
	})
}
