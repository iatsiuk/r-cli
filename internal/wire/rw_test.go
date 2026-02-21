package wire

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

// slowReader returns one byte at a time to simulate a slow network connection.
type slowReader struct {
	data []byte
	pos  int
}

func (r *slowReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	p[0] = r.data[r.pos]
	r.pos++
	return 1, nil
}

func TestReadResponse(t *testing.T) {
	t.Parallel()

	token := uint64(42)
	payload := []byte(`[1,"foo",{}]`)
	frame := Encode(token, payload)

	t.Run("basic read from bytes.Reader", func(t *testing.T) {
		t.Parallel()
		gotToken, gotPayload, err := ReadResponse(bytes.NewReader(frame))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotToken != token {
			t.Errorf("token=%d, want %d", gotToken, token)
		}
		if !bytes.Equal(gotPayload, payload) {
			t.Errorf("payload=%q, want %q", gotPayload, payload)
		}
	})

	t.Run("partial data slow reader", func(t *testing.T) {
		t.Parallel()
		gotToken, gotPayload, err := ReadResponse(&slowReader{data: frame})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotToken != token {
			t.Errorf("token=%d, want %d", gotToken, token)
		}
		if !bytes.Equal(gotPayload, payload) {
			t.Errorf("payload=%q, want %q", gotPayload, payload)
		}
	})

	t.Run("EOF mid-header", func(t *testing.T) {
		t.Parallel()
		_, _, err := ReadResponse(bytes.NewReader(frame[:5]))
		if err == nil {
			t.Fatal("expected error for truncated header, got nil")
		}
	})

	t.Run("payload exceeds MaxFrameSize", func(t *testing.T) {
		t.Parallel()
		var hdr [12]byte
		binary.LittleEndian.PutUint64(hdr[0:8], 1)
		binary.LittleEndian.PutUint32(hdr[8:12], maxFrameSize+1)
		_, _, err := ReadResponse(bytes.NewReader(hdr[:]))
		if err == nil {
			t.Fatal("expected error for oversized payload, got nil")
		}
	})
}

func TestWriteQuery(t *testing.T) {
	t.Parallel()

	token := uint64(7)
	payload := []byte(`[1,"bar",{}]`)
	var buf bytes.Buffer
	if err := WriteQuery(&buf, token, payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := Encode(token, payload)
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("got %x, want %x", buf.Bytes(), want)
	}
}
