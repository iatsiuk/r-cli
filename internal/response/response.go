package response

import (
	"encoding/json"
	"fmt"

	"r-cli/internal/proto"
)

// Response is a parsed RethinkDB server response.
type Response struct {
	Type    proto.ResponseType `json:"t"`
	Results []json.RawMessage  `json:"r"`
	ErrType proto.ErrorType    `json:"e,omitempty"`
	// b holds raw backtrace frames for error responses
	Backtrace []json.RawMessage    `json:"b,omitempty"`
	Notes     []proto.ResponseNote `json:"n,omitempty"`
	Profile   json.RawMessage      `json:"p,omitempty"`
}

// Parse unmarshals a raw JSON payload into a Response.
func Parse(data []byte) (*Response, error) {
	var r Response
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("response: parse: %w", err)
	}
	return &r, nil
}
