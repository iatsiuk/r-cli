//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"r-cli/internal/reql"
)

func TestTablePermissions(t *testing.T) {
	host, port := startRethinkDBWithPassword(t, "testpass")
	admin := adminExecAt(t, host, port, "testpass")
	ctx := context.Background()

	const (
		testDB          = "tperm_db"
		allowedTable    = "allowed"
		restrictedTable = "restricted"
		otherDB         = "tperm_other_db"
	)

	for _, db := range []string{testDB, otherDB} {
		db := db
		_, cur, err := admin.Run(ctx, reql.DBCreate(db), nil)
		closeCursor(cur)
		if err != nil {
			t.Fatalf("create db %s: %v", db, err)
		}
		t.Cleanup(func() {
			_, c, _ := admin.Run(context.Background(), reql.DBDrop(db), nil)
			closeCursor(c)
		})
	}
	for _, tbl := range []string{allowedTable, restrictedTable} {
		_, cur, err := admin.Run(ctx, reql.DB(testDB).TableCreate(tbl), nil)
		closeCursor(cur)
		if err != nil {
			t.Fatalf("create table %s.%s: %v", testDB, tbl, err)
		}
	}
	_, cur, err := admin.Run(ctx, reql.DB(otherDB).TableCreate(allowedTable), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("create table %s.%s: %v", otherDB, allowedTable, err)
	}

	t.Run("TableReadAllowsQueryInGrantedTable", func(t *testing.T) {
		t.Parallel()
		createUser(t, admin, "tp_reader", "pass")
		_, c, err := admin.Run(ctx,
			reql.DB(testDB).Table(allowedTable).Grant("tp_reader", map[string]interface{}{"read": true}),
			nil,
		)
		closeCursor(c)
		if err != nil {
			t.Fatalf("grant table read: %v", err)
		}
		userExec := execAs(t, host, port, "tp_reader", "pass")
		qCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, c, err = userExec.Run(qCtx, reql.DB(testDB).Table(allowedTable).Count(), nil)
		closeCursor(c)
		if err != nil {
			t.Errorf("expected count success in granted table, got: %v", err)
		}
	})

	t.Run("TableReadDeniesQueryInOtherTable", func(t *testing.T) {
		t.Parallel()
		createUser(t, admin, "tp_reader_other", "pass")
		_, c, err := admin.Run(ctx,
			reql.DB(testDB).Table(allowedTable).Grant("tp_reader_other", map[string]interface{}{"read": true}),
			nil,
		)
		closeCursor(c)
		if err != nil {
			t.Fatalf("grant table read: %v", err)
		}
		userExec := execAs(t, host, port, "tp_reader_other", "pass")
		qCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, c, err = userExec.Run(qCtx, reql.DB(testDB).Table(restrictedTable).Count(), nil)
		closeCursor(c)
		if !isPermissionError(err) {
			t.Errorf("expected ReqlPermissionError for non-granted table, got %v (type %T)", err, err)
		}
	})

	t.Run("TableWriteAllowsInsertInGrantedTable", func(t *testing.T) {
		t.Parallel()
		createUser(t, admin, "tp_writer", "pass")
		_, c, err := admin.Run(ctx,
			reql.DB(testDB).Table(allowedTable).Grant("tp_writer", map[string]interface{}{"read": true, "write": true}),
			nil,
		)
		closeCursor(c)
		if err != nil {
			t.Fatalf("grant table read+write: %v", err)
		}
		userExec := execAs(t, host, port, "tp_writer", "pass")
		qCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, c, err = userExec.Run(qCtx,
			reql.DB(testDB).Table(allowedTable).Insert(map[string]interface{}{"v": 1}),
			nil,
		)
		closeCursor(c)
		if err != nil {
			t.Errorf("expected insert success in granted table, got: %v", err)
		}
	})

	t.Run("TableWriteDeniesInsertInOtherTable", func(t *testing.T) {
		t.Parallel()
		createUser(t, admin, "tp_writer_other", "pass")
		_, c, err := admin.Run(ctx,
			reql.DB(testDB).Table(allowedTable).Grant("tp_writer_other", map[string]interface{}{"read": true, "write": true}),
			nil,
		)
		closeCursor(c)
		if err != nil {
			t.Fatalf("grant table read+write: %v", err)
		}
		userExec := execAs(t, host, port, "tp_writer_other", "pass")
		qCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, c, err = userExec.Run(qCtx,
			reql.DB(testDB).Table(restrictedTable).Insert(map[string]interface{}{"v": 2}),
			nil,
		)
		closeCursor(c)
		if !isPermissionError(err) {
			t.Errorf("expected ReqlPermissionError for insert in non-granted table, got %v (type %T)", err, err)
		}
	})

	t.Run("GlobalReadDBWriteOverride", func(t *testing.T) {
		t.Parallel()
		createUser(t, admin, "tp_global_db", "pass")
		_, c, err := admin.Run(ctx,
			reql.Grant("tp_global_db", map[string]interface{}{"read": true}),
			nil,
		)
		closeCursor(c)
		if err != nil {
			t.Fatalf("grant global read: %v", err)
		}
		_, c, err = admin.Run(ctx,
			reql.DB(testDB).Grant("tp_global_db", map[string]interface{}{"write": true}),
			nil,
		)
		closeCursor(c)
		if err != nil {
			t.Fatalf("grant db write: %v", err)
		}
		userExec := execAs(t, host, port, "tp_global_db", "pass")
		qCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// global read: can read from any db
		_, c, err = userExec.Run(qCtx, reql.DB(otherDB).Table(allowedTable).Count(), nil)
		closeCursor(c)
		if err != nil {
			t.Errorf("expected count success with global read in other db, got: %v", err)
		}

		// global read + db write: can insert in testDB
		_, c, err = userExec.Run(qCtx,
			reql.DB(testDB).Table(allowedTable).Insert(map[string]interface{}{"v": 3}),
			nil,
		)
		closeCursor(c)
		if err != nil {
			t.Errorf("expected insert success in db with write grant, got: %v", err)
		}

		// no write in otherDB
		_, c, err = userExec.Run(qCtx,
			reql.DB(otherDB).Table(allowedTable).Insert(map[string]interface{}{"v": 4}),
			nil,
		)
		closeCursor(c)
		if !isPermissionError(err) {
			t.Errorf("expected ReqlPermissionError for insert in other db without write, got %v (type %T)", err, err)
		}
	})

	t.Run("DBReadFalseOverridesGlobalRead", func(t *testing.T) {
		t.Parallel()
		createUser(t, admin, "tp_db_override", "pass")
		_, c, err := admin.Run(ctx,
			reql.Grant("tp_db_override", map[string]interface{}{"read": true}),
			nil,
		)
		closeCursor(c)
		if err != nil {
			t.Fatalf("grant global read: %v", err)
		}
		_, c, err = admin.Run(ctx,
			reql.DB(testDB).Grant("tp_db_override", map[string]interface{}{"read": false}),
			nil,
		)
		closeCursor(c)
		if err != nil {
			t.Fatalf("revoke db read: %v", err)
		}
		userExec := execAs(t, host, port, "tp_db_override", "pass")
		qCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// global read still applies for otherDB
		_, c, err = userExec.Run(qCtx, reql.DB(otherDB).Table(allowedTable).Count(), nil)
		closeCursor(c)
		if err != nil {
			t.Errorf("expected count success in other db, got: %v", err)
		}

		// db-level read=false overrides global read for testDB
		_, c, err = userExec.Run(qCtx, reql.DB(testDB).Table(allowedTable).Count(), nil)
		closeCursor(c)
		if !isPermissionError(err) {
			t.Errorf("expected ReqlPermissionError for read in restricted db, got %v (type %T)", err, err)
		}
	})

	t.Run("TableGrantOverridesDB", func(t *testing.T) {
		t.Parallel()
		createUser(t, admin, "tp_tbl_override", "pass")
		// deny read at db level
		_, c, err := admin.Run(ctx,
			reql.DB(testDB).Grant("tp_tbl_override", map[string]interface{}{"read": false}),
			nil,
		)
		closeCursor(c)
		if err != nil {
			t.Fatalf("deny db read: %v", err)
		}
		// grant read at table level (more specific wins)
		_, c, err = admin.Run(ctx,
			reql.DB(testDB).Table(allowedTable).Grant("tp_tbl_override", map[string]interface{}{"read": true}),
			nil,
		)
		closeCursor(c)
		if err != nil {
			t.Fatalf("grant table read: %v", err)
		}
		userExec := execAs(t, host, port, "tp_tbl_override", "pass")
		qCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// table-level grant overrides db denial
		_, c, err = userExec.Run(qCtx, reql.DB(testDB).Table(allowedTable).Count(), nil)
		closeCursor(c)
		if err != nil {
			t.Errorf("expected count success for table with explicit grant, got: %v", err)
		}

		// db-level denial still applies for restrictedTable
		_, c, err = userExec.Run(qCtx, reql.DB(testDB).Table(restrictedTable).Count(), nil)
		closeCursor(c)
		if !isPermissionError(err) {
			t.Errorf("expected ReqlPermissionError for non-granted table, got %v (type %T)", err, err)
		}
	})
}
