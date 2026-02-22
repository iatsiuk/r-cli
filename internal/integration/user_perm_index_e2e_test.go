//go:build integration

package integration

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"r-cli/internal/reql"
)

func TestUserE2E(t *testing.T) {
	t.Parallel()
	// uses shared container: admin has no password, so no -p flag needed.
	// the local --password flag on user create does not conflict with the
	// persistent --password/-p flag because we are not passing -p here.
	username := sanitizeID(t.Name())

	_, stderr, code := cliRun(t, "", cliArgs("user", "create", username, "--password", "pass123")...)
	if code != 0 {
		t.Fatalf("user create: exit code %d, stderr: %s", code, stderr)
	}
	t.Cleanup(func() {
		cliRun(t, "", cliArgs("user", "delete", username, "-y")...)
	})

	// list: user appears
	stdout, _, code := cliRun(t, "", cliArgs("-f", "json", "user", "list")...)
	if code != 0 {
		t.Fatalf("user list: exit code %d", code)
	}
	if !strings.Contains(stdout, username) {
		t.Errorf("user list %q does not contain %q", stdout, username)
	}

	// delete user
	_, _, code = cliRun(t, "", cliArgs("user", "delete", username, "-y")...)
	if code != 0 {
		t.Fatalf("user delete: exit code %d", code)
	}

	// list: user removed
	stdout, _, code = cliRun(t, "", cliArgs("-f", "json", "user", "list")...)
	if code != 0 {
		t.Fatalf("user list after delete: exit code %d", code)
	}
	if strings.Contains(stdout, username) {
		t.Errorf("user list %q still contains %q after delete", stdout, username)
	}
}

func TestGrantE2E(t *testing.T) {
	t.Parallel()
	// uses shared container: create user, grant db-level read, verify via CLI and driver.
	qexec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	const tableName = "items"
	username := sanitizeID(t.Name() + "user")

	setupTestDB(t, qexec, dbName)
	createTestTable(t, qexec, dbName, tableName)

	// create user via CLI (no connection password needed)
	_, stderr, code := cliRun(t, "", cliArgs("user", "create", username, "--password", "userpass")...)
	if code != 0 {
		t.Fatalf("user create: exit code %d, stderr: %s", code, stderr)
	}
	t.Cleanup(func() {
		cliRun(t, "", cliArgs("user", "delete", username, "-y")...)
	})

	// grant read on testdb via CLI
	_, stderr, code = cliRun(t, "", cliArgs("grant", username, "--read", "--db", dbName)...)
	if code != 0 {
		t.Fatalf("grant read: exit code %d, stderr: %s", code, stderr)
	}

	// verify via driver: user can count the table
	userExec := execAs(t, containerHost, containerPort, username, "userpass")
	_, cur, err := userExec.Run(ctx, reql.DB(dbName).Table(tableName).Count(), nil)
	closeCursor(cur)
	if err != nil {
		t.Errorf("user count after db-level grant: %v", err)
	}

	// also verify via CLI: root query command uses -p flag which has no local conflict
	userCLIArgs := []string{
		"-H", containerHost,
		"-P", strconv.Itoa(containerPort),
		"-u", username,
		"-p", "userpass",
	}
	expr := `r.db("` + dbName + `").table("` + tableName + `").count()`
	_, stderr, code = cliRun(t, "", append(userCLIArgs, expr)...)
	if code != 0 {
		t.Errorf("user CLI query after db-level grant: exit code %d, stderr: %s", code, stderr)
	}
}

func TestIndexE2E(t *testing.T) {
	t.Parallel()
	qexec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	tableName := "idxe2e_items"
	const indexName = "by_value"

	setupTestDB(t, qexec, dbName)
	createTestTable(t, qexec, dbName, tableName)

	args := func(extra ...string) []string {
		return append(cliArgs("-d", dbName), extra...)
	}

	// create index
	_, stderr, code := cliRun(t, "", args("index", "create", tableName, indexName)...)
	if code != 0 {
		t.Fatalf("index create: exit code %d, stderr: %s", code, stderr)
	}

	// list: index appears
	stdout, _, code := cliRun(t, "", args("-f", "json", "index", "list", tableName)...)
	if code != 0 {
		t.Fatalf("index list: exit code %d", code)
	}
	if !strings.Contains(stdout, indexName) {
		t.Errorf("index list %q does not contain %q", stdout, indexName)
	}

	// wait for index ready
	_, _, code = cliRun(t, "", args("index", "wait", tableName, indexName)...)
	if code != 0 {
		t.Fatalf("index wait: exit code %d", code)
	}

	// drop index
	_, _, code = cliRun(t, "", args("index", "drop", tableName, indexName)...)
	if code != 0 {
		t.Fatalf("index drop: exit code %d", code)
	}

	// list: index removed
	stdout, _, code = cliRun(t, "", args("-f", "json", "index", "list", tableName)...)
	if code != 0 {
		t.Fatalf("index list after drop: exit code %d", code)
	}
	if strings.Contains(stdout, indexName) {
		t.Errorf("index list %q still contains %q after drop", stdout, indexName)
	}
}
