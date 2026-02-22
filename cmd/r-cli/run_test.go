package main

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

// stubIter is a minimal RowIterator for testing writeOutput.
type stubIter struct {
	rows []json.RawMessage
	i    int
}

func (s *stubIter) Next() (json.RawMessage, error) {
	if s.i >= len(s.rows) {
		return nil, io.EOF
	}
	row := s.rows[s.i]
	s.i++
	return row, nil
}

func TestWriteOutput(t *testing.T) {
	t.Parallel()
	row := json.RawMessage(`{"key":"val"}`)
	tests := []struct {
		format string
		check  func(string) bool
	}{
		{"json", func(s string) bool { return strings.Contains(s, `"key"`) }},
		{"jsonl", func(s string) bool { return strings.TrimSpace(s) == `{"key":"val"}` }},
		{"raw", func(s string) bool { return strings.Contains(s, "val") }},
		{"unknown", func(s string) bool { return strings.Contains(s, `"key"`) }}, // falls to default JSON
		{"", func(s string) bool { return strings.Contains(s, `"key"`) }},        // falls to default JSON
	}
	for _, tc := range tests {
		t.Run(tc.format, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			iter := &stubIter{rows: []json.RawMessage{row}}
			if err := writeOutput(&buf, tc.format, iter); err != nil {
				t.Fatalf("writeOutput(%q): %v", tc.format, err)
			}
			if !tc.check(buf.String()) {
				t.Errorf("writeOutput(%q): unexpected output: %q", tc.format, buf.String())
			}
		})
	}
}

func TestReadTermFromArg(t *testing.T) {
	t.Parallel()
	term := `[15,[[14,["test"]],"users"]]`
	got, err := readTerm([]string{term}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != term {
		t.Errorf("got %q, want %q", got, term)
	}
}

func TestReadTermFromStdin(t *testing.T) {
	t.Parallel()
	term := `[15,[[14,["test"]],"users"]]`
	got, err := readTerm(nil, strings.NewReader(term))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != term {
		t.Errorf("got %q, want %q", got, term)
	}
}

func TestReadTermStdinWithNewline(t *testing.T) {
	t.Parallel()
	term := `[1,2,3]`
	got, err := readTerm(nil, strings.NewReader(term+"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != term {
		t.Errorf("got %q, want %q", got, term)
	}
}

func TestReadTermInvalidArgJSON(t *testing.T) {
	t.Parallel()
	_, err := readTerm([]string{"not-json"}, nil)
	if err == nil {
		t.Error("expected error for invalid JSON arg, got nil")
	}
}

func TestReadTermInvalidStdinJSON(t *testing.T) {
	t.Parallel()
	_, err := readTerm(nil, strings.NewReader("not-json"))
	if err == nil {
		t.Error("expected error for invalid JSON stdin, got nil")
	}
}

func TestRunCmdRegistered(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	for _, sub := range root.Commands() {
		if sub.Name() == "run" {
			return
		}
	}
	t.Error("run subcommand not registered on root command")
}

func TestRunCmdUsage(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	for _, sub := range root.Commands() {
		if sub.Name() == "run" {
			if sub.Use != "run [term]" {
				t.Errorf("run Use: got %q, want %q", sub.Use, "run [term]")
			}
			return
		}
	}
	t.Error("run subcommand not found")
}
