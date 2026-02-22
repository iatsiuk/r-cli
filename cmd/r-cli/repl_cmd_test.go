package main

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestReplCmdRegistered(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	for _, sub := range root.Commands() {
		if sub.Name() == "repl" {
			return
		}
	}
	t.Error("repl subcommand not registered on root command")
}

func TestReplCmdUsage(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	for _, sub := range root.Commands() {
		if sub.Name() == "repl" {
			if sub.Use != "repl" {
				t.Errorf("repl Use: got %q, want %q", sub.Use, "repl")
			}
			return
		}
	}
	t.Error("repl subcommand not found")
}

func TestReplCmdRejectsArgs(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	root.SetArgs([]string{"repl", "extra-arg"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	if err := root.Execute(); err == nil {
		t.Error("expected error when passing args to repl command, got nil")
	}
}

// TestRootNoArgsTTYStartsREPL verifies that the root command starts the REPL
// when no args are given and stdin is a TTY.
func TestRootNoArgsTTYStartsREPL(t *testing.T) {
	oldTTY := stdinIsTTY
	stdinIsTTY = func() bool { return true }
	defer func() { stdinIsTTY = oldTTY }()

	started := false
	oldStart := replStart
	replStart = func(_ context.Context, _ *rootConfig, _, _ io.Writer) error {
		started = true
		return nil
	}
	defer func() { replStart = oldStart }()

	root := buildRootCmd(&rootConfig{})
	root.SetArgs([]string{})
	root.SetIn(strings.NewReader(""))
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !started {
		t.Error("REPL not started when stdin is TTY and no args given")
	}
}

// TestRootNoArgsNonTTYReadsStdin verifies that without a TTY, the root command
// reads a query from stdin instead of starting the REPL.
func TestRootNoArgsNonTTYReadsStdin(t *testing.T) {
	t.Parallel()
	// stdinIsTTY returns false in normal test environment (no real TTY)
	root := buildRootCmd(&rootConfig{})
	root.SetArgs([]string{})
	root.SetIn(strings.NewReader("!!!invalid!!!"))
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	err := root.Execute()
	// expects a parse error because we read from stdin, not REPL mode
	if err == nil {
		t.Error("expected parse error for invalid stdin query, got nil")
	}
}

// TestReplCmdStartsREPL verifies that `r-cli repl` invokes the REPL runner.
func TestReplCmdStartsREPL(t *testing.T) {
	started := false
	oldStart := replStart
	replStart = func(_ context.Context, _ *rootConfig, _, _ io.Writer) error {
		started = true
		return nil
	}
	defer func() { replStart = oldStart }()

	root := buildRootCmd(&rootConfig{})
	root.SetArgs([]string{"repl"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !started {
		t.Error("REPL not started via 'repl' subcommand")
	}
}

// TestReplInheritsGlobalFlags verifies that the repl command inherits
// the persistent connection flags from the root command.
func TestReplInheritsGlobalFlags(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	var replCmd *cobra.Command
	for _, sub := range root.Commands() {
		if sub.Name() == "repl" {
			replCmd = sub
			break
		}
	}
	if replCmd == nil {
		t.Fatal("repl subcommand not found")
	}
	for _, flag := range []string{"host", "port", "db", "user", "password"} {
		if replCmd.InheritedFlags().Lookup(flag) == nil {
			t.Errorf("repl cmd: --%s flag not inherited from root", flag)
		}
	}
}

func TestReplHistoryFileContainsName(t *testing.T) {
	t.Parallel()
	path := replHistoryFile()
	if path != "" && !strings.HasSuffix(path, ".r-cli_history") {
		t.Errorf("replHistoryFile: got %q, want path ending with .r-cli_history", path)
	}
}

func TestMakeReplExecParseError(t *testing.T) {
	t.Parallel()
	// makeReplExec should propagate parser errors without attempting connection
	cfg := &rootConfig{}
	execFn := makeReplExec(nil, cfg)
	err := execFn(context.Background(), "!!!invalid!!!", io.Discard)
	if err == nil {
		t.Error("expected parse error for invalid expression, got nil")
	}
}

func TestJsonRowsToStrings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		rows []string
		want []string
	}{
		{"empty", nil, nil},
		{"strings", []string{`"foo"`, `"bar"`}, []string{"foo", "bar"}},
		{"skip non-string", []string{`"ok"`, `123`, `"good"`}, []string{"ok", "good"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var rows []json.RawMessage
			for _, r := range tc.rows {
				rows = append(rows, json.RawMessage(r))
			}
			got := jsonRowsToStrings(rows)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("[%d]: got %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}
