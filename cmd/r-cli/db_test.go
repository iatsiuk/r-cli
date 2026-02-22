package main

import (
	"strings"
	"testing"
)

func TestDBCmdRegistered(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	for _, sub := range root.Commands() {
		if sub.Name() == "db" {
			return
		}
	}
	t.Error("db subcommand not registered on root command")
}

func TestDBSubcommands(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	for _, sub := range root.Commands() {
		if sub.Name() != "db" {
			continue
		}
		want := map[string]bool{"list": false, "create": false, "drop": false}
		for _, s := range sub.Commands() {
			want[s.Name()] = true
		}
		for name, found := range want {
			if !found {
				t.Errorf("db %s subcommand not found", name)
			}
		}
		return
	}
	t.Error("db subcommand not found on root command")
}

func TestDBListNoArgs(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newDBListCmd(cfg)
	if cmd.Args == nil {
		t.Error("db list: expected Args validator")
	}
	if err := cmd.Args(cmd, []string{"extra"}); err == nil {
		t.Error("db list: expected error for extra arg, got nil")
	}
	if err := cmd.Args(cmd, []string{}); err != nil {
		t.Errorf("db list: expected no error for zero args, got %v", err)
	}
}

func TestDBCreateExactArgs(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newDBCreateCmd(cfg)
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("db create: expected error for zero args, got nil")
	}
	if err := cmd.Args(cmd, []string{"mydb"}); err != nil {
		t.Errorf("db create: expected no error for one arg, got %v", err)
	}
	if err := cmd.Args(cmd, []string{"mydb", "extra"}); err == nil {
		t.Error("db create: expected error for two args, got nil")
	}
}

func TestDBDropExactArgs(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newDBDropCmd(cfg)
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("db drop: expected error for zero args, got nil")
	}
	if err := cmd.Args(cmd, []string{"mydb"}); err != nil {
		t.Errorf("db drop: expected no error for one arg, got %v", err)
	}
	if err := cmd.Args(cmd, []string{"mydb", "extra"}); err == nil {
		t.Error("db drop: expected error for two args, got nil")
	}
}

func TestDBDropYesFlag(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newDBDropCmd(cfg)
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

func TestDBDropYesFlagShorthand(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newDBDropCmd(cfg)
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

func TestConfirmDropYes(t *testing.T) {
	t.Parallel()
	for _, input := range []string{"y", "Y", "yes", "YES", "Yes"} {
		if err := confirmDrop("database", "mydb", strings.NewReader(input)); err != nil {
			t.Errorf("confirmDrop %q: expected nil, got %v", input, err)
		}
	}
}

func TestConfirmDropNo(t *testing.T) {
	t.Parallel()
	for _, input := range []string{"n", "N", "no", ""} {
		if err := confirmDrop("database", "mydb", strings.NewReader(input)); err == nil {
			t.Errorf("confirmDrop %q: expected error, got nil", input)
		}
	}
}
