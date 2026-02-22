package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRootHostDefault(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	host, err := cmd.PersistentFlags().GetString("host")
	if err != nil {
		t.Fatal(err)
	}
	if host != "localhost" {
		t.Errorf("got %q, want %q", host, "localhost")
	}
}

func TestRootPortDefault(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	port, err := cmd.PersistentFlags().GetInt("port")
	if err != nil {
		t.Fatal(err)
	}
	if port != 28015 {
		t.Errorf("got %d, want %d", port, 28015)
	}
}

func TestRootDBDefault(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	db, err := cmd.PersistentFlags().GetString("db")
	if err != nil {
		t.Fatal(err)
	}
	if db != "" {
		t.Errorf("got %q, want empty", db)
	}
}

func TestRootUserDefault(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	user, err := cmd.PersistentFlags().GetString("user")
	if err != nil {
		t.Fatal(err)
	}
	if user != "admin" {
		t.Errorf("got %q, want %q", user, "admin")
	}
}

func TestRootPasswordDefault(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	password, err := cmd.PersistentFlags().GetString("password")
	if err != nil {
		t.Fatal(err)
	}
	if password != "" {
		t.Errorf("got %q, want empty", password)
	}
}

func TestRootTimeoutDefault(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	timeout, err := cmd.PersistentFlags().GetDuration("timeout")
	if err != nil {
		t.Fatal(err)
	}
	if timeout != 30*time.Second {
		t.Errorf("got %v, want %v", timeout, 30*time.Second)
	}
}

func TestRootFormatDefault(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	format, err := cmd.PersistentFlags().GetString("format")
	if err != nil {
		t.Fatal(err)
	}
	if format != "json" {
		t.Errorf("got %q, want %q", format, "json")
	}
}

func TestRootHostShorthand(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	if err := cmd.ParseFlags([]string{"-H", "myhost"}); err != nil {
		t.Fatal(err)
	}
	got, _ := cmd.PersistentFlags().GetString("host")
	if got != "myhost" {
		t.Errorf("got %q, want %q", got, "myhost")
	}
}

func TestRootPortShorthand(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	if err := cmd.ParseFlags([]string{"-P", "19015"}); err != nil {
		t.Fatal(err)
	}
	got, _ := cmd.PersistentFlags().GetInt("port")
	if got != 19015 {
		t.Errorf("got %d, want %d", got, 19015)
	}
}

func TestRootDBShorthand(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	if err := cmd.ParseFlags([]string{"-d", "mydb"}); err != nil {
		t.Fatal(err)
	}
	got, _ := cmd.PersistentFlags().GetString("db")
	if got != "mydb" {
		t.Errorf("got %q, want %q", got, "mydb")
	}
}

func TestRootUserShorthand(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	if err := cmd.ParseFlags([]string{"-u", "myuser"}); err != nil {
		t.Fatal(err)
	}
	got, _ := cmd.PersistentFlags().GetString("user")
	if got != "myuser" {
		t.Errorf("got %q, want %q", got, "myuser")
	}
}

func TestRootPasswordShorthand(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	if err := cmd.ParseFlags([]string{"-p", "secret"}); err != nil {
		t.Fatal(err)
	}
	got, _ := cmd.PersistentFlags().GetString("password")
	if got != "secret" {
		t.Errorf("got %q, want %q", got, "secret")
	}
}

func TestRootTimeoutShorthand(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	if err := cmd.ParseFlags([]string{"-t", "10s"}); err != nil {
		t.Fatal(err)
	}
	got, _ := cmd.PersistentFlags().GetDuration("timeout")
	if got != 10*time.Second {
		t.Errorf("got %v, want %v", got, 10*time.Second)
	}
}

func TestRootFormatShorthand(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	if err := cmd.ParseFlags([]string{"-f", "jsonl"}); err != nil {
		t.Fatal(err)
	}
	got, _ := cmd.PersistentFlags().GetString("format")
	if got != "jsonl" {
		t.Errorf("got %q, want %q", got, "jsonl")
	}
}

func TestRootFormatValues(t *testing.T) {
	t.Parallel()
	for _, v := range []string{"json", "jsonl", "raw", "table"} {
		cmd := newRootCmd()
		if err := cmd.ParseFlags([]string{"--format", v}); err != nil {
			t.Fatalf("format %q: %v", v, err)
		}
		got, _ := cmd.PersistentFlags().GetString("format")
		if got != v {
			t.Errorf("format %q: got %q", v, got)
		}
	}
}

func TestPasswordFileFlag(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := dir + "/password.txt"
	if err := os.WriteFile(path, []byte("mysecret\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := &rootConfig{passwordFile: path}
	if err := cfg.resolvePassword(); err != nil {
		t.Fatalf("resolvePassword: %v", err)
	}
	if cfg.password != "mysecret" {
		t.Errorf("password: got %q, want %q", cfg.password, "mysecret")
	}
}

func TestPasswordFileNotFound(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{passwordFile: "/nonexistent/path/password.txt"}
	if err := cfg.resolvePassword(); err == nil {
		t.Error("expected error for missing password file, got nil")
	}
}

func TestPasswordFileStripsNewline(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"unix newline", "pass\n", "pass"},
		{"windows newline", "pass\r\n", "pass"},
		{"no newline", "pass", "pass"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := dir + "/password.txt"
			if err := os.WriteFile(path, []byte(tc.content), 0o600); err != nil {
				t.Fatal(err)
			}

			cfg := &rootConfig{passwordFile: path}
			if err := cfg.resolvePassword(); err != nil {
				t.Fatal(err)
			}
			if cfg.password != tc.want {
				t.Errorf("got %q, want %q", cfg.password, tc.want)
			}
		})
	}
}

func TestVersionFlag(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "r-cli") {
		t.Errorf("version output does not contain 'r-cli': %q", out)
	}
}
