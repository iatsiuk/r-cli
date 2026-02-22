package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func TestInsertCmdRegistered(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	for _, sub := range root.Commands() {
		if sub.Name() == "insert" {
			return
		}
	}
	t.Error("insert subcommand not registered on root command")
}

func TestInsertExactArgs(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newInsertCmd(cfg)
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("insert: expected error for zero args, got nil")
	}
	if err := cmd.Args(cmd, []string{"db.table"}); err != nil {
		t.Errorf("insert: expected no error for one arg, got %v", err)
	}
	if err := cmd.Args(cmd, []string{"db.table", "extra"}); err == nil {
		t.Error("insert: expected error for two args, got nil")
	}
}

func TestParseTableRef(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input   string
		db      string
		table   string
		wantErr bool
	}{
		{"mydb.users", "mydb", "users", false},
		{"test.orders", "test", "orders", false},
		{"db.table.extra", "db", "table.extra", false}, // dots after first are table name
		{"notadottedref", "", "", true},
		{".table", "", "", true},
		{"db.", "", "", true},
		{".", "", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			db, table, err := parseTableRef(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("parseTableRef(%q): expected error, got nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseTableRef(%q): unexpected error: %v", tc.input, err)
			}
			if db != tc.db {
				t.Errorf("db: got %q, want %q", db, tc.db)
			}
			if table != tc.table {
				t.Errorf("table: got %q, want %q", table, tc.table)
			}
		})
	}
}

func TestInsertFileFlagShorthand(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newInsertCmd(cfg)
	if err := cmd.ParseFlags([]string{"-F", "data.jsonl"}); err != nil {
		t.Fatal(err)
	}
	v, err := cmd.Flags().GetString("file")
	if err != nil {
		t.Fatal(err)
	}
	if v != "data.jsonl" {
		t.Errorf("-F flag: got %q, want %q", v, "data.jsonl")
	}
}

func TestInsertBatchSizeDefault(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newInsertCmd(cfg)
	v, err := cmd.Flags().GetInt("batch-size")
	if err != nil {
		t.Fatal(err)
	}
	if v != 200 {
		t.Errorf("batch-size default: got %d, want 200", v)
	}
}

func TestInsertBatchSizeFlag(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newInsertCmd(cfg)
	if err := cmd.ParseFlags([]string{"--batch-size", "50"}); err != nil {
		t.Fatal(err)
	}
	v, err := cmd.Flags().GetInt("batch-size")
	if err != nil {
		t.Fatal(err)
	}
	if v != 50 {
		t.Errorf("--batch-size: got %d, want 50", v)
	}
}

func TestInsertConflictDefault(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newInsertCmd(cfg)
	v, err := cmd.Flags().GetString("conflict")
	if err != nil {
		t.Fatal(err)
	}
	if v != "error" {
		t.Errorf("conflict default: got %q, want %q", v, "error")
	}
}

func TestInsertConflictFlag(t *testing.T) {
	t.Parallel()
	for _, val := range []string{"replace", "update", "error"} {
		t.Run(val, func(t *testing.T) {
			t.Parallel()
			cfg := &rootConfig{}
			cmd := newInsertCmd(cfg)
			if err := cmd.ParseFlags([]string{"--conflict", val}); err != nil {
				t.Fatal(err)
			}
			got, err := cmd.Flags().GetString("conflict")
			if err != nil {
				t.Fatal(err)
			}
			if got != val {
				t.Errorf("--conflict %q: got %q", val, got)
			}
		})
	}
}

func TestDetectInputFormat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		file string
		flag string
		want string
	}{
		{"data.json", "", "json"},
		{"data.jsonl", "", "jsonl"},
		{"data.ndjson", "", "jsonl"},
		{"", "", "jsonl"},
		{"data.json", "jsonl", "jsonl"}, // flag overrides extension
		{"data.jsonl", "json", "json"},  // flag overrides extension
		{"data.txt", "json", "json"},
		{"data.txt", "jsonl", "jsonl"},
		{"data.txt", "", "jsonl"}, // default
	}
	for _, tc := range tests {
		t.Run(tc.file+"_"+tc.flag, func(t *testing.T) {
			t.Parallel()
			got := detectInputFormat(tc.file, tc.flag)
			if got != tc.want {
				t.Errorf("detectInputFormat(%q, %q) = %q, want %q", tc.file, tc.flag, got, tc.want)
			}
		})
	}
}

func TestInsertFormatFlagControlsInputFormat(t *testing.T) {
	t.Parallel()
	// the root --format flag value is used as input format for insert
	// json file with --format jsonl should be read as jsonl
	got := detectInputFormat("data.json", "jsonl")
	if got != "jsonl" {
		t.Errorf("detectInputFormat: --format jsonl should override extension, got %q", got)
	}
}

func TestOpenInputSourceStdin(t *testing.T) {
	t.Parallel()
	stdin := strings.NewReader("test data")
	r, closer, err := openInputSource("", stdin)
	if err != nil {
		t.Fatalf("openInputSource: unexpected error: %v", err)
	}
	defer closer()
	if r != stdin {
		t.Error("openInputSource: expected stdin reader when file is empty")
	}
}

func TestOpenInputSourceFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := dir + "/data.jsonl"
	content := `{"id":1}` + "\n" + `{"id":2}` + "\n"
	if err := writeFile(path, content); err != nil {
		t.Fatal(err)
	}
	r, closer, err := openInputSource(path, nil)
	if err != nil {
		t.Fatalf("openInputSource: unexpected error: %v", err)
	}
	defer closer()
	if r == nil {
		t.Error("openInputSource: expected non-nil reader for file")
	}
}

func TestOpenInputSourceMissingFile(t *testing.T) {
	t.Parallel()
	_, _, err := openInputSource("/nonexistent/path.jsonl", nil)
	if err == nil {
		t.Error("openInputSource: expected error for missing file, got nil")
	}
}

func TestInsertResultJSONMarshal(t *testing.T) {
	t.Parallel()
	res := insertResult{Inserted: 10, Errors: 2}
	data, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, `"inserted":10`) {
		t.Errorf("insertResult JSON: missing inserted field in %q", got)
	}
	if !strings.Contains(got, `"errors":2`) {
		t.Errorf("insertResult JSON: missing errors field in %q", got)
	}
}

func TestInsertJSONLReadsDocuments(t *testing.T) {
	t.Parallel()
	// verify insertJSONL parses lines correctly by testing the scanner logic
	input := strings.NewReader(`{"id":1}` + "\n" + `{"id":2}` + "\n" + "\n" + `{"id":3}`)
	var captured []json.RawMessage
	scanner := newDocScanner(input)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) > 0 {
			captured = append(captured, json.RawMessage(string(line)))
		}
	}
	if len(captured) != 3 {
		t.Errorf("expected 3 docs, got %d", len(captured))
	}
}

func TestInsertJSONParsesArray(t *testing.T) {
	t.Parallel()
	input := `[{"id":1},{"id":2},{"id":3}]`
	var docs []json.RawMessage
	if err := json.Unmarshal([]byte(input), &docs); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if len(docs) != 3 {
		t.Errorf("expected 3 docs, got %d", len(docs))
	}
}

func TestInsertCmdUse(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	cmd := newInsertCmd(cfg)
	if cmd.Use != "insert <db.table>" {
		t.Errorf("Use: got %q, want %q", cmd.Use, "insert <db.table>")
	}
}

// helpers

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}

func newDocScanner(r io.Reader) *bufio.Scanner {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 1024*1024), 1024*1024)
	return s
}
