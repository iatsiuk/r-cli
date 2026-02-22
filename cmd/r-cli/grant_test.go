package main

import (
	"encoding/json"
	"strings"
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
	term := grantTerm("", "", "alice", map[string]interface{}{"read": true})
	data, err := json.Marshal(term)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, `"alice"`) {
		t.Errorf("grantTerm global: expected alice in %s", got)
	}
	// global grant must not wrap in a DB term (termType 14)
	if strings.Contains(got, `[14,`) {
		t.Errorf("grantTerm global: unexpected DB term in %s", got)
	}
}

func TestGrantTermDB(t *testing.T) {
	t.Parallel()
	term := grantTerm("test", "", "alice", map[string]interface{}{"read": true})
	data, err := json.Marshal(term)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, `[14,`) {
		t.Errorf("grantTerm db: expected DB term in %s", got)
	}
	if !strings.Contains(got, `"alice"`) {
		t.Errorf("grantTerm db: expected alice in %s", got)
	}
	// db-scoped grant must not wrap in a Table term (termType 15)
	if strings.Contains(got, `[15,`) {
		t.Errorf("grantTerm db: unexpected Table term in %s", got)
	}
}

func TestGrantTermTable(t *testing.T) {
	t.Parallel()
	term := grantTerm("test", "users", "alice", map[string]interface{}{"read": true})
	data, err := json.Marshal(term)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, `[14,`) {
		t.Errorf("grantTerm table: expected DB term in %s", got)
	}
	if !strings.Contains(got, `[15,`) {
		t.Errorf("grantTerm table: expected Table term in %s", got)
	}
	if !strings.Contains(got, `"alice"`) {
		t.Errorf("grantTerm table: expected alice in %s", got)
	}
}

func TestBuildGrantPermsBothFlags(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newGrantCmd(cfg)
	if err := cmd.ParseFlags([]string{"--read", "--write"}); err != nil {
		t.Fatal(err)
	}
	read, _ := cmd.Flags().GetBool("read")
	write, _ := cmd.Flags().GetBool("write")
	perms := buildGrantPerms(cmd, read, write)
	if v, ok := perms["read"]; !ok || v != true {
		t.Errorf("buildGrantPerms both: expected read=true, got %v", perms)
	}
	if v, ok := perms["write"]; !ok || v != true {
		t.Errorf("buildGrantPerms both: expected write=true, got %v", perms)
	}
	if len(perms) != 2 {
		t.Errorf("buildGrantPerms both: expected 2 permissions, got %d", len(perms))
	}
}

func TestGrantTableRequiresDB(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newGrantCmd(cfg)
	if err := cmd.ParseFlags([]string{"--table", "users", "--read"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.RunE(cmd, []string{"alice"})
	if err == nil {
		t.Error("grant --table without --db: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "--table requires --db") {
		t.Errorf("grant --table without --db: unexpected error: %v", err)
	}
}
