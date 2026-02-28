//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"r-cli/internal/reql"
	"r-cli/internal/response"
)

func TestTableCreate(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	tableName := "tbl"

	_, cur, err := exec.Run(ctx, reql.DB(dbName).TableCreate(tableName), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("table create: %v", err)
	}

	_, cur2, err := exec.Run(ctx, reql.DB(dbName).TableList(), nil)
	if err != nil {
		t.Fatalf("table list: %v", err)
	}
	names := cursorStrings(t, cur2)

	if !strContains(names, tableName) {
		t.Errorf("table list does not include %q after create, got %v", tableName, names)
	}
}

func TestTableCreatePrimaryKey(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	tableName := "tbl"

	_, cur, err := exec.Run(ctx, reql.DB(dbName).TableCreate(tableName, reql.OptArgs{"primary_key": "email"}), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("table create with primary_key: %v", err)
	}

	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table(tableName).Config(), nil)
	if err != nil {
		t.Fatalf("table config: %v", err)
	}
	defer closeCursor(cur2)

	raw, err := cur2.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}

	var cfg struct {
		PrimaryKey string `json:"primary_key"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if cfg.PrimaryKey != "email" {
		t.Errorf("primary_key=%q, want %q", cfg.PrimaryKey, "email")
	}
}

func TestTableCreateDuplicate(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	tableName := "tbl"
	createTestTable(t, exec, dbName, tableName)

	_, cur, err := exec.Run(ctx, reql.DB(dbName).TableCreate(tableName), nil)
	closeCursor(cur)
	if err == nil {
		t.Fatal("expected error for duplicate table create, got nil")
	}
	var runtimeErr *response.ReqlRuntimeError
	if !errors.As(err, &runtimeErr) {
		t.Errorf("expected ReqlRuntimeError, got %T: %v", err, err)
	}
}

func TestTableDrop(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	tableName := "tbl"

	_, cur, err := exec.Run(ctx, reql.DB(dbName).TableCreate(tableName), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("table create: %v", err)
	}

	_, cur2, err := exec.Run(ctx, reql.DB(dbName).TableDrop(tableName), nil)
	closeCursor(cur2)
	if err != nil {
		t.Fatalf("table drop: %v", err)
	}

	_, cur3, err := exec.Run(ctx, reql.DB(dbName).TableList(), nil)
	if err != nil {
		t.Fatalf("table list: %v", err)
	}
	names := cursorStrings(t, cur3)

	if strContains(names, tableName) {
		t.Errorf("table list still contains %q after drop", tableName)
	}
}

func TestTableDropNonexistent(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)

	_, cur, err := exec.Run(ctx, reql.DB(dbName).TableDrop("nonexistent_tbl"), nil)
	closeCursor(cur)
	if err == nil {
		t.Fatal("expected error for dropping nonexistent table, got nil")
	}
	var runtimeErr *response.ReqlRuntimeError
	if !errors.As(err, &runtimeErr) {
		t.Errorf("expected ReqlRuntimeError, got %T: %v", err, err)
	}
}

func TestTableConfig(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	tableName := "tbl"
	createTestTable(t, exec, dbName, tableName)

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table(tableName).Config(), nil)
	if err != nil {
		t.Fatalf("table config: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}

	var cfg struct {
		ID         string        `json:"id"`
		Name       string        `json:"name"`
		DB         string        `json:"db"`
		PrimaryKey string        `json:"primary_key"`
		Shards     []interface{} `json:"shards"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if cfg.ID == "" {
		t.Error("config.id is empty")
	}
	if cfg.Name != tableName {
		t.Errorf("config.name=%q, want %q", cfg.Name, tableName)
	}
	if cfg.DB != dbName {
		t.Errorf("config.db=%q, want %q", cfg.DB, dbName)
	}
	if cfg.PrimaryKey == "" {
		t.Error("config.primary_key is empty")
	}
	if len(cfg.Shards) == 0 {
		t.Error("config.shards is empty")
	}
}

func TestTableStatus(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	tableName := "tbl"
	createTestTable(t, exec, dbName, tableName)

	// wait for the table to be ready
	_, wCur, err := exec.Run(ctx, reql.DB(dbName).Table(tableName).Wait(), nil)
	closeCursor(wCur)
	if err != nil {
		t.Fatalf("table wait: %v", err)
	}

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table(tableName).Status(), nil)
	if err != nil {
		t.Fatalf("table status: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}

	var status struct {
		Status struct {
			AllReplicasReady bool `json:"all_replicas_ready"`
		} `json:"status"`
	}
	if err := json.Unmarshal(raw, &status); err != nil {
		t.Fatalf("unmarshal status: %v", err)
	}
	if !status.Status.AllReplicasReady {
		t.Error("status.all_replicas_ready is false")
	}
}

func TestTableSync(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	tableName := "tbl"
	createTestTable(t, exec, dbName, tableName)

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table(tableName).Sync(), nil)
	if err != nil {
		t.Fatalf("table sync: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}

	var result struct {
		Synced int `json:"synced"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal sync result: %v", err)
	}
	if result.Synced != 1 {
		t.Errorf("synced=%d, want 1", result.Synced)
	}
}

func TestTableReconfigure(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	tableName := "tbl"
	createTestTable(t, exec, dbName, tableName)

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table(tableName).Reconfigure(reql.OptArgs{"shards": 1, "replicas": 1}), nil)
	if err != nil {
		t.Fatalf("table reconfigure: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}

	var result struct {
		Reconfigured int `json:"reconfigured"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal reconfigure result: %v", err)
	}
	if result.Reconfigured < 0 {
		t.Errorf("reconfigured=%d, want >= 0", result.Reconfigured)
	}
}

func TestTableRebalance(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	tableName := "tbl"
	createTestTable(t, exec, dbName, tableName)

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table(tableName).Rebalance(), nil)
	if err != nil {
		t.Fatalf("table rebalance: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}

	var result struct {
		Rebalanced int `json:"rebalanced"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal rebalance result: %v", err)
	}
	if result.Rebalanced < 0 {
		t.Errorf("rebalanced=%d, want >= 0", result.Rebalanced)
	}
}
