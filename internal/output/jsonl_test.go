package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestJSONL_SingleDocument(t *testing.T) {
	t.Parallel()
	iter := newIter(`{"name":"alice","age":30}`)
	var buf bytes.Buffer
	if err := JSONL(&buf, iter); err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(buf.String())
	// must be compact (no indentation newlines within the object)
	if strings.Count(got, "\n") != 0 {
		t.Errorf("expected single line, got: %q", got)
	}
	var v interface{}
	if err := json.Unmarshal([]byte(got), &v); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}
}

func TestJSONL_SequenceAsOnePerLine(t *testing.T) {
	t.Parallel()
	iter := newIter(`{"a":1}`, `{"b":2}`, `{"c":3}`)
	var buf bytes.Buffer
	if err := JSONL(&buf, iter); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), buf.String())
	}
	// each line must be valid JSON (not wrapped in array)
	for i, line := range lines {
		var v interface{}
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			t.Errorf("line %d is not valid JSON: %v (line: %q)", i, err, line)
		}
	}
	// must not be wrapped in array
	if strings.HasPrefix(strings.TrimSpace(buf.String()), "[") {
		t.Errorf("JSONL output must not be wrapped in array")
	}
}

func TestJSONL_StreamingOutput(t *testing.T) {
	t.Parallel()
	// simulates changefeed-style continuous rows
	iter := newIter(`{"type":"add","id":1}`, `{"type":"change","id":2}`, `{"type":"remove","id":3}`)
	var buf bytes.Buffer
	if err := JSONL(&buf, iter); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	for i, line := range lines {
		var v map[string]interface{}
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			t.Errorf("line %d invalid JSON: %v", i, err)
		}
	}
}

func TestJSONL_IteratorError(t *testing.T) {
	t.Parallel()
	errStream := errors.New("stream error")
	iter := &mockIter{items: []json.RawMessage{json.RawMessage(`{"a":1}`)}, err: errStream}
	var buf bytes.Buffer
	if err := JSONL(&buf, iter); !errors.Is(err, errStream) {
		t.Errorf("expected stream error, got %v", err)
	}
}
