package main

import (
	"testing"
)

func TestTableCmdRegistered(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	for _, sub := range root.Commands() {
		if sub.Name() == "table" {
			return
		}
	}
	t.Error("table subcommand not registered on root command")
}

func TestTableSubcommands(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	for _, sub := range root.Commands() {
		if sub.Name() != "table" {
			continue
		}
		want := map[string]bool{"list": false, "create": false, "drop": false, "info": false}
		for _, s := range sub.Commands() {
			want[s.Name()] = true
		}
		for name, found := range want {
			if !found {
				t.Errorf("table %s subcommand not found", name)
			}
		}
		return
	}
	t.Error("table subcommand not found on root command")
}

func TestTableListNoArgs(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{database: "test"}
	cmd := newTableListCmd(cfg)
	if cmd.Args == nil {
		t.Error("table list: expected Args validator")
	}
	if err := cmd.Args(cmd, []string{"extra"}); err == nil {
		t.Error("table list: expected error for extra arg, got nil")
	}
	if err := cmd.Args(cmd, []string{}); err != nil {
		t.Errorf("table list: expected no error for zero args, got %v", err)
	}
}

func TestTableCreateExactArgs(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{database: "test"}
	cmd := newTableCreateCmd(cfg)
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("table create: expected error for zero args, got nil")
	}
	if err := cmd.Args(cmd, []string{"users"}); err != nil {
		t.Errorf("table create: expected no error for one arg, got %v", err)
	}
	if err := cmd.Args(cmd, []string{"users", "extra"}); err == nil {
		t.Error("table create: expected error for two args, got nil")
	}
}

func TestTableDropExactArgs(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{database: "test"}
	cmd := newTableDropCmd(cfg)
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("table drop: expected error for zero args, got nil")
	}
	if err := cmd.Args(cmd, []string{"users"}); err != nil {
		t.Errorf("table drop: expected no error for one arg, got %v", err)
	}
	if err := cmd.Args(cmd, []string{"users", "extra"}); err == nil {
		t.Error("table drop: expected error for two args, got nil")
	}
}

func TestTableInfoExactArgs(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{database: "test"}
	cmd := newTableInfoCmd(cfg)
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("table info: expected error for zero args, got nil")
	}
	if err := cmd.Args(cmd, []string{"users"}); err != nil {
		t.Errorf("table info: expected no error for one arg, got %v", err)
	}
	if err := cmd.Args(cmd, []string{"users", "extra"}); err == nil {
		t.Error("table info: expected error for two args, got nil")
	}
}

func TestTableDropYesFlag(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{database: "test"}
	cmd := newTableDropCmd(cfg)
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

func TestTableDropYesFlagShorthand(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{database: "test"}
	cmd := newTableDropCmd(cfg)
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

func TestTableDBRequiresDatabase(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	_, err := tableDB(cfg)
	if err == nil {
		t.Error("tableDB: expected error when database is empty, got nil")
	}
}

func TestTableDBReturnsTerm(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{database: "mydb"}
	term, err := tableDB(cfg)
	if err != nil {
		t.Fatalf("tableDB: unexpected error: %v", err)
	}
	// verify term serializes correctly as DB("mydb")
	data, err := term.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: unexpected error: %v", err)
	}
	got := string(data)
	want := `[14,["mydb"]]`
	if got != want {
		t.Errorf("tableDB term: got %s, want %s", got, want)
	}
}
