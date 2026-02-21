package reql

import (
	"encoding/json"
	"testing"
)

func TestDatumEncoding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		term Term
		want string
	}{
		{"string", Datum("foo"), `"foo"`},
		{"number", Datum(42), `42`},
		{"bool", Datum(true), `true`},
		{"nil", Datum(nil), `null`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tc.term)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestArray(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		term Term
		want string
	}{
		{"simple", Array(10, 20, 30), `[2,[10,20,30]]`},
		{"empty", Array(), `[2,[]]`},
		{"nested", Array(Array(1, 2), 3), `[2,[[2,[1,2]],3]]`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tc.term)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}
