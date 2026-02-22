package main

import (
	"testing"
)

func TestGrantCmdRegistered(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	for _, sub := range root.Commands() {
		if sub.Name() == "grant" {
			return
		}
	}
	t.Error("grant subcommand not registered on root command")
}

func TestGrantExactArgs(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newGrantCmd(cfg)
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("grant: expected error for zero args, got nil")
	}
	if err := cmd.Args(cmd, []string{"alice"}); err != nil {
		t.Errorf("grant: expected no error for one arg, got %v", err)
	}
	if err := cmd.Args(cmd, []string{"alice", "extra"}); err == nil {
		t.Error("grant: expected error for two args, got nil")
	}
}

func TestGrantReadFlag(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newGrantCmd(cfg)
	if err := cmd.ParseFlags([]string{"--read"}); err != nil {
		t.Fatal(err)
	}
	got, err := cmd.Flags().GetBool("read")
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("--read flag: expected true")
	}
}

func TestGrantWriteFlag(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newGrantCmd(cfg)
	if err := cmd.ParseFlags([]string{"--write"}); err != nil {
		t.Fatal(err)
	}
	got, err := cmd.Flags().GetBool("write")
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("--write flag: expected true")
	}
}

func TestGrantReadFalse(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newGrantCmd(cfg)
	if err := cmd.ParseFlags([]string{"--read=false"}); err != nil {
		t.Fatal(err)
	}
	// flag was explicitly changed to false
	if !cmd.Flags().Changed("read") {
		t.Error("--read=false: expected flag to be changed")
	}
	got, err := cmd.Flags().GetBool("read")
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Error("--read=false: expected false")
	}
}

func TestGrantTableFlag(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newGrantCmd(cfg)
	if err := cmd.ParseFlags([]string{"--table", "users"}); err != nil {
		t.Fatal(err)
	}
	got, err := cmd.Flags().GetString("table")
	if err != nil {
		t.Fatal(err)
	}
	if got != "users" {
		t.Errorf("--table flag: got %q, want %q", got, "users")
	}
}

func TestBuildGrantPermsReadOnly(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newGrantCmd(cfg)
	if err := cmd.ParseFlags([]string{"--read"}); err != nil {
		t.Fatal(err)
	}
	perms := buildGrantPerms(cmd, true, false)
	if v, ok := perms["read"]; !ok || v != true {
		t.Errorf("buildGrantPerms: expected read=true, got %v", perms)
	}
	if _, ok := perms["write"]; ok {
		t.Error("buildGrantPerms: write should not be set when --write not provided")
	}
}

func TestBuildGrantPermsRevoke(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newGrantCmd(cfg)
	if err := cmd.ParseFlags([]string{"--read=false"}); err != nil {
		t.Fatal(err)
	}
	read, _ := cmd.Flags().GetBool("read")
	perms := buildGrantPerms(cmd, read, false)
	if v, ok := perms["read"]; !ok || v != false {
		t.Errorf("buildGrantPerms: expected read=false, got %v", perms)
	}
}

func TestGrantTermGlobal(t *testing.T) {
	t.Parallel()
	// global grant: no db, no table
	term := grantTerm("", "", "alice", map[string]interface{}{"read": true})
	_ = term // verify no panic
}

func TestGrantTermDB(t *testing.T) {
	t.Parallel()
	term := grantTerm("test", "", "alice", map[string]interface{}{"read": true})
	_ = term
}

func TestGrantTermTable(t *testing.T) {
	t.Parallel()
	term := grantTerm("test", "users", "alice", map[string]interface{}{"read": true, "write": true})
	_ = term
}
