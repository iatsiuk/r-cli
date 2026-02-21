package wire

import (
	"fmt"
	"io"

	"r-cli/internal/proto"
)

// ReadResponse reads a RethinkDB wire frame from r: 12-byte header then payload.
func ReadResponse(r io.Reader) (token uint64, payload []byte, err error) {
	var hdr [12]byte
	if _, err = io.ReadFull(r, hdr[:]); err != nil {
		return 0, nil, fmt.Errorf("read header: %w", err)
	}
	token, length := DecodeHeader(hdr)
	if length > proto.MaxFrameSize {
		return 0, nil, fmt.Errorf("payload length %d exceeds max %d", length, proto.MaxFrameSize)
	}
	payload = make([]byte, length) //nolint:gosec // G115: bounded by proto.MaxFrameSize check above
	if _, err = io.ReadFull(r, payload); err != nil {
		return 0, nil, fmt.Errorf("read payload: %w", err)
	}
	return token, payload, nil
}

// WriteQuery encodes and writes a RethinkDB query frame to w.
func WriteQuery(w io.Writer, token uint64, payload []byte) error {
	frame, err := Encode(token, payload)
	if err != nil {
		return fmt.Errorf("encode query: %w", err)
	}
	if _, err = w.Write(frame); err != nil {
		return fmt.Errorf("write query: %w", err)
	}
	return nil
}
