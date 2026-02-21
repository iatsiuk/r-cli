package wire

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestEncode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		token   uint64
		payload []byte
		want    []byte
	}{
		{
			name:    "basic query payload",
			token:   1,
			payload: []byte(`[1,"foo",{}]`),
			// token=1 LE: 01 00 00 00 00 00 00 00
			// len=12 LE: 0c 00 00 00
			want: append([]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0c, 0x00, 0x00, 0x00}, []byte(`[1,"foo",{}]`)...),
		},
		{
			name:    "zero token",
			token:   0,
			payload: []byte(`[]`),
			want:    []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, '[', ']'},
		},
		{
			name:    "large payload length field",
			token:   0xdeadbeefcafe1234,
			payload: bytes.Repeat([]byte("x"), 1024),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := Encode(tc.token, tc.payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(got) != 12+len(tc.payload) {
				t.Fatalf("len=%d, want %d", len(got), 12+len(tc.payload))
			}

			// verify token bytes using binary package to avoid unsafe integer casts
			var wantTokenBytes [8]byte
			binary.LittleEndian.PutUint64(wantTokenBytes[:], tc.token)
			if !bytes.Equal(got[:8], wantTokenBytes[:]) {
				t.Errorf("token bytes %x, want %x", got[:8], wantTokenBytes)
			}

			// verify length field
			gotLen := binary.LittleEndian.Uint32(got[8:12])
			if int(gotLen) != len(tc.payload) {
				t.Errorf("length field=%d, want %d", gotLen, len(tc.payload))
			}

			// verify payload
			if !bytes.Equal(got[12:], tc.payload) {
				t.Errorf("payload mismatch")
			}

			// verify exact bytes for cases with known want
			if tc.want != nil && !bytes.Equal(got, tc.want) {
				t.Errorf("got %x, want %x", got, tc.want)
			}
		})
	}
}

func TestDecodeHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		data       [12]byte
		wantToken  uint64
		wantLength uint32
	}{
		{
			name:       "token=1 length=12",
			data:       [12]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0c, 0x00, 0x00, 0x00},
			wantToken:  1,
			wantLength: 12,
		},
		{
			name:       "zero header",
			data:       [12]byte{},
			wantToken:  0,
			wantLength: 0,
		},
		{
			// DecodeHeader takes a fixed [12]byte so insufficient bytes is a compile-time check;
			// this test verifies a large token value decodes correctly.
			name: "large token and length",
			data: [12]byte{
				0x34, 0x12, 0xfe, 0xca, 0xef, 0xbe, 0xad, 0xde,
				0x00, 0x04, 0x00, 0x00,
			},
			wantToken:  0xdeadbeefcafe1234,
			wantLength: 1024,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			token, length := DecodeHeader(tc.data)
			if token != tc.wantToken {
				t.Errorf("token=%d, want %d", token, tc.wantToken)
			}
			if length != tc.wantLength {
				t.Errorf("length=%d, want %d", length, tc.wantLength)
			}
		})
	}
}

func TestEncodeDecodeRoundtrip(t *testing.T) {
	t.Parallel()

	token := uint64(42)
	payload := []byte(`[1,"foo",{}]`)
	frame, err := Encode(token, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var hdr [12]byte
	copy(hdr[:], frame[:12])
	gotToken, gotLen := DecodeHeader(hdr)

	if gotToken != token {
		t.Errorf("token=%d, want %d", gotToken, token)
	}
	if int(gotLen) != len(payload) {
		t.Errorf("length=%d, want %d", gotLen, len(payload))
	}
}
