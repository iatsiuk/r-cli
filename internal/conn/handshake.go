package conn

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"

	"r-cli/internal/proto"
)

// ErrReqlAuth indicates an authentication error (error_code 10-20) during handshake.
var ErrReqlAuth = errors.New("reql: authentication error")

type step3Request struct {
	ProtocolVersion      int    `json:"protocol_version"`
	AuthenticationMethod string `json:"authentication_method"`
	Authentication       string `json:"authentication"`
}

type step5Request struct {
	Authentication string `json:"authentication"`
}

type step2Response struct {
	Success            bool   `json:"success"`
	MinProtocolVersion int    `json:"min_protocol_version"`
	MaxProtocolVersion int    `json:"max_protocol_version"`
	ServerVersion      string `json:"server_version"`
}

type step4Response struct {
	Success        bool   `json:"success"`
	Authentication string `json:"authentication"`
	Error          string `json:"error"`
	ErrorCode      int    `json:"error_code"`
}

type step6Response struct {
	Success        bool   `json:"success"`
	Authentication string `json:"authentication"`
	Error          string `json:"error"`
}

// buildStep1 returns the 4-byte little-endian V1_0 magic number.
func buildStep1() []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(proto.V1_0))
	return b
}

// buildStep3 returns the null-terminated JSON authentication request for step 3.
func buildStep3(clientFirstMsg string) ([]byte, error) {
	req := step3Request{
		ProtocolVersion:      0,
		AuthenticationMethod: "SCRAM-SHA-256",
		Authentication:       clientFirstMsg,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("buildStep3: %w", err)
	}
	return append(data, 0x00), nil
}

// buildStep5 returns the null-terminated JSON client-final-message for step 5.
func buildStep5(clientFinalMsg string) ([]byte, error) {
	req := step5Request{Authentication: clientFinalMsg}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("buildStep5: %w", err)
	}
	return append(data, 0x00), nil
}

// parseStep2 parses the server's step 2 response, returning server version info.
func parseStep2(data []byte) (*step2Response, error) {
	var resp step2Response
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parseStep2: %w", err)
	}
	if !resp.Success {
		return nil, fmt.Errorf("parseStep2: server returned failure")
	}
	return &resp, nil
}

// parseStep4 parses the server's step 4 response, returning the authentication field.
// Error codes 10-20 wrap ErrReqlAuth.
func parseStep4(data []byte) (string, error) {
	var resp step4Response
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("parseStep4: %w", err)
	}
	if !resp.Success {
		if resp.ErrorCode >= 10 && resp.ErrorCode <= 20 {
			return "", fmt.Errorf("%w: %s", ErrReqlAuth, resp.Error)
		}
		return "", fmt.Errorf("parseStep4: authentication failed: %s", resp.Error)
	}
	return resp.Authentication, nil
}

// parseStep6 parses the server's step 6 final response, returning the authentication field.
func parseStep6(data []byte) (string, error) {
	var resp step6Response
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("parseStep6: %w", err)
	}
	if !resp.Success {
		return "", fmt.Errorf("parseStep6: server returned failure: %s", resp.Error)
	}
	return resp.Authentication, nil
}
