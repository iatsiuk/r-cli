package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"r-cli/internal/parselog"
)

func TestQueryCmdRegistered(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	for _, sub := range root.Commands() {
		if sub.Name() == "query" {
			return
		}
	}
	t.Error("query subcommand not registered on root command")
}

func TestQueryCmdUsage(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	for _, sub := range root.Commands() {
		if sub.Name() == "query" {
			if sub.Use != "query [expression]" {
				t.Errorf("query Use: got %q, want %q", sub.Use, "query [expression]")
			}
			return
		}
	}
	t.Error("query subcommand not found")
}

func TestQueryCmdFlags(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	for _, sub := range root.Commands() {
		if sub.Name() == "query" {
			if sub.Flags().Lookup("file") == nil {
				t.Error("query: --file flag not defined")
			}
			if sub.Flags().Lookup("stop-on-error") == nil {
				t.Error("query: --stop-on-error flag not defined")
			}
			return
		}
	}
	t.Error("query subcommand not found")
}

func TestRootHasDefaultRunE(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	if root.RunE == nil {
		t.Error("root command: RunE not set (default query mode disabled)")
	}
}

func TestReadQueryExprFromArg(t *testing.T) {
	t.Parallel()
	got, err := readQueryExpr([]string{`r.table("users")`}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != `r.table("users")` {
		t.Errorf("got %q, want %q", got, `r.table("users")`)
	}
}

func TestReadQueryExprFromStdin(t *testing.T) {
	t.Parallel()
	expr := `r.table("users")`
	got, err := readQueryExpr(nil, strings.NewReader(expr+"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got != expr {
		t.Errorf("got %q, want %q", got, expr)
	}
}

func TestReadQueryExprStdinTrimmed(t *testing.T) {
	t.Parallel()
	got, err := readQueryExpr(nil, strings.NewReader("  r.now()  \n"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "r.now()" {
		t.Errorf("got %q, want %q", got, "r.now()")
	}
}

func TestReadQueryExprArgWinsOverStdin(t *testing.T) {
	t.Parallel()
	got, err := readQueryExpr([]string{"r.now()"}, strings.NewReader("r.db()"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "r.now()" {
		t.Errorf("got %q, want %q", got, "r.now()")
	}
}

func TestRunQueryExprParseError(t *testing.T) {
	t.Parallel()
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cfg := &rootConfig{}
	err := runQueryExpr(cmd, cfg, "!!!invalid!!!")
	if err == nil {
		t.Error("expected parse error, got nil")
	}
	if !strings.Contains(err.Error(), "query:") {
		t.Errorf("error should contain 'query:' prefix, got: %v", err)
	}
}

func TestRunQueryExprLogsParseError(t *testing.T) {
	dir := t.TempDir()
	parselog.SetDir(dir)
	t.Cleanup(func() { parselog.SetDir(testLogDir) })
	parselog.SetVersion("test-ver")
	t.Cleanup(func() { parselog.SetVersion("") })

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cfg := &rootConfig{}
	_ = runQueryExpr(cmd, cfg, "!!!invalid!!!")

	data, err := os.ReadFile(filepath.Join(dir, "parser-errors.log"))
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	var entry struct {
		Ver  string `json:"ver"`
		Err  string `json:"err"`
		Expr string `json:"expr"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(data), &entry); err != nil {
		t.Fatalf("invalid JSONL: %v", err)
	}
	if entry.Ver != "test-ver" {
		t.Errorf("ver: got %q, want %q", entry.Ver, "test-ver")
	}
	if entry.Expr != "!!!invalid!!!" {
		t.Errorf("expr: got %q, want %q", entry.Expr, "!!!invalid!!!")
	}
	if entry.Err == "" {
		t.Error("err field is empty")
	}
}

func TestSplitQueriesEmpty(t *testing.T) {
	t.Parallel()
	got, err := splitQueries(strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("got %d queries, want 0", len(got))
	}
}

func TestSplitQueriesSingle(t *testing.T) {
	t.Parallel()
	got, err := splitQueries(strings.NewReader(`r.table("users")`))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d queries, want 1", len(got))
	}
	if got[0] != `r.table("users")` {
		t.Errorf("got %q, want %q", got[0], `r.table("users")`)
	}
}

func TestSplitQueriesMultiple(t *testing.T) {
	t.Parallel()
	input := "r.dbList()\n---\nr.tableList()\n---\nr.now()"
	got, err := splitQueries(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d queries, want 3: %v", len(got), got)
	}
	want := []string{"r.dbList()", "r.tableList()", "r.now()"}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("query[%d]: got %q, want %q", i, got[i], w)
		}
	}
}

func TestSplitQueriesTrailingSeparator(t *testing.T) {
	t.Parallel()
	input := "r.now()\n---\n"
	got, err := splitQueries(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d queries, want 1: %v", len(got), got)
	}
	if got[0] != "r.now()" {
		t.Errorf("got %q, want %q", got[0], "r.now()")
	}
}

func TestSplitQueriesLeadingSeparator(t *testing.T) {
	t.Parallel()
	input := "---\nr.now()"
	got, err := splitQueries(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d queries, want 1: %v", len(got), got)
	}
	if got[0] != "r.now()" {
		t.Errorf("got %q, want %q", got[0], "r.now()")
	}
}

func TestSplitQueriesWhitespaceSeparator(t *testing.T) {
	t.Parallel()
	input := "r.now()\n  ---  \nr.dbList()"
	got, err := splitQueries(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d queries, want 2: %v", len(got), got)
	}
	if got[0] != "r.now()" {
		t.Errorf("query[0]: got %q, want %q", got[0], "r.now()")
	}
	if got[1] != "r.dbList()" {
		t.Errorf("query[1]: got %q, want %q", got[1], "r.dbList()")
	}
}

func TestSplitQueriesMultilineQuery(t *testing.T) {
	t.Parallel()
	input := "r.db(\"test\")\n  .table(\"users\")\n---\nr.now()"
	got, err := splitQueries(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d queries, want 2: %v", len(got), got)
	}
	want0 := "r.db(\"test\")\n  .table(\"users\")"
	if got[0] != want0 {
		t.Errorf("query[0]: got %q, want %q", got[0], want0)
	}
	if got[1] != "r.now()" {
		t.Errorf("query[1]: got %q, want %q", got[1], "r.now()")
	}
}

func TestRunQueryFileNotFound(t *testing.T) {
	t.Parallel()
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cfg := &rootConfig{}
	err := runQueryFile(cmd, cfg, "/nonexistent/path/query.rql", false)
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "query:") {
		t.Errorf("error should contain 'query:' prefix, got: %v", err)
	}
}

func TestRunQueryFileStopOnError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := dir + "/queries.rql"
	if err := os.WriteFile(path, []byte("!!!bad1\n---\n!!!bad2"), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	var errBuf strings.Builder
	cmd.SetErr(&errBuf)
	cfg := &rootConfig{}

	err := runQueryFile(cmd, cfg, path, true)
	if err == nil {
		t.Error("expected error, got nil")
	}
	// stop-on-error returns immediately without printing to stderr
	if errBuf.Len() != 0 {
		t.Errorf("stop-on-error should not print to stderr, got: %q", errBuf.String())
	}
}

func TestRunQueryFileContinueOnError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := dir + "/queries.rql"
	if err := os.WriteFile(path, []byte("!!!bad1\n---\n!!!bad2"), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	var errBuf strings.Builder
	cmd.SetErr(&errBuf)
	cfg := &rootConfig{}

	err := runQueryFile(cmd, cfg, path, false)
	if err == nil {
		t.Error("expected error, got nil")
	}
	// continue mode prints each error to stderr and attempts all queries
	if count := strings.Count(errBuf.String(), "query error:"); count != 2 {
		t.Errorf("continue mode: expected 2 errors on stderr, got %d; stderr: %q", count, errBuf.String())
	}
}

func TestRunQueryFileNoQueries(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := dir + "/empty.rql"
	if err := os.WriteFile(path, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cfg := &rootConfig{}

	err := runQueryFile(cmd, cfg, path, false)
	if err != nil {
		t.Errorf("empty file: expected nil error, got: %v", err)
	}
}

func TestRunQueryFileStdinDash(t *testing.T) {
	t.Parallel()
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetIn(strings.NewReader("!!!bad_query"))
	cfg := &rootConfig{}

	err := runQueryFile(cmd, cfg, "-", false)
	// must get a parse/query error, not a file-open error
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if strings.Contains(err.Error(), "open -") {
		t.Errorf("runQueryFile('-') tried to open file named '-', want stdin read; error: %v", err)
	}
	if !strings.Contains(err.Error(), "query") {
		t.Errorf("expected query-related error, got: %v", err)
	}
}
