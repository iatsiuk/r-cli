package parselog

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func setDir(t *testing.T, dir string) {
	t.Helper()
	prev := logDir
	logDir = dir
	t.Cleanup(func() { logDir = prev })
}

func setVersion(t *testing.T, v string) {
	t.Helper()
	prev := logVersion
	logVersion = v
	t.Cleanup(func() { logVersion = prev })
}

type logEntry struct {
	Ts   string `json:"ts"`
	Ver  string `json:"ver"`
	Err  string `json:"err"`
	Expr string `json:"expr"`
}

func readEntries(t *testing.T, logFile string) []logEntry {
	t.Helper()
	f, err := os.Open(logFile)
	if err != nil {
		t.Fatalf("open log: %v", err)
	}
	defer func() { _ = f.Close() }()

	var entries []logEntry
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		var e logEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("unmarshal line %q: %v", line, err)
		}
		entries = append(entries, e)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan log: %v", err)
	}
	return entries
}

func TestLog_AppendsCorrectJSONL(t *testing.T) {
	dir := t.TempDir()
	setDir(t, dir)
	setVersion(t, "v1.2.3")

	before := time.Now().Truncate(time.Second)
	Log("r.table(\"t\")", errors.New("expected ')'"))
	after := time.Now().Add(time.Second).Truncate(time.Second)

	entries := readEntries(t, filepath.Join(dir, logFileName))
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]

	if e.Ver != "v1.2.3" {
		t.Errorf("ver = %q, want v1.2.3", e.Ver)
	}
	if e.Err != "expected ')'" {
		t.Errorf("err = %q, want 'expected )'", e.Err)
	}
	if e.Expr != "r.table(\"t\")" {
		t.Errorf("expr = %q, want r.table(\"t\")", e.Expr)
	}
	ts, err := time.Parse(time.RFC3339, e.Ts)
	if err != nil {
		t.Fatalf("parse ts %q: %v", e.Ts, err)
	}
	tsUTC := ts.UTC()
	if tsUTC.Before(before.UTC()) || tsUTC.After(after.UTC()) {
		t.Errorf("ts %v not in [%v, %v]", tsUTC, before.UTC(), after.UTC())
	}
}

func TestLog_CreatesDirectoryIfMissing(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "nested", "dir")
	setDir(t, dir)

	Log("r.db(\"x\")", errors.New("some error"))

	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	entries := readEntries(t, filepath.Join(dir, logFileName))
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestLog_SilentlyIgnoresWriteErrors(t *testing.T) {
	dir := t.TempDir()
	// make dir read-only so file creation fails
	if err := os.Chmod(dir, 0o500); err != nil { //nolint:gosec // intentional read-only dir for test
		t.Skip("cannot chmod:", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) }) //nolint:gosec // restore dir permissions after test
	setDir(t, dir)

	// must not panic or return error
	Log("r.table(\"x\")", errors.New("boom"))
}

func TestLog_ConcurrentCallsProduceValidJSONL(t *testing.T) {
	dir := t.TempDir()
	setDir(t, dir)

	var wg sync.WaitGroup
	n := 50
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			Log("expr", errors.New("err"))
			_ = i
		}(i)
	}
	wg.Wait()

	entries := readEntries(t, filepath.Join(dir, logFileName))
	if len(entries) != n {
		t.Fatalf("expected %d entries, got %d", n, len(entries))
	}
}

func TestLog_DoesNothingWhenErrIsNil(t *testing.T) {
	dir := t.TempDir()
	setDir(t, dir)

	Log("r.table(\"t\")", nil)

	logFile := filepath.Join(dir, logFileName)
	if _, err := os.Stat(logFile); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("log file should not exist when err is nil")
	}
}

func TestLog_TruncatesLongExpressions(t *testing.T) {
	dir := t.TempDir()
	setDir(t, dir)

	long := strings.Repeat("x", maxExprLen+100)
	Log(long, errors.New("too long"))

	entries := readEntries(t, filepath.Join(dir, logFileName))
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if len(entries[0].Expr) != maxExprLen {
		t.Errorf("expr len = %d, want %d", len(entries[0].Expr), maxExprLen)
	}
}

func TestLog_EscapesSpecialCharsAndUnicode(t *testing.T) {
	dir := t.TempDir()
	setDir(t, dir)

	expr := "line1\nline2\ttab\rreturn\u4e2d\u6587"
	Log(expr, errors.New("parse error"))

	entries := readEntries(t, filepath.Join(dir, logFileName))
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Expr != expr {
		t.Errorf("expr = %q, want %q", entries[0].Expr, expr)
	}
}

func TestLog_UserHomeDirFailureIsSilent(t *testing.T) {
	// simulate HomeDir failure by using empty logDir and blocking resolution
	prev := logDir
	logDir = ""
	prevHome := os.Getenv("HOME")
	_ = os.Unsetenv("HOME")
	t.Cleanup(func() {
		logDir = prev
		_ = os.Setenv("HOME", prevHome)
	})

	// must not panic
	Log("expr", errors.New("oops"))
}

func TestSetDirRestoresPreviousState(t *testing.T) {
	orig := logDir
	// setDir uses t.Cleanup; run in subtest so cleanup fires before we check
	t.Run("inner", func(t *testing.T) {
		setDir(t, "/tmp/test-inner")
		if logDir != "/tmp/test-inner" {
			t.Fatal("dir not set")
		}
	})
	if logDir != orig {
		t.Errorf("logDir not restored: got %q, want %q", logDir, orig)
	}
}

func TestSetVersionRestoresPreviousState(t *testing.T) {
	orig := logVersion
	t.Run("inner", func(t *testing.T) {
		setVersion(t, "vtest")
		if logVersion != "vtest" {
			t.Fatal("version not set")
		}
	})
	if logVersion != orig {
		t.Errorf("logVersion not restored: got %q, want %q", logVersion, orig)
	}
}
