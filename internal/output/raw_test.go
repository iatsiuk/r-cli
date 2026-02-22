package output

import (
	"bytes"
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
