package proto

import "testing"

func TestResponseTypeConstants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		got  ResponseType
		want ResponseType
	}{
		{"SUCCESS_ATOM", ResponseSuccessAtom, 1},
		{"SUCCESS_SEQUENCE", ResponseSuccessSequence, 2},
		{"SUCCESS_PARTIAL", ResponseSuccessPartial, 3},
		{"WAIT_COMPLETE", ResponseWaitComplete, 4},
		{"SERVER_INFO", ResponseServerInfo, 5},
		{"CLIENT_ERROR", ResponseClientError, 16},
		{"COMPILE_ERROR", ResponseCompileError, 17},
		{"RUNTIME_ERROR", ResponseRuntimeError, 18},
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

func TestResponseTypeIsError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		rt   ResponseType
		want bool
	}{
		{"SUCCESS_ATOM", ResponseSuccessAtom, false},
		{"SUCCESS_SEQUENCE", ResponseSuccessSequence, false},
		{"SUCCESS_PARTIAL", ResponseSuccessPartial, false},
		{"WAIT_COMPLETE", ResponseWaitComplete, false},
		{"SERVER_INFO", ResponseServerInfo, false},
		{"CLIENT_ERROR", ResponseClientError, true},
		{"COMPILE_ERROR", ResponseCompileError, true},
		{"RUNTIME_ERROR", ResponseRuntimeError, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.rt.IsError(); got != tc.want {
				t.Errorf("%s.IsError() = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}
