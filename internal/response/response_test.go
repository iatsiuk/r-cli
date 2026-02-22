package response

import (
	"encoding/json"
	"testing"

	"r-cli/internal/proto"
)

func TestParse_SuccessAtom(t *testing.T) {
	t.Parallel()
	data := []byte(`{"t":1,"r":["foo"]}`)
	resp, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Type != proto.ResponseSuccessAtom {
		t.Errorf("got type %d, want %d", resp.Type, proto.ResponseSuccessAtom)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("got %d results, want 1", len(resp.Results))
	}
	var s string
	if err := json.Unmarshal(resp.Results[0], &s); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if s != "foo" {
		t.Errorf("got %q, want %q", s, "foo")
	}
}

func TestParse_ErrorResponse(t *testing.T) {
	t.Parallel()
	// runtime error with error type and backtrace
	data := []byte(`{"t":18,"r":["query error"],"e":3000000,"b":[[0],[1]]}`)
	resp, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Type != proto.ResponseRuntimeError {
		t.Errorf("got type %d, want %d", resp.Type, proto.ResponseRuntimeError)
	}
	if resp.ErrType != proto.ErrorQueryLogic {
		t.Errorf("got errtype %d, want %d", resp.ErrType, proto.ErrorQueryLogic)
	}
	if len(resp.Backtrace) != 2 {
		t.Errorf("got %d backtrace frames, want 2", len(resp.Backtrace))
	}
	if len(resp.Results) != 1 {
		t.Fatalf("got %d results, want 1", len(resp.Results))
	}
	var msg string
	if err := json.Unmarshal(resp.Results[0], &msg); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if msg != "query error" {
		t.Errorf("got %q, want %q", msg, "query error")
	}
}

func TestParse_WithNotes(t *testing.T) {
	t.Parallel()
	data := []byte(`{"t":3,"r":[{"id":1}],"n":[1]}`)
	resp, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Type != proto.ResponseSuccessPartial {
		t.Errorf("got type %d, want %d", resp.Type, proto.ResponseSuccessPartial)
	}
	if len(resp.Notes) != 1 {
		t.Fatalf("got %d notes, want 1", len(resp.Notes))
	}
	if resp.Notes[0] != proto.NoteSequenceFeed {
		t.Errorf("got note %d, want %d", resp.Notes[0], proto.NoteSequenceFeed)
	}
}

func TestParse_WithProfile(t *testing.T) {
	t.Parallel()
	data := []byte(`{"t":1,"r":["val"],"p":{"time":1.23}}`)
	resp, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Profile == nil {
		t.Fatal("expected non-nil profile")
	}
	// profile is kept as raw JSON
	var profile map[string]interface{}
	if err := json.Unmarshal(resp.Profile, &profile); err != nil {
		t.Fatalf("unmarshal profile: %v", err)
	}
	if _, ok := profile["time"]; !ok {
		t.Error("expected 'time' field in profile")
	}
}
