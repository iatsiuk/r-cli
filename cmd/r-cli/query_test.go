package main

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
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
	cfg := &rootConfig{}

	err := runQueryFile(cmd, cfg, path, true)
	if err == nil {
		t.Error("expected error, got nil")
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
	cfg := &rootConfig{}

	err := runQueryFile(cmd, cfg, path, false)
	if err == nil {
		t.Error("expected error, got nil")
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
