package main

import (
	"errors"
	"strings"
	"testing"
)

func TestUserCmdRegistered(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	for _, sub := range root.Commands() {
		if sub.Name() == "user" {
			return
		}
	}
	t.Error("user subcommand not registered on root command")
}

func TestUserSubcommands(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	for _, sub := range root.Commands() {
		if sub.Name() != "user" {
			continue
		}
		want := map[string]bool{
			"list":         false,
			"create":       false,
			"delete":       false,
			"set-password": false,
		}
		for _, s := range sub.Commands() {
			want[s.Name()] = true
		}
		for name, found := range want {
			if !found {
				t.Errorf("user %s subcommand not found", name)
			}
		}
		return
	}
	t.Error("user subcommand not found on root command")
}

func TestUserListNoArgs(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newUserListCmd(cfg)
	if err := cmd.Args(cmd, []string{}); err != nil {
		t.Errorf("user list: expected no error for zero args, got %v", err)
	}
	if err := cmd.Args(cmd, []string{"extra"}); err == nil {
		t.Error("user list: expected error for extra arg, got nil")
	}
}

func TestUserCreateExactArgs(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newUserCreateCmd(cfg)
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("user create: expected error for zero args, got nil")
	}
	if err := cmd.Args(cmd, []string{"alice"}); err != nil {
		t.Errorf("user create: expected no error for one arg, got %v", err)
	}
	if err := cmd.Args(cmd, []string{"alice", "extra"}); err == nil {
		t.Error("user create: expected error for two args, got nil")
	}
}

func TestUserCreatePasswordFlag(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newUserCreateCmd(cfg)
	if err := cmd.ParseFlags([]string{"--new-password", "secret"}); err != nil {
		t.Fatal(err)
	}
	pwd, err := cmd.Flags().GetString("new-password")
	if err != nil {
		t.Fatal(err)
	}
	if pwd != "secret" {
		t.Errorf("--password flag: got %q, want %q", pwd, "secret")
	}
}

func TestUserCreatePromptNoEcho(t *testing.T) {
	t.Parallel()
	// simulate non-TTY stdin: promptPassword reads a line from r
	got, err := promptPassword(&strings.Builder{}, strings.NewReader("mypassword\n"))
	if err != nil {
		t.Fatalf("promptPassword: unexpected error: %v", err)
	}
	if got != "mypassword" {
		t.Errorf("promptPassword: got %q, want %q", got, "mypassword")
	}
}

func TestUserDeleteExactArgs(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newUserDeleteCmd(cfg)
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("user delete: expected error for zero args, got nil")
	}
	if err := cmd.Args(cmd, []string{"alice"}); err != nil {
		t.Errorf("user delete: expected no error for one arg, got %v", err)
	}
	if err := cmd.Args(cmd, []string{"alice", "extra"}); err == nil {
		t.Error("user delete: expected error for two args, got nil")
	}
}

func TestUserDeleteYesFlag(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newUserDeleteCmd(cfg)
	if err := cmd.ParseFlags([]string{"--yes"}); err != nil {
		t.Fatal(err)
	}
	yes, err := cmd.Flags().GetBool("yes")
	if err != nil {
		t.Fatal(err)
	}
	if !yes {
		t.Error("--yes flag: expected true")
	}
}

func TestUserDeleteYesFlagShorthand(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newUserDeleteCmd(cfg)
	if err := cmd.ParseFlags([]string{"-y"}); err != nil {
		t.Fatal(err)
	}
	yes, err := cmd.Flags().GetBool("yes")
	if err != nil {
		t.Fatal(err)
	}
	if !yes {
		t.Error("-y flag: expected true")
	}
}

func TestUserDeleteConfirmation(t *testing.T) {
	t.Parallel()
	// confirmDrop is reused for user delete
	if err := confirmDrop("user", "alice", strings.NewReader("y"), false); err != nil {
		t.Errorf("confirmDrop user: expected nil for 'y', got %v", err)
	}
	if err := confirmDrop("user", "alice", strings.NewReader("n"), false); !errors.Is(err, errAborted) {
		t.Errorf("confirmDrop user: expected errAborted for 'n', got %v", err)
	}
}

func TestUserSetPasswordExactArgs(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newUserSetPasswordCmd(cfg)
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("user set-password: expected error for zero args, got nil")
	}
	if err := cmd.Args(cmd, []string{"alice"}); err != nil {
		t.Errorf("user set-password: expected no error for one arg, got %v", err)
	}
	if err := cmd.Args(cmd, []string{"alice", "extra"}); err == nil {
		t.Error("user set-password: expected error for two args, got nil")
	}
}

func TestUserSetPasswordPrompt(t *testing.T) {
	t.Parallel()
	// simulate non-TTY stdin
	got, err := promptPassword(&strings.Builder{}, strings.NewReader("newpwd\n"))
	if err != nil {
		t.Fatalf("promptPassword: unexpected error: %v", err)
	}
	if got != "newpwd" {
		t.Errorf("promptPassword: got %q, want %q", got, "newpwd")
	}
}
