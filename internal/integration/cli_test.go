//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	"r-cli/internal/reql"
)

var (
	cliOnce sync.Once
	cliBin  string
	cliErr  error
	cliDir  string
)

func buildCLIBinary() (string, error) {
	cliOnce.Do(func() {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			cliErr = fmt.Errorf("cannot determine source path")
			return
		}
		root := filepath.Join(filepath.Dir(file), "../..")
		dir, err := os.MkdirTemp("", "r-cli-int-*")
		if err != nil {
			cliErr = fmt.Errorf("mktemp: %w", err)
			return
		}
		cliDir = dir
		bin := filepath.Join(dir, "r-cli")
		cmd := exec.Command("go", "build", "-o", bin, "./cmd/r-cli")
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			cliErr = fmt.Errorf("build r-cli: %w: %s", err, out)
			return
		}
		cliBin = bin
	})
	return cliBin, cliErr
}

// cliRun executes r-cli with args and optional stdin. Returns stdout, stderr, exit code.
func cliRun(t *testing.T, stdin string, args ...string) (string, string, int) {
	t.Helper()
	bin, err := buildCLIBinary()
	if err != nil {
		t.Fatalf("build cli: %v", err)
	}
	cmd := exec.Command(bin, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	runErr := cmd.Run()
	code := 0
	if runErr != nil {
		ee, ok := runErr.(*exec.ExitError)
		if !ok {
			t.Fatalf("exec r-cli: %v", runErr)
		}
		code = ee.ExitCode()
	}
	return outBuf.String(), errBuf.String(), code
}

// cliArgs prepends -H <host> -P <port> to extra args.
func cliArgs(extra ...string) []string {
	base := []string{"-H", containerHost, "-P", strconv.Itoa(containerPort)}
	return append(base, extra...)
}

func TestCLIDbListExpression(t *testing.T) {
	t.Parallel()
	stdout, _, code := cliRun(t, "", cliArgs("r.dbList()")...)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(stdout, "test") {
		t.Errorf("output %q does not contain 'test'", stdout)
	}
}

func TestCLIDbListSubcommand(t *testing.T) {
	t.Parallel()
	stdout, _, code := cliRun(t, "", cliArgs("db", "list")...)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(stdout, "test") {
		t.Errorf("output %q does not contain 'test'", stdout)
	}
}

func TestCLITableList(t *testing.T) {
	t.Parallel()
	qexec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	setupTestDB(t, qexec, dbName)
	stdout, _, code := cliRun(t, "", cliArgs("-d", dbName, "-f", "json", "table", "list")...)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	var arr []interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &arr); err != nil {
		t.Errorf("output is not valid JSON array: %v\noutput: %q", err, stdout)
	}
}

func TestCLIStatus(t *testing.T) {
	t.Parallel()
	qexec := newExecutor(t)
	info, err := qexec.ServerInfo(context.Background())
	if err != nil {
		t.Fatalf("server info: %v", err)
	}
	stdout, _, code := cliRun(t, "", cliArgs("status")...)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(stdout, info.Name) {
		t.Errorf("output %q does not contain server name %q", stdout, info.Name)
	}
}

func TestCLIFormatJSON(t *testing.T) {
	t.Parallel()
	stdout, _, code := cliRun(t, "", cliArgs("-f", "json", "r.dbList()")...)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	var v interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &v); err != nil {
		t.Errorf("output is not valid JSON: %v\noutput: %q", err, stdout)
	}
}

func TestCLIFormatTable(t *testing.T) {
	t.Parallel()
	qexec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	tableName := "items"
	setupTestDB(t, qexec, dbName)
	createTestTable(t, qexec, dbName, tableName)

	ctx := context.Background()
	docs := reql.Array(
		map[string]interface{}{"id": "1", "name": "alpha", "value": 10},
		map[string]interface{}{"id": "2", "name": "beta", "value": 20},
	)
	_, cur, err := qexec.Run(ctx, reql.DB(dbName).Table(tableName).Insert(docs), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("insert docs: %v", err)
	}

	expr := fmt.Sprintf(`r.db("%s").table("%s").limit(5)`, dbName, tableName)
	stdout, _, code := cliRun(t, "", cliArgs("-f", "table", expr)...)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(stdout, "+") || !strings.Contains(stdout, "|") {
		t.Errorf("output does not look like ASCII table: %q", stdout)
	}
}

func TestCLIFormatRaw(t *testing.T) {
	t.Parallel()
	stdout, _, code := cliRun(t, "", cliArgs("-f", "raw", "r.dbList()")...)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	// dbList returns SUCCESS_ATOM with an array value; raw format outputs it as compact JSON
	if !strings.Contains(stdout, "rethinkdb") {
		t.Errorf("output %q does not contain 'rethinkdb'", stdout)
	}
	// no pretty-printing (raw/compact output has no newlines inside the value)
	if strings.Contains(stdout, "  ") {
		t.Errorf("output appears to be pretty-printed: %q", stdout)
	}
}

func TestCLIBadHost(t *testing.T) {
	t.Parallel()
	port := strconv.Itoa(containerPort)
	args := []string{"-H", "badhost.invalid", "-P", port, "--timeout", "5s", "r.dbList()"}
	_, stderr, code := cliRun(t, "", args...)
	if code != 1 {
		t.Errorf("exit code %d, want 1", code)
	}
	if stderr == "" {
		t.Error("expected error message on stderr, got nothing")
	}
}

func TestCLIRunRaw(t *testing.T) {
	t.Parallel()
	// [59,[]] is raw ReQL JSON for r.dbList()
	stdout, _, code := cliRun(t, "", cliArgs("run", "[59,[]]")...)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(stdout, "rethinkdb") {
		t.Errorf("output %q does not contain 'rethinkdb'", stdout)
	}
}

func TestCLIStdin(t *testing.T) {
	t.Parallel()
	stdout, _, code := cliRun(t, "r.dbList()", cliArgs()...)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(stdout, "rethinkdb") {
		t.Errorf("output %q does not contain 'rethinkdb'", stdout)
	}
}

func TestCLIQueryFile(t *testing.T) {
	t.Parallel()
	f, err := os.CreateTemp(t.TempDir(), "test-*.reql")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := f.WriteString("r.dbList()"); err != nil {
		t.Fatalf("write query file: %v", err)
	}
	_ = f.Close()
	stdout, _, code := cliRun(t, "", cliArgs("query", "-F", f.Name())...)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(stdout, "rethinkdb") {
		t.Errorf("output %q does not contain 'rethinkdb'", stdout)
	}
}

func TestCLIDbRoundtrip(t *testing.T) {
	t.Parallel()
	dbName := sanitizeID(t.Name())
	_, _, code := cliRun(t, "", cliArgs("db", "create", dbName)...)
	if code != 0 {
		t.Fatalf("db create exit code %d", code)
	}
	t.Cleanup(func() {
		cliRun(t, "", cliArgs("db", "drop", dbName, "-y")...)
	})
	stdout, _, code := cliRun(t, "", cliArgs("db", "list")...)
	if code != 0 {
		t.Fatalf("db list exit code %d", code)
	}
	if !strings.Contains(stdout, dbName) {
		t.Errorf("db list %q does not contain %q", stdout, dbName)
	}
	_, _, code = cliRun(t, "", cliArgs("db", "drop", dbName, "-y")...)
	if code != 0 {
		t.Fatalf("db drop exit code %d", code)
	}
}

func TestCLITableRoundtrip(t *testing.T) {
	t.Parallel()
	qexec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	setupTestDB(t, qexec, dbName)
	tableName := "cli_tbl"
	_, _, code := cliRun(t, "", cliArgs("table", "create", tableName, "-d", dbName)...)
	if code != 0 {
		t.Fatalf("table create exit code %d", code)
	}
	stdout, _, code := cliRun(t, "", cliArgs("-d", dbName, "table", "list")...)
	if code != 0 {
		t.Fatalf("table list exit code %d", code)
	}
	if !strings.Contains(stdout, tableName) {
		t.Errorf("table list %q does not contain %q", stdout, tableName)
	}
	_, _, code = cliRun(t, "", cliArgs("table", "drop", tableName, "-d", dbName, "-y")...)
	if code != 0 {
		t.Fatalf("table drop exit code %d", code)
	}
}
