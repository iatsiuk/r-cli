//go:build integration

package integration

import (
	"os"
	"strings"
	"testing"
)

func TestREPLQueryViaPipe(t *testing.T) {
	t.Parallel()
	// pipe query to `query` subcommand via stdin (distinct from root command stdin mode)
	stdout, _, code := cliRun(t, "r.dbList()", cliArgs("query")...)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(stdout, "rethinkdb") {
		t.Errorf("output %q does not contain 'rethinkdb'", stdout)
	}
}

func TestREPLMultipleQueriesFile(t *testing.T) {
	t.Parallel()
	// create a file with two queries separated by "---"
	f, err := os.CreateTemp(t.TempDir(), "multi-*.reql")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	content := "r.dbList()\n---\nr.expr(42)\n"
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write query file: %v", err)
	}
	_ = f.Close()

	stdout, _, code := cliRun(t, "", cliArgs("query", "-F", f.Name())...)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	// first query: dbList contains "rethinkdb"
	if !strings.Contains(stdout, "rethinkdb") {
		t.Errorf("output %q does not contain 'rethinkdb' (first query result)", stdout)
	}
	// second query: r.expr(42) outputs 42
	if !strings.Contains(stdout, "42") {
		t.Errorf("output %q does not contain '42' (second query result)", stdout)
	}
}
