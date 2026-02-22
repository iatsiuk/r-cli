package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestStatusCmdRegistered(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	for _, sub := range root.Commands() {
		if sub.Name() == "status" {
			return
		}
	}
	t.Error("status subcommand not registered on root command")
}

func TestStatusCmdNoArgs(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newStatusCmd(cfg)
	if cmd.Args == nil {
		t.Error("status: expected Args validator")
	}
	if err := cmd.Args(cmd, []string{"extra"}); err == nil {
		t.Error("status: expected error for extra arg, got nil")
	}
	if err := cmd.Args(cmd, []string{}); err != nil {
		t.Errorf("status: expected no error for zero args, got %v", err)
	}
}

func TestCompletionBash(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"completion", "bash"})
	if err := root.Execute(); err != nil {
		t.Fatalf("completion bash: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "bash") {
		t.Errorf("completion bash: output does not look like bash completion script")
	}
}

func TestCompletionZsh(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"completion", "zsh"})
	if err := root.Execute(); err != nil {
		t.Fatalf("completion zsh: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "zsh") && !strings.Contains(out, "#compdef") {
		t.Errorf("completion zsh: output does not look like zsh completion script")
	}
}

func TestCompletionFish(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"completion", "fish"})
	if err := root.Execute(); err != nil {
		t.Fatalf("completion fish: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "fish") && !strings.Contains(out, "complete") {
		t.Errorf("completion fish: output does not look like fish completion script")
	}
}
