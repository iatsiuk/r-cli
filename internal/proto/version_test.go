package proto

import "testing"

func TestVersionConstants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		got  Version
		want Version
	}{
		{"V1_0", V1_0, 0x34c2bdc3},
		{"V0_4", V0_4, 0x400c2d20},
		{"V0_3", V0_3, 0x5f75e83e},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.got != tc.want {
				t.Errorf("%s = 0x%08x, want 0x%08x", tc.name, tc.got, tc.want)
			}
		})
	}
}
