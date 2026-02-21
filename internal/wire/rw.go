package wire

import (
	"fmt"
	"io"
)

// maxFrameSize is the maximum allowed payload size (64MB) to prevent OOM.
const maxFrameSize uint32 = 64 * 1024 * 1024

// ReadResponse reads a RethinkDB wire frame from r: 12-byte header then payload.
func ReadResponse(r io.Reader) (token uint64, payload []byte, err error) {
	var hdr [12]byte
	if _, err = io.ReadFull(r, hdr[:]); err != nil {
		return 0, nil, fmt.Errorf("read header: %w", err)
	}
	token, length := DecodeHeader(hdr)
	if length > maxFrameSize {
		return 0, nil, fmt.Errorf("payload length %d exceeds max %d", length, maxFrameSize)
	}
	payload = make([]byte, length) //nolint:gosec // G115: bounded by maxFrameSize check above
	if _, err = io.ReadFull(r, payload); err != nil {
		return 0, nil, fmt.Errorf("read payload: %w", err)
	}
	return token, payload, nil
}

// WriteQuery encodes and writes a RethinkDB query frame to w.
func WriteQuery(w io.Writer, token uint64, payload []byte) error {
	_, err := w.Write(Encode(token, payload))
	if err != nil {
		return fmt.Errorf("write query: %w", err)
	}
	return nil
}
