package main

import (
	"testing"
)

func TestIndexCmdRegistered(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	for _, sub := range root.Commands() {
		if sub.Name() == "index" {
			return
		}
	}
	t.Error("index subcommand not registered on root command")
}

func TestIndexSubcommands(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	for _, sub := range root.Commands() {
		if sub.Name() != "index" {
			continue
		}
		want := map[string]bool{
			"list":   false,
			"create": false,
			"drop":   false,
			"rename": false,
			"status": false,
			"wait":   false,
		}
		for _, s := range sub.Commands() {
			want[s.Name()] = true
		}
		for name, found := range want {
			if !found {
				t.Errorf("index %s subcommand not found", name)
			}
		}
		return
	}
	t.Error("index subcommand not found on root command")
}

func TestIndexListExactArgs(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{database: "test"}
	cmd := newIndexListCmd(cfg)
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("index list: expected error for zero args, got nil")
	}
	if err := cmd.Args(cmd, []string{"mytable"}); err != nil {
		t.Errorf("index list: expected no error for one arg, got %v", err)
	}
	if err := cmd.Args(cmd, []string{"mytable", "extra"}); err == nil {
		t.Error("index list: expected error for two args, got nil")
	}
}

func TestIndexCreateExactArgs(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{database: "test"}
	cmd := newIndexCreateCmd(cfg)
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("index create: expected error for zero args, got nil")
	}
	if err := cmd.Args(cmd, []string{"mytable"}); err == nil {
		t.Error("index create: expected error for one arg, got nil")
	}
	if err := cmd.Args(cmd, []string{"mytable", "myindex"}); err != nil {
		t.Errorf("index create: expected no error for two args, got %v", err)
	}
	if err := cmd.Args(cmd, []string{"mytable", "myindex", "extra"}); err == nil {
		t.Error("index create: expected error for three args, got nil")
	}
}

func TestIndexCreateGeoFlag(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{database: "test"}
	cmd := newIndexCreateCmd(cfg)
	if err := cmd.ParseFlags([]string{"--geo"}); err != nil {
		t.Fatal(err)
	}
	geo, err := cmd.Flags().GetBool("geo")
	if err != nil {
		t.Fatal(err)
	}
	if !geo {
		t.Error("--geo flag: expected true")
	}
}

func TestIndexCreateMultiFlag(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{database: "test"}
	cmd := newIndexCreateCmd(cfg)
	if err := cmd.ParseFlags([]string{"--multi"}); err != nil {
		t.Fatal(err)
	}
	multi, err := cmd.Flags().GetBool("multi")
	if err != nil {
		t.Fatal(err)
	}
	if !multi {
		t.Error("--multi flag: expected true")
	}
}

func TestIndexDropExactArgs(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{database: "test"}
	cmd := newIndexDropCmd(cfg)
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("index drop: expected error for zero args, got nil")
	}
	if err := cmd.Args(cmd, []string{"mytable"}); err == nil {
		t.Error("index drop: expected error for one arg, got nil")
	}
	if err := cmd.Args(cmd, []string{"mytable", "myindex"}); err != nil {
		t.Errorf("index drop: expected no error for two args, got %v", err)
	}
	if err := cmd.Args(cmd, []string{"mytable", "myindex", "extra"}); err == nil {
		t.Error("index drop: expected error for three args, got nil")
	}
}

func TestIndexRenameExactArgs(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{database: "test"}
	cmd := newIndexRenameCmd(cfg)
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("index rename: expected error for zero args, got nil")
	}
	if err := cmd.Args(cmd, []string{"mytable", "old", "new"}); err != nil {
		t.Errorf("index rename: expected no error for three args, got %v", err)
	}
	if err := cmd.Args(cmd, []string{"mytable", "old", "new", "extra"}); err == nil {
		t.Error("index rename: expected error for four args, got nil")
	}
}

func TestIndexStatusRangeArgs(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{database: "test"}
	cmd := newIndexStatusCmd(cfg)
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("index status: expected error for zero args, got nil")
	}
	if err := cmd.Args(cmd, []string{"mytable"}); err != nil {
		t.Errorf("index status: expected no error for one arg, got %v", err)
	}
	if err := cmd.Args(cmd, []string{"mytable", "myindex"}); err != nil {
		t.Errorf("index status: expected no error for two args, got %v", err)
	}
	if err := cmd.Args(cmd, []string{"mytable", "myindex", "extra"}); err == nil {
		t.Error("index status: expected error for three args, got nil")
	}
}

func TestIndexWaitRangeArgs(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{database: "test"}
	cmd := newIndexWaitCmd(cfg)
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("index wait: expected error for zero args, got nil")
	}
	if err := cmd.Args(cmd, []string{"mytable"}); err != nil {
		t.Errorf("index wait: expected no error for one arg, got %v", err)
	}
	if err := cmd.Args(cmd, []string{"mytable", "myindex"}); err != nil {
		t.Errorf("index wait: expected no error for two args, got %v", err)
	}
	if err := cmd.Args(cmd, []string{"mytable", "myindex", "extra"}); err == nil {
		t.Error("index wait: expected error for three args, got nil")
	}
}

func TestIndexTableRequiresDatabase(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	_, err := indexTable(cfg, "mytable")
	if err == nil {
		t.Error("indexTable: expected error when database is empty, got nil")
	}
}

func TestIndexTableReturnsTerm(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{database: "mydb"}
	term, err := indexTable(cfg, "mytable")
	if err != nil {
		t.Fatalf("indexTable: unexpected error: %v", err)
	}
	data, err := term.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: unexpected error: %v", err)
	}
	got := string(data)
	// TABLE([14,["mydb"]], "mytable") = [15,[[14,["mydb"]],"mytable"]]
	want := `[15,[[14,["mydb"]],"mytable"]]`
	if got != want {
		t.Errorf("indexTable term: got %s, want %s", got, want)
	}
}
