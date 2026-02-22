//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"testing"

	"r-cli/internal/cursor"
	"r-cli/internal/reql"
	"r-cli/internal/response"
)

// uuidRe matches standard UUID format: 8-4-4-4-12 lowercase hex.
var uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func TestServerInfoName(t *testing.T) {
	t.Parallel()
	exec, cleanup := newExecutor()
	defer cleanup()

	info, err := exec.ServerInfo(context.Background())
	if err != nil {
		t.Fatalf("server info: %v", err)
	}
	if info.Name == "" {
		t.Error("server name is empty")
	}
	if info.ID == "" {
		t.Error("server id is empty")
	}
}

func TestServerInfoUUID(t *testing.T) {
	t.Parallel()
	exec, cleanup := newExecutor()
	defer cleanup()

	info, err := exec.ServerInfo(context.Background())
	if err != nil {
		t.Fatalf("server info: %v", err)
	}
	if !uuidRe.MatchString(info.ID) {
		t.Errorf("server id %q is not a valid UUID", info.ID)
	}
}

func TestDatabaseList(t *testing.T) {
	t.Parallel()
	exec, cleanup := newExecutor()
	defer cleanup()

	_, cur, err := exec.Run(context.Background(), reql.DBList(), nil)
	if err != nil {
		t.Fatalf("db list: %v", err)
	}
	names := cursorStrings(t, cur)

	if !strContains(names, "rethinkdb") {
		t.Errorf("db list missing 'rethinkdb', got %v", names)
	}
	if !strContains(names, "test") {
		t.Errorf("db list missing 'test', got %v", names)
	}
}

func TestDatabaseCreate(t *testing.T) {
	t.Parallel()
	exec, cleanup := newExecutor()
	defer cleanup()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)

	_, cur, err := exec.Run(context.Background(), reql.DBList(), nil)
	if err != nil {
		t.Fatalf("db list: %v", err)
	}
	names := cursorStrings(t, cur)

	if !strContains(names, dbName) {
		t.Errorf("db list does not include %q after create", dbName)
	}
}

func TestDatabaseCreateDuplicate(t *testing.T) {
	t.Parallel()
	exec, cleanup := newExecutor()
	defer cleanup()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)

	_, cur, err := exec.Run(context.Background(), reql.DBCreate(dbName), nil)
	closeCursor(cur)
	if err == nil {
		t.Fatal("expected error for duplicate db create, got nil")
	}
	var runtimeErr *response.ReqlRuntimeError
	if !errors.As(err, &runtimeErr) {
		t.Errorf("expected ReqlRuntimeError, got %T: %v", err, err)
	}
}

func TestDatabaseDrop(t *testing.T) {
	t.Parallel()
	exec, cleanup := newExecutor()
	defer cleanup()

	ctx := context.Background()
	dbName := sanitizeID(t.Name())

	_, cur, err := exec.Run(ctx, reql.DBCreate(dbName), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("create db: %v", err)
	}

	_, cur2, err := exec.Run(ctx, reql.DBDrop(dbName), nil)
	closeCursor(cur2)
	if err != nil {
		t.Fatalf("drop db: %v", err)
	}

	_, cur3, err := exec.Run(ctx, reql.DBList(), nil)
	if err != nil {
		t.Fatalf("db list: %v", err)
	}
	names := cursorStrings(t, cur3)

	if strContains(names, dbName) {
		t.Errorf("db list still contains %q after drop", dbName)
	}
}

func TestDatabaseDropNonexistent(t *testing.T) {
	t.Parallel()
	exec, cleanup := newExecutor()
	defer cleanup()

	_, cur, err := exec.Run(context.Background(), reql.DBDrop("nonexistent_zzz_xyz"), nil)
	closeCursor(cur)
	if err == nil {
		t.Fatal("expected error for dropping nonexistent db, got nil")
	}
	var runtimeErr *response.ReqlRuntimeError
	if !errors.As(err, &runtimeErr) {
		t.Errorf("expected ReqlRuntimeError, got %T: %v", err, err)
	}
}

// sanitizeID converts a test name to a valid RethinkDB identifier (max 62 chars).
func sanitizeID(name string) string {
	s := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '_'
	}, name)
	if len(s) > 62 {
		s = s[:62]
	}
	return s
}

// strContains reports whether target is in names.
func strContains(names []string, target string) bool {
	for _, n := range names {
		if n == target {
			return true
		}
	}
	return false
}

// cursorStrings reads the first item from cur and unmarshals it as []string.
// Used for queries that return a SUCCESS_ATOM containing a JSON array of strings
// (e.g. dbList(), tableList()).
func cursorStrings(t *testing.T, cur cursor.Cursor) []string {
	t.Helper()
	defer closeCursor(cur)
	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var names []string
	if err := json.Unmarshal(raw, &names); err != nil {
		t.Fatalf("unmarshal string list: %v", err)
	}
	return names
}
