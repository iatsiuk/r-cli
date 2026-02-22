package output

import (
	"os"
)

// isattyFn allows overriding terminal detection in tests.
var isattyFn = isTerminal

// DetectFormat returns the output format to use. If flagFormat is non-empty it
// is returned directly (explicit flag wins). Otherwise "json" for a TTY stdout
// or "jsonl" for a non-TTY (pipe, redirect, etc.).
func DetectFormat(stdout *os.File, flagFormat string) string {
	if flagFormat != "" {
		return flagFormat
	}
	if isattyFn(stdout) {
		return "json"
	}
	return "jsonl"
}

// NoColor reports whether ANSI color output should be suppressed.
// Returns true when the NO_COLOR environment variable is set (any value).
func NoColor() bool {
	_, ok := os.LookupEnv("NO_COLOR")
	return ok
}

// isTerminal reports whether f is connected to a terminal character device.
func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	mode := fi.Mode()
	return mode&os.ModeDevice != 0 && mode&os.ModeCharDevice != 0
}
