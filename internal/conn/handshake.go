package conn

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"r-cli/internal/proto"
	"r-cli/internal/scram"
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
	Error              string `json:"error"`
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
		if resp.Error != "" {
			return nil, fmt.Errorf("parseStep2: server returned failure: %s", resp.Error)
		}
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

// Handshake performs the RethinkDB V1_0 handshake over rw, authenticating as user with password.
// Steps 1 and 3 are pipelined (sent together) to save one round trip.
func Handshake(rw io.ReadWriter, user, password string) error {
	conv := scram.NewConversation(user, password)
	if err := writePipelined(rw, conv.ClientFirst()); err != nil {
		return err
	}
	serverFirstMsg, err := exchangeInitial(rw)
	if err != nil {
		return err
	}
	clientFinal, err := conv.ServerFirst(serverFirstMsg)
	if err != nil {
		return fmt.Errorf("handshake: %w", err)
	}
	serverFinalMsg, err := exchangeFinal(rw, clientFinal)
	if err != nil {
		return err
	}
	if err := conv.ServerFinal(serverFinalMsg); err != nil {
		return fmt.Errorf("handshake: %w", err)
	}
	return nil
}

// writePipelined writes step 1 (magic) and step 3 (client-first-message) in a single call.
func writePipelined(w io.Writer, clientFirstMsg string) error {
	step1 := buildStep1()
	step3, err := buildStep3(clientFirstMsg)
	if err != nil {
		return fmt.Errorf("handshake: %w", err)
	}
	pipeline := make([]byte, len(step1)+len(step3))
	copy(pipeline, step1)
	copy(pipeline[len(step1):], step3)
	if _, err := w.Write(pipeline); err != nil {
		return fmt.Errorf("handshake: write: %w", err)
	}
	return nil
}

// exchangeInitial reads step 2 (server info) and step 4 (server-first-message).
func exchangeInitial(r io.Reader) (string, error) {
	data, err := readNullTerminated(r)
	if err != nil {
		return "", fmt.Errorf("handshake: read step 2: %w", err)
	}
	step2Resp, err := parseStep2(data)
	if err != nil {
		return "", fmt.Errorf("handshake: %w", err)
	}
	if step2Resp.MinProtocolVersion > 0 {
		return "", fmt.Errorf("handshake: server requires min protocol_version=%d, client supports 0",
			step2Resp.MinProtocolVersion)
	}
	data, err = readNullTerminated(r)
	if err != nil {
		return "", fmt.Errorf("handshake: read step 4: %w", err)
	}
	msg, err := parseStep4(data)
	if err != nil {
		return "", fmt.Errorf("handshake: %w", err)
	}
	return msg, nil
}

// exchangeFinal writes step 5 (client-final-message) and reads step 6 (server-final-message).
func exchangeFinal(rw io.ReadWriter, clientFinal string) (string, error) {
	step5, err := buildStep5(clientFinal)
	if err != nil {
		return "", fmt.Errorf("handshake: %w", err)
	}
	if _, err := rw.Write(step5); err != nil {
		return "", fmt.Errorf("handshake: write step 5: %w", err)
	}
	data, err := readNullTerminated(rw)
	if err != nil {
		return "", fmt.Errorf("handshake: read step 6: %w", err)
	}
	msg, err := parseStep6(data)
	if err != nil {
		return "", fmt.Errorf("handshake: %w", err)
	}
	return msg, nil
}
