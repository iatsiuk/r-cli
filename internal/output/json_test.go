package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
)

type mockIter struct {
	items []json.RawMessage
	pos   int
	err   error // returned after all items
}

func (m *mockIter) Next() (json.RawMessage, error) {
	if m.pos >= len(m.items) {
		if m.err != nil {
			return nil, m.err
		}
		return nil, io.EOF
	}
	item := m.items[m.pos]
	m.pos++
	return item, nil
}

func (m *mockIter) Close() error { return nil }

func newIter(items ...string) *mockIter {
	raw := make([]json.RawMessage, len(items))
	for i, s := range items {
		raw[i] = json.RawMessage(s)
	}
	return &mockIter{items: raw}
}

func TestJSON_SingleDocument(t *testing.T) {
	t.Parallel()
	iter := newIter(`{"name":"alice","age":30}`)
	var buf bytes.Buffer
	if err := JSON(&buf, iter); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, `"name"`) || !strings.Contains(got, `"age"`) {
		t.Errorf("unexpected output: %q", got)
	}
	// must not be wrapped in array
	if strings.HasPrefix(strings.TrimSpace(got), "[") {
		t.Errorf("single document should not be wrapped in array, got: %q", got)
	}
	// must be valid JSON
	var v interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(got)), &v); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}
	// must be pretty-printed (contains newlines)
	if !strings.Contains(got, "\n") {
		t.Errorf("expected pretty-printed output with newlines, got: %q", got)
	}
}

func TestJSON_ArrayOfDocuments(t *testing.T) {
	t.Parallel()
	iter := newIter(`{"a":1}`, `{"b":2}`, `{"c":3}`)
	var buf bytes.Buffer
	if err := JSON(&buf, iter); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	trimmed := strings.TrimSpace(got)
	// must be a JSON array
	if !strings.HasPrefix(trimmed, "[") || !strings.HasSuffix(trimmed, "]") {
		t.Errorf("expected JSON array, got: %q", got)
	}
	// must be valid JSON
	var arr []interface{}
	if err := json.Unmarshal([]byte(trimmed), &arr); err != nil {
		t.Errorf("output is not valid JSON array: %v\noutput: %q", err, got)
	}
	if len(arr) != 3 {
		t.Errorf("expected 3 items, got %d", len(arr))
	}
	// must be pretty-printed
	if !strings.Contains(got, "\n") {
		t.Errorf("expected pretty-printed output, got: %q", got)
	}
}

func TestJSON_EmptyResult(t *testing.T) {
	t.Parallel()
	iter := newIter()
	var buf bytes.Buffer
	if err := JSON(&buf, iter); err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(buf.String())
	if got != "[]" {
		t.Errorf("expected [], got: %q", got)
	}
}

func TestJSON_IteratorError(t *testing.T) {
	t.Parallel()
	errStream := errors.New("stream error")
	iter := &mockIter{items: []json.RawMessage{json.RawMessage(`{"a":1}`)}, err: errStream}
	var buf bytes.Buffer
	if err := JSON(&buf, iter); !errors.Is(err, errStream) {
		t.Errorf("expected stream error, got %v", err)
	}
}

func TestJSON_InvalidJSONFallback(t *testing.T) {
	t.Parallel()
	iter := &mockIter{items: []json.RawMessage{json.RawMessage("not-valid-json")}}
	var buf bytes.Buffer
	if err := JSON(&buf, iter); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(buf.String()); got != "not-valid-json" {
		t.Errorf("expected raw fallback for invalid JSON, got: %q", got)
	}
}
