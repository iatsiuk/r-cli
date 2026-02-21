package proto

import "testing"

func TestDatumTypeConstants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		got  DatumType
		want DatumType
	}{
		{"R_NULL", DatumNull, 1},
		{"R_BOOL", DatumBool, 2},
		{"R_NUM", DatumNum, 3},
		{"R_STR", DatumStr, 4},
		{"R_ARRAY", DatumArray, 5},
		{"R_OBJECT", DatumObject, 6},
		{"R_JSON", DatumJSON, 7},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.got != tc.want {
				t.Errorf("%s = %d, want %d", tc.name, tc.got, tc.want)
			}
		})
	}
}
