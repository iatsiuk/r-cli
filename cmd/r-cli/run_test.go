package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
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

func TestConvertingIterTimePseudoType(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{"$reql_type$":"TIME","epoch_time":0,"timezone":"+00:00"}`)
	iter := &convertingIter{inner: &stubIter{rows: []json.RawMessage{raw}}}
	got, err := iter.Next()
	if err != nil {
		t.Fatal(err)
	}
	var parsed time.Time
	if jsonErr := json.Unmarshal(got, &parsed); jsonErr != nil {
		t.Fatalf("expected time.Time JSON, got %q: %v", got, jsonErr)
	}
	if parsed.UTC() != time.Unix(0, 0).UTC() {
		t.Errorf("got %v, want unix epoch", parsed)
	}
}

func TestConvertingIterBinaryPseudoType(t *testing.T) {
	t.Parallel()
	// "aGVsbG8=" is base64 for "hello"
	raw := json.RawMessage(`{"$reql_type$":"BINARY","data":"aGVsbG8="}`)
	iter := &convertingIter{inner: &stubIter{rows: []json.RawMessage{raw}}}
	got, err := iter.Next()
	if err != nil {
		t.Fatal(err)
	}
	// []byte marshals to base64 JSON string
	if string(got) != `"aGVsbG8="` {
		t.Errorf("got %q, want base64 string", got)
	}
}

func TestConvertingIterPassthrough(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{"key":"value"}`)
	iter := &convertingIter{inner: &stubIter{rows: []json.RawMessage{raw}}}
	got, err := iter.Next()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"key":"value"}` {
		t.Errorf("got %q, want original JSON", got)
	}
}

func TestConvertingIterEOF(t *testing.T) {
	t.Parallel()
	iter := &convertingIter{inner: &stubIter{rows: nil}}
	_, err := iter.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("got %v, want io.EOF", err)
	}
}
