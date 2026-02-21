package wire

import (
	"encoding/binary"
	"fmt"
)

// Encode builds a RethinkDB wire frame: 8-byte LE token + 4-byte LE payload length + payload.
// Returns an error if len(payload) exceeds maxFrameSize (64MB).
func Encode(token uint64, payload []byte) ([]byte, error) {
	if len(payload) > int(maxFrameSize) {
		return nil, fmt.Errorf("payload length %d exceeds max %d", len(payload), maxFrameSize)
	}
	frame := make([]byte, 12+len(payload))
	binary.LittleEndian.PutUint64(frame[0:8], token)
	binary.LittleEndian.PutUint32(frame[8:12], uint32(len(payload))) //nolint:gosec // G115: payload length is protocol-bounded, always < 64MB
	copy(frame[12:], payload)
	return frame, nil
}

// DecodeHeader parses a 12-byte wire frame header into token and payload length.
func DecodeHeader(data [12]byte) (token uint64, length uint32) {
	token = binary.LittleEndian.Uint64(data[0:8])
	length = binary.LittleEndian.Uint32(data[8:12])
	return token, length
}
