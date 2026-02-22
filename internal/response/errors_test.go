package response

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"r-cli/internal/proto"
)

func rawMessages(vals ...string) []json.RawMessage {
	msgs := make([]json.RawMessage, len(vals))
	for i, v := range vals {
		msgs[i] = json.RawMessage(v)
	}
	return msgs
}

func TestMapError_ClientError(t *testing.T) {
	t.Parallel()
	resp := &Response{
		Type:    proto.ResponseClientError,
		Results: rawMessages(`"bad client request"`),
	}
	err := MapError(resp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *ReqlClientError
	if !errors.As(err, &e) {
		t.Fatalf("expected *ReqlClientError, got %T", err)
	}
	if e.Msg != "bad client request" {
		t.Errorf("got %q, want %q", e.Msg, "bad client request")
	}
}

func TestMapError_CompileError(t *testing.T) {
	t.Parallel()
	resp := &Response{
		Type:    proto.ResponseCompileError,
		Results: rawMessages(`"syntax error"`),
	}
	err := MapError(resp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *ReqlCompileError
	if !errors.As(err, &e) {
		t.Fatalf("expected *ReqlCompileError, got %T", err)
	}
	if e.Msg != "syntax error" {
		t.Errorf("got %q, want %q", e.Msg, "syntax error")
	}
}

func TestMapError_RuntimeError(t *testing.T) {
	t.Parallel()
	resp := &Response{
		Type:    proto.ResponseRuntimeError,
		ErrType: proto.ErrorQueryLogic,
		Results: rawMessages(`"query logic error"`),
	}
	err := MapError(resp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *ReqlRuntimeError
	if !errors.As(err, &e) {
		t.Fatalf("expected *ReqlRuntimeError, got %T", err)
	}
	if e.Msg != "query logic error" {
		t.Errorf("got %q, want %q", e.Msg, "query logic error")
	}
}

func TestMapError_NonExistenceError(t *testing.T) {
	t.Parallel()
	resp := &Response{
		Type:    proto.ResponseRuntimeError,
		ErrType: proto.ErrorNonExistence,
		Results: rawMessages(`"key not found"`),
	}
	err := MapError(resp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *ReqlNonExistenceError
	if !errors.As(err, &e) {
		t.Fatalf("expected *ReqlNonExistenceError, got %T", err)
	}
	if e.Msg != "key not found" {
		t.Errorf("got %q, want %q", e.Msg, "key not found")
	}
}

func TestMapError_PermissionError(t *testing.T) {
	t.Parallel()
	resp := &Response{
		Type:    proto.ResponseRuntimeError,
		ErrType: proto.ErrorPermission,
		Results: rawMessages(`"access denied"`),
	}
	err := MapError(resp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *ReqlPermissionError
	if !errors.As(err, &e) {
		t.Fatalf("expected *ReqlPermissionError, got %T", err)
	}
	if e.Msg != "access denied" {
		t.Errorf("got %q, want %q", e.Msg, "access denied")
	}
}

func TestMapError_BacktraceInMessage(t *testing.T) {
	t.Parallel()
	resp := &Response{
		Type:      proto.ResponseRuntimeError,
		ErrType:   proto.ErrorQueryLogic,
		Results:   rawMessages(`"some error"`),
		Backtrace: rawMessages(`[0]`, `[1,2]`),
	}
	err := MapError(resp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "some error") {
		t.Errorf("message %q missing base message", msg)
	}
	if !strings.Contains(msg, "[0]") {
		t.Errorf("message %q missing backtrace frame [0]", msg)
	}
	if !strings.Contains(msg, "[1,2]") {
		t.Errorf("message %q missing backtrace frame [1,2]", msg)
	}
}

func TestMapError_EmptyResults(t *testing.T) {
	t.Parallel()
	resp := &Response{
		Type:    proto.ResponseClientError,
		Results: nil,
	}
	err := MapError(resp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *ReqlClientError
	if !errors.As(err, &e) {
		t.Fatalf("expected *ReqlClientError, got %T", err)
	}
	if e.Msg != "" {
		t.Errorf("expected empty message for nil results, got %q", e.Msg)
	}
}

func TestMapError_NonError(t *testing.T) {
	t.Parallel()
	resp := &Response{
		Type:    proto.ResponseSuccessAtom,
		Results: rawMessages(`"ok"`),
	}
	if err := MapError(resp); err != nil {
		t.Errorf("expected nil for non-error response, got %v", err)
	}
}
