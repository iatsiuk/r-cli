package conn

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestReadNullTerminatedBasic(t *testing.T) {
	t.Parallel()

	t.Run("reads until null terminator", func(t *testing.T) {
		t.Parallel()
		r := bytes.NewReader([]byte("hello\x00extra"))
		got, err := readNullTerminated(r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(got) != "hello" {
			t.Fatalf("got %q, want %q", got, "hello")
		}
	})

	t.Run("returns empty slice for immediate null", func(t *testing.T) {
		t.Parallel()
		r := bytes.NewReader([]byte("\x00"))
		got, err := readNullTerminated(r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("got %q, want empty", got)
		}
	})

	t.Run("partial reads (1-byte chunks)", func(t *testing.T) {
		t.Parallel()
		r := &oneByteReader{data: []byte("ab\x00")}
		got, err := readNullTerminated(r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(got) != "ab" {
			t.Fatalf("got %q, want %q", got, "ab")
		}
	})
}

func TestReadNullTerminatedErrors(t *testing.T) {
	t.Parallel()

	t.Run("EOF before null terminator", func(t *testing.T) {
		t.Parallel()
		r := bytes.NewReader([]byte("no-null"))
		_, err := readNullTerminated(r)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "EOF") {
			t.Fatalf("expected EOF error, got %v", err)
		}
	})

	t.Run("empty reader returns EOF error", func(t *testing.T) {
		t.Parallel()
		r := bytes.NewReader([]byte{})
		_, err := readNullTerminated(r)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("exceeds maxHandshakeSize", func(t *testing.T) {
		t.Parallel()
		data := make([]byte, maxHandshakeSize+2)
		for i := range data {
			data[i] = 'x'
		}
		data[len(data)-1] = 0x00
		r := bytes.NewReader(data)
		_, err := readNullTerminated(r)
		if err == nil {
			t.Fatal("expected error for oversized message, got nil")
		}
		if !strings.Contains(err.Error(), "exceeds") {
			t.Fatalf("expected 'exceeds' error, got %v", err)
		}
	})
}

func TestWriteNullTerminated(t *testing.T) {
	t.Parallel()

	t.Run("appends null terminator", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		if err := writeNullTerminated(&buf, []byte("hello")); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(buf.Bytes(), []byte("hello\x00")) {
			t.Fatalf("got %v, want hello\\x00", buf.Bytes())
		}
	})

	t.Run("empty data writes only null", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		if err := writeNullTerminated(&buf, []byte{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := buf.Bytes()
		if len(got) != 1 || got[0] != 0x00 {
			t.Fatalf("got %v, want [0x00]", got)
		}
	})

	t.Run("write error propagated", func(t *testing.T) {
		t.Parallel()
		err := writeNullTerminated(&errWriter{}, []byte("data"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

// oneByteReader reads exactly one byte at a time.
type oneByteReader struct {
	data []byte
	pos  int
}

func (r *oneByteReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	p[0] = r.data[r.pos]
	r.pos++
	return 1, nil
}

// errWriter always returns an error on Write.
type errWriter struct{}

func (w *errWriter) Write(_ []byte) (int, error) {
	return 0, io.ErrClosedPipe
}
