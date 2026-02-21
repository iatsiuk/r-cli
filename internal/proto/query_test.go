package proto

import "testing"

func TestQueryTypeConstants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		got  QueryType
		want QueryType
	}{
		{"START", QueryStart, 1},
		{"CONTINUE", QueryContinue, 2},
		{"STOP", QueryStop, 3},
		{"NOREPLY_WAIT", QueryNoreplyWait, 4},
		{"SERVER_INFO", QueryServerInfo, 5},
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
