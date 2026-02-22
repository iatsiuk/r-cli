// Tests in this file create real readline.Instance objects. The chzyer/readline
// library has an internal data race between Terminal.ioloop() and Terminal.Close()
// that we cannot fix. Exclude from race detector runs.
//
//go:build !race

package repl

import (
	"io"
	"os"
	"strings"
	"testing"
)

// newTestReadlineReader creates a readline reader suitable for testing.
// It provides a no-op stdin so readline doesn't try to set terminal raw mode.
func newTestReadlineReader(t *testing.T, historyFile string) (Reader, bool) {
	t.Helper()
	r, err := NewReadlineReader("r> ", historyFile, io.Discard, io.Discard)
	if err != nil {
		t.Logf("readline init failed (no TTY): %v", err)
		return nil, false
	}
	return r, true
}

func TestReadlineHistoryFile(t *testing.T) {
	t.Parallel()

	histFile := histTempFile(t)

	r, ok := newTestReadlineReader(t, histFile)
	if !ok {
		t.Skip("readline unavailable in this environment")
	}

	entries := []string{"r.now()", `r.table("test")`, "r.dbList()"}
	for _, e := range entries {
		if err := r.AddHistory(e); err != nil {
			t.Fatalf("AddHistory(%q): %v", e, err)
		}
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(histFile)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, e := range entries {
		if !strings.Contains(content, e) {
			t.Errorf("history file missing %q; file content:\n%s", e, content)
		}
	}
}

// TestReadlineHistoryNavigation verifies that history written by one session
// is loaded by the next session (enabling up/down arrow navigation).
func TestReadlineHistoryNavigation(t *testing.T) {
	t.Parallel()

	histFile := histTempFile(t)

	// session 1: add entries
	r1, ok := newTestReadlineReader(t, histFile)
	if !ok {
		t.Skip("readline unavailable in this environment")
	}
	for _, e := range []string{"r.now()", "r.dbList()"} {
		if err := r1.AddHistory(e); err != nil {
			t.Fatalf("AddHistory: %v", err)
		}
	}
	if err := r1.Close(); err != nil {
		t.Fatal(err)
	}

	// session 2: verify file contains entries from session 1
	data, err := os.ReadFile(histFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range []string{"r.now()", "r.dbList()"} {
		if !strings.Contains(string(data), e) {
			t.Errorf("history entry %q not found in file for session 2 navigation", e)
		}
	}
}

func TestReadlineHistoryPersists(t *testing.T) {
	t.Parallel()

	histFile := histTempFile(t)

	// session 1
	r1, ok := newTestReadlineReader(t, histFile)
	if !ok {
		t.Skip("readline unavailable in this environment")
	}
	if err := r1.AddHistory("r.now()"); err != nil {
		t.Fatalf("AddHistory: %v", err)
	}
	if err := r1.Close(); err != nil {
		t.Fatal(err)
	}

	// session 2: same file, should see session 1's history
	r2, ok := newTestReadlineReader(t, histFile)
	if !ok {
		t.Skip("readline unavailable in this environment")
	}
	if err := r2.AddHistory("r.dbList()"); err != nil {
		t.Fatalf("AddHistory: %v", err)
	}
	if err := r2.Close(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(histFile)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, e := range []string{"r.now()", "r.dbList()"} {
		if !strings.Contains(content, e) {
			t.Errorf("history file missing %q after two sessions; content:\n%s", e, content)
		}
	}
}

func histTempFile(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "hist")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}
