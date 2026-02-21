package conn

import (
	"errors"
	"fmt"
	"io"
)

const maxHandshakeSize = 16 * 1024 // 16KB, prevent OOM during handshake

// readNullTerminated reads bytes from r until \x00, returning data without the terminator.
func readNullTerminated(r io.Reader) ([]byte, error) {
	var buf []byte
	b := make([]byte, 1)
	for {
		_, err := r.Read(b)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, fmt.Errorf("readNullTerminated: unexpected EOF")
			}
			return nil, fmt.Errorf("readNullTerminated: %w", err)
		}
		if b[0] == 0x00 {
			return buf, nil
		}
		buf = append(buf, b[0])
		if len(buf) > maxHandshakeSize {
			return nil, fmt.Errorf("readNullTerminated: message exceeds %d bytes", maxHandshakeSize)
		}
	}
}

// writeNullTerminated writes data followed by a null terminator to w.
func writeNullTerminated(w io.Writer, data []byte) error {
	out := make([]byte, len(data)+1)
	copy(out, data)
	out[len(data)] = 0x00
	if _, err := w.Write(out); err != nil {
		return fmt.Errorf("writeNullTerminated: %w", err)
	}
	return nil
}
