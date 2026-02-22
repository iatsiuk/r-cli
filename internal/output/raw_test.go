package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestRaw_SingleStringValue(t *testing.T) {
	t.Parallel()
	iter := newIter(`"hello world"`)
	var buf bytes.Buffer
	if err := Raw(&buf, iter); err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(buf.String())
	if got != "hello world" {
		t.Errorf("expected plain string, got: %q", got)
	}
}

func TestRaw_EachRowOnSeparateLine(t *testing.T) {
	t.Parallel()
	iter := newIter(`"foo"`, `"bar"`, `"baz"`)
	var buf bytes.Buffer
	if err := Raw(&buf, iter); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), buf.String())
	}
	if lines[0] != "foo" || lines[1] != "bar" || lines[2] != "baz" {
		t.Errorf("unexpected lines: %v", lines)
	}
}

func TestRaw_NonStringValue(t *testing.T) {
	t.Parallel()
	iter := newIter(`{"key":"val"}`)
	var buf bytes.Buffer
	if err := Raw(&buf, iter); err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(buf.String())
	if got != `{"key":"val"}` {
		t.Errorf("expected raw JSON, got: %q", got)
	}
}

func TestRaw_IteratorError(t *testing.T) {
	t.Parallel()
	errStream := errors.New("stream error")
	iter := &mockIter{items: []json.RawMessage{json.RawMessage(`"hello"`)}, err: errStream}
	var buf bytes.Buffer
	if err := Raw(&buf, iter); !errors.Is(err, errStream) {
		t.Errorf("expected stream error, got %v", err)
	}
}
