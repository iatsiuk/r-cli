package output

import (
	"os"
	"testing"
)

func TestDetectFormatTTY(t *testing.T) {
	orig := isattyFn
	defer func() { isattyFn = orig }()
	isattyFn = func(*os.File) bool { return true }

	if got := DetectFormat(nil, ""); got != "json" {
		t.Errorf("expected json for TTY, got %q", got)
	}
}

func TestDetectFormatNonTTY(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() }) //nolint:errcheck
	t.Cleanup(func() { w.Close() }) //nolint:errcheck

	if got := DetectFormat(w, ""); got != "jsonl" {
		t.Errorf("expected jsonl for non-TTY pipe, got %q", got)
	}
}

func TestDetectFormatFlagOverride(t *testing.T) {
	orig := isattyFn
	defer func() { isattyFn = orig }()

	for _, flag := range []string{"json", "jsonl", "raw", "table"} {
		// test with TTY to confirm flag wins over detection
		isattyFn = func(*os.File) bool { return true }
		if got := DetectFormat(nil, flag); got != flag {
			t.Errorf("flag %q: expected %q, got %q", flag, flag, got)
		}
		// test with non-TTY to confirm flag wins over detection
		isattyFn = func(*os.File) bool { return false }
		if got := DetectFormat(nil, flag); got != flag {
			t.Errorf("flag %q (non-tty): expected %q, got %q", flag, flag, got)
		}
	}
}

func TestNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if !NoColor() {
		t.Error("expected NoColor() true when NO_COLOR env var is set")
	}
}

func TestNoColorUnset(t *testing.T) {
	os.Unsetenv("NO_COLOR") //nolint:errcheck
	if NoColor() {
		t.Error("expected NoColor() false when NO_COLOR env var is not set")
	}
}
