package response

import (
	"encoding/json"
	"fmt"
	"strings"

	"r-cli/internal/proto"
)

// ReqlClientError is returned when the server reports a CLIENT_ERROR (response type 16).
type ReqlClientError struct {
	Msg       string
	backtrace []json.RawMessage
}

func (e *ReqlClientError) Error() string { return formatMsg(e.Msg, e.backtrace) }

// ReqlCompileError is returned when the server reports a COMPILE_ERROR (response type 17).
type ReqlCompileError struct {
	Msg       string
	backtrace []json.RawMessage
}

func (e *ReqlCompileError) Error() string { return formatMsg(e.Msg, e.backtrace) }

// ReqlRuntimeError is returned for RUNTIME_ERROR (response type 18) with no specific subtype.
type ReqlRuntimeError struct {
	Msg       string
	backtrace []json.RawMessage
}

func (e *ReqlRuntimeError) Error() string { return formatMsg(e.Msg, e.backtrace) }

// ReqlNonExistenceError is a RUNTIME_ERROR with ErrorType NON_EXISTENCE.
type ReqlNonExistenceError struct {
	Msg       string
	backtrace []json.RawMessage
}

func (e *ReqlNonExistenceError) Error() string { return formatMsg(e.Msg, e.backtrace) }

// ReqlPermissionError is a RUNTIME_ERROR with ErrorType PERMISSION_ERROR.
type ReqlPermissionError struct {
	Msg       string
	backtrace []json.RawMessage
}

func (e *ReqlPermissionError) Error() string { return formatMsg(e.Msg, e.backtrace) }

// MapError converts a server error response into a typed Go error.
// Returns nil for non-error response types.
func MapError(resp *Response) error {
	if !resp.Type.IsError() {
		return nil
	}
	msg := extractMessage(resp.Results)
	bt := resp.Backtrace

	switch resp.Type {
	case proto.ResponseClientError:
		return &ReqlClientError{Msg: msg, backtrace: bt}
	case proto.ResponseCompileError:
		return &ReqlCompileError{Msg: msg, backtrace: bt}
	case proto.ResponseRuntimeError:
		return mapRuntimeError(msg, resp.ErrType, bt)
	default:
		return fmt.Errorf("reql: unknown error response type %d: %s", resp.Type, msg)
	}
}

func mapRuntimeError(msg string, errType proto.ErrorType, bt []json.RawMessage) error {
	switch errType {
	case proto.ErrorNonExistence:
		return &ReqlNonExistenceError{Msg: msg, backtrace: bt}
	case proto.ErrorPermission:
		return &ReqlPermissionError{Msg: msg, backtrace: bt}
	default:
		return &ReqlRuntimeError{Msg: msg, backtrace: bt}
	}
}

// extractMessage returns the first string result from the results array.
func extractMessage(results []json.RawMessage) string {
	if len(results) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(results[0], &s); err != nil {
		return string(results[0])
	}
	return s
}

// formatMsg appends backtrace frames to the message when frames are present.
func formatMsg(msg string, bt []json.RawMessage) string {
	if len(bt) == 0 {
		return msg
	}
	frames := make([]string, len(bt))
	for i, f := range bt {
		frames[i] = string(f)
	}
	return fmt.Sprintf("%s\nBacktrace: %s", msg, strings.Join(frames, ", "))
}
