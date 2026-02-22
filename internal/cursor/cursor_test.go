package cursor

import (
	"encoding/json"
	"errors"
	"io"
	"testing"

	"r-cli/internal/proto"
	"r-cli/internal/response"
)

func rawMsg(s string) json.RawMessage { return json.RawMessage(s) }

func TestAtomCursor_SingleValue(t *testing.T) {
	t.Parallel()
	resp := &response.Response{
		Type:    proto.ResponseSuccessAtom,
		Results: []json.RawMessage{rawMsg(`"hello"`)},
	}
	c := NewAtom(resp)

	item, err := c.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(item) != `"hello"` {
		t.Fatalf("got %s, want %q", item, "hello")
	}

	// second call must return EOF
	_, err = c.Next()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

func TestAtomCursor_All(t *testing.T) {
	t.Parallel()
	resp := &response.Response{
		Type:    proto.ResponseSuccessAtom,
		Results: []json.RawMessage{rawMsg(`42`)},
	}
	c := NewAtom(resp)

	all, err := c.All()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 element, got %d", len(all))
	}
	if string(all[0]) != `42` {
		t.Fatalf("got %s, want 42", all[0])
	}
}

func TestAtomCursor_EOF_Immediately_When_Empty(t *testing.T) {
	t.Parallel()
	resp := &response.Response{
		Type:    proto.ResponseSuccessAtom,
		Results: nil,
	}
	c := NewAtom(resp)

	item, err := c.Next()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF for empty atom, got err=%v item=%v", err, item)
	}
}

func TestSeqCursor_IterateAll(t *testing.T) {
	t.Parallel()
	resp := &response.Response{
		Type: proto.ResponseSuccessSequence,
		Results: []json.RawMessage{
			rawMsg(`1`),
			rawMsg(`2`),
			rawMsg(`3`),
		},
	}
	c := NewSequence(resp)

	for i := 1; i <= 3; i++ {
		item, err := c.Next()
		if err != nil {
			t.Fatalf("step %d: unexpected error: %v", i, err)
		}
		want := string(rawMsg(string(rune('0' + i))))
		if string(item) != want {
			t.Fatalf("step %d: got %s, want %s", i, item, want)
		}
	}

	_, err := c.Next()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF after exhaustion, got %v", err)
	}
}

func TestSeqCursor_All(t *testing.T) {
	t.Parallel()
	resp := &response.Response{
		Type: proto.ResponseSuccessSequence,
		Results: []json.RawMessage{
			rawMsg(`"a"`),
			rawMsg(`"b"`),
		},
	}
	c := NewSequence(resp)

	all, err := c.All()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(all))
	}
	if string(all[0]) != `"a"` || string(all[1]) != `"b"` {
		t.Fatalf("unexpected values: %v", all)
	}
}

func TestSeqCursor_Empty(t *testing.T) {
	t.Parallel()
	resp := &response.Response{
		Type:    proto.ResponseSuccessSequence,
		Results: nil,
	}
	c := NewSequence(resp)

	_, err := c.Next()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF for empty sequence, got %v", err)
	}

	all, err := c.All()
	if err != nil {
		t.Fatalf("unexpected error from All(): %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("expected empty slice, got %v", all)
	}
}
