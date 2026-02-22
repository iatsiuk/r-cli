package repl

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

// fakeReader is a test Reader that serves lines from a slice, returning EOF when exhausted.
// The sentinel "\x03" triggers ErrInterrupt.
type fakeReader struct {
	lines   []string
	pos     int
	prompt  string
	prompts []string // history of all SetPrompt calls
	history []string
}

func (f *fakeReader) Readline() (string, error) {
	if f.pos >= len(f.lines) {
		return "", io.EOF
	}
	line := f.lines[f.pos]
	f.pos++
	if line == "\x03" {
		return "", ErrInterrupt
	}
	return line, nil
}

func (f *fakeReader) SetPrompt(prompt string) {
	f.prompt = prompt
	f.prompts = append(f.prompts, prompt)
}
func (f *fakeReader) AddHistory(line string) error { f.history = append(f.history, line); return nil }
func (f *fakeReader) Close() error                 { return nil }

func TestReplQueryExecution(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	called := 0

	r := New(&Config{
		Reader: &fakeReader{lines: []string{`r.table("test")`, `r.now()`}},
		Exec: func(_ context.Context, expr string, w io.Writer) error {
			called++
			_, _ = fmt.Fprintln(w, "result:"+expr)
			return nil
		},
		Out:    &out,
		ErrOut: io.Discard,
	})

	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called != 2 {
		t.Errorf("exec called %d times, want 2", called)
	}
	if !strings.Contains(out.String(), `result:r.table("test")`) {
		t.Errorf("output missing first query result: %q", out.String())
	}
	if !strings.Contains(out.String(), "result:r.now()") {
		t.Errorf("output missing second query result: %q", out.String())
	}
}

func TestReplEmptyInput(t *testing.T) {
	t.Parallel()
	called := 0

	r := New(&Config{
		Reader: &fakeReader{lines: []string{"", "   ", "\t"}},
		Exec: func(_ context.Context, _ string, _ io.Writer) error {
			called++
			return nil
		},
		Out:    io.Discard,
		ErrOut: io.Discard,
	})

	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called != 0 {
		t.Errorf("exec called %d times, want 0", called)
	}
}

func TestReplEOF(t *testing.T) {
	t.Parallel()

	r := New(&Config{
		Reader: &fakeReader{},
		Exec:   func(_ context.Context, _ string, _ io.Writer) error { return nil },
		Out:    io.Discard,
		ErrOut: io.Discard,
	})

	if err := r.Run(context.Background()); err != nil {
		t.Errorf("expected nil on EOF, got %v", err)
	}
}

func TestReplCtrlCDuringInput(t *testing.T) {
	t.Parallel()
	called := 0

	r := New(&Config{
		// Ctrl+C first, then a real query, then EOF
		Reader: &fakeReader{lines: []string{"\x03", "\x03", "r.now()"}},
		Exec: func(_ context.Context, _ string, _ io.Writer) error {
			called++
			return nil
		},
		Out:    io.Discard,
		ErrOut: io.Discard,
	})

	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called != 1 {
		t.Errorf("exec called %d times, want 1", called)
	}
}

func TestReplHistorySaved(t *testing.T) {
	t.Parallel()
	fr := &fakeReader{lines: []string{"r.now()", "", "r.dbList()"}}

	r := New(&Config{
		Reader: fr,
		Exec:   func(_ context.Context, _ string, _ io.Writer) error { return nil },
		Out:    io.Discard,
		ErrOut: io.Discard,
	})

	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// empty line must not be saved to history
	want := []string{"r.now()", "r.dbList()"}
	if len(fr.history) != len(want) {
		t.Fatalf("history len %d, want %d: %v", len(fr.history), len(want), fr.history)
	}
	for i, w := range want {
		if fr.history[i] != w {
			t.Errorf("history[%d] = %q, want %q", i, fr.history[i], w)
		}
	}
}

func TestReplMultilineUnclosedParen(t *testing.T) {
	t.Parallel()
	var capturedExpr string
	fr := &fakeReader{
		lines: []string{
			`r.table(`, // incomplete
			`"test")`,  // closes paren
		},
	}

	r := New(&Config{
		Reader: fr,
		Exec: func(_ context.Context, expr string, _ io.Writer) error {
			capturedExpr = expr
			return nil
		},
		Out:    io.Discard,
		ErrOut: io.Discard,
	})

	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// prompt must have switched to continuation prompt
	if !strings.Contains(strings.Join(fr.prompts, ","), contPrompt) {
		t.Errorf("continuation prompt not seen; prompts: %v", fr.prompts)
	}
	if !strings.Contains(capturedExpr, `r.table(`) || !strings.Contains(capturedExpr, `"test")`) {
		t.Errorf("unexpected captured expr: %q", capturedExpr)
	}
}

func TestReplMultilineUnclosedBrace(t *testing.T) {
	t.Parallel()
	var capturedExpr string
	fr := &fakeReader{
		lines: []string{
			`r.table("t").insert({`,
			`"name": "x"})`,
		},
	}

	r := New(&Config{
		Reader: fr,
		Exec: func(_ context.Context, expr string, _ io.Writer) error {
			capturedExpr = expr
			return nil
		},
		Out:    io.Discard,
		ErrOut: io.Discard,
	})

	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(strings.Join(fr.prompts, ","), contPrompt) {
		t.Errorf("continuation prompt not seen; prompts: %v", fr.prompts)
	}
	if !strings.Contains(capturedExpr, `"name"`) {
		t.Errorf("unexpected captured expr: %q", capturedExpr)
	}
}

func TestReplMultilineCompleteQueryExecutes(t *testing.T) {
	t.Parallel()
	var capturedExpr string
	called := 0
	fr := &fakeReader{
		lines: []string{
			`r.table(`,
			`"heroes"`,
			`)`,
		},
	}

	r := New(&Config{
		Reader: fr,
		Exec: func(_ context.Context, expr string, _ io.Writer) error {
			called++
			capturedExpr = expr
			return nil
		},
		Out:    io.Discard,
		ErrOut: io.Discard,
	})

	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if called != 1 {
		t.Errorf("exec called %d times, want 1", called)
	}
	want := "r.table(\n\"heroes\"\n)"
	if capturedExpr != want {
		t.Errorf("capturedExpr = %q, want %q", capturedExpr, want)
	}
	// verify the history entry is the joined multiline expression
	if len(fr.history) != 1 {
		t.Errorf("history len %d, want 1", len(fr.history))
	} else if fr.history[0] != capturedExpr {
		t.Errorf("history[0] = %q, want %q", fr.history[0], capturedExpr)
	}
}

func TestReplDotExit(t *testing.T) {
	t.Parallel()
	called := 0
	r := New(&Config{
		Reader: &fakeReader{lines: []string{".exit", "r.now()"}},
		Exec: func(_ context.Context, _ string, _ io.Writer) error {
			called++
			return nil
		},
		Out:    io.Discard,
		ErrOut: io.Discard,
	})
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called != 0 {
		t.Errorf("exec called %d times after .exit, want 0", called)
	}
}

func TestReplDotQuit(t *testing.T) {
	t.Parallel()
	called := 0
	r := New(&Config{
		Reader: &fakeReader{lines: []string{".quit", "r.now()"}},
		Exec: func(_ context.Context, _ string, _ io.Writer) error {
			called++
			return nil
		},
		Out:    io.Discard,
		ErrOut: io.Discard,
	})
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called != 0 {
		t.Errorf("exec called %d times after .quit, want 0", called)
	}
}

func TestReplDotUse(t *testing.T) {
	t.Parallel()
	var usedDB string
	r := New(&Config{
		Reader: &fakeReader{lines: []string{".use mydb"}},
		Exec:   func(_ context.Context, _ string, _ io.Writer) error { return nil },
		Out:    io.Discard,
		ErrOut: io.Discard,
		OnUseDB: func(db string) {
			usedDB = db
		},
	})
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usedDB != "mydb" {
		t.Errorf("OnUseDB called with %q, want %q", usedDB, "mydb")
	}
}

func TestReplDotFormat(t *testing.T) {
	t.Parallel()
	var setFmt string
	r := New(&Config{
		Reader: &fakeReader{lines: []string{".format jsonl"}},
		Exec:   func(_ context.Context, _ string, _ io.Writer) error { return nil },
		Out:    io.Discard,
		ErrOut: io.Discard,
		OnFormat: func(format string) {
			setFmt = format
		},
	})
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if setFmt != "jsonl" {
		t.Errorf("OnFormat called with %q, want %q", setFmt, "jsonl")
	}
}

func TestReplDotHelp(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	r := New(&Config{
		Reader: &fakeReader{lines: []string{".help"}},
		Exec:   func(_ context.Context, _ string, _ io.Writer) error { return nil },
		Out:    &out,
		ErrOut: io.Discard,
	})
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := out.String()
	for _, want := range []string{".exit", ".quit", ".use", ".format", ".help"} {
		if !strings.Contains(output, want) {
			t.Errorf(".help output missing %q; got: %q", want, output)
		}
	}
}

func TestReplReadlineError(t *testing.T) {
	t.Parallel()
	readErr := fmt.Errorf("terminal read error")
	r := New(&Config{
		Reader: &errorReader{err: readErr},
		Exec:   func(_ context.Context, _ string, _ io.Writer) error { return nil },
		Out:    io.Discard,
		ErrOut: io.Discard,
	})
	err := r.Run(context.Background())
	if err == nil {
		t.Fatal("expected error from Run, got nil")
	}
	if err.Error() != readErr.Error() {
		t.Errorf("Run error = %v, want %v", err, readErr)
	}
}

// errorReader returns a fixed error from Readline (not EOF, not interrupt).
type errorReader struct {
	err error
}

func (e *errorReader) Readline() (string, error) { return "", e.err }
func (e *errorReader) SetPrompt(_ string)        {}
func (e *errorReader) AddHistory(_ string) error { return nil }
func (e *errorReader) Close() error              { return nil }

func TestReplRunQueryExecError(t *testing.T) {
	t.Parallel()
	var errOut bytes.Buffer
	execErr := fmt.Errorf("query execution failed")
	called := 0

	r := New(&Config{
		Reader: &fakeReader{lines: []string{"r.now()", "r.dbList()"}},
		Exec: func(_ context.Context, expr string, _ io.Writer) error {
			called++
			if called == 1 {
				return execErr
			}
			return nil
		},
		Out:    io.Discard,
		ErrOut: &errOut,
	})

	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("unexpected Run error: %v", err)
	}
	if called != 2 {
		t.Errorf("exec called %d times, want 2", called)
	}
	if !strings.Contains(errOut.String(), execErr.Error()) {
		t.Errorf("errOut missing exec error message: %q", errOut.String())
	}
}

func TestReplDotUseNoArg(t *testing.T) {
	t.Parallel()
	var errOut bytes.Buffer
	called := false
	r := New(&Config{
		Reader: &fakeReader{lines: []string{".use"}},
		Exec:   func(_ context.Context, _ string, _ io.Writer) error { return nil },
		Out:    io.Discard,
		ErrOut: &errOut,
		OnUseDB: func(_ string) {
			called = true
		},
	})
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("OnUseDB should not be called when .use has no argument")
	}
	if !strings.Contains(errOut.String(), "usage:") {
		t.Errorf("expected usage message in errOut, got: %q", errOut.String())
	}
}

func TestReplDotFormatNoArg(t *testing.T) {
	t.Parallel()
	var errOut bytes.Buffer
	called := false
	r := New(&Config{
		Reader: &fakeReader{lines: []string{".format"}},
		Exec:   func(_ context.Context, _ string, _ io.Writer) error { return nil },
		Out:    io.Discard,
		ErrOut: &errOut,
		OnFormat: func(_ string) {
			called = true
		},
	})
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("OnFormat should not be called when .format has no argument")
	}
	if !strings.Contains(errOut.String(), "usage:") {
		t.Errorf("expected usage message in errOut, got: %q", errOut.String())
	}
}

func TestReplDotUnknownCommand(t *testing.T) {
	t.Parallel()
	var errOut bytes.Buffer
	called := 0
	r := New(&Config{
		Reader: &fakeReader{lines: []string{".foo", "r.now()"}},
		Exec: func(_ context.Context, _ string, _ io.Writer) error {
			called++
			return nil
		},
		Out:    io.Discard,
		ErrOut: &errOut,
	})
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called != 1 {
		t.Errorf("exec called %d times, want 1 (unknown command should not exit)", called)
	}
	if !strings.Contains(errOut.String(), "unknown command") {
		t.Errorf("expected 'unknown command' in errOut, got: %q", errOut.String())
	}
}

func TestReplCtrlCDuringMultiline(t *testing.T) {
	t.Parallel()
	var capturedExprs []string
	fr := &fakeReader{
		// start multiline, then Ctrl+C to abort, then a fresh query
		lines: []string{"r.table(", "\x03", "r.now()"},
	}

	r := New(&Config{
		Reader: fr,
		Exec: func(_ context.Context, expr string, _ io.Writer) error {
			capturedExprs = append(capturedExprs, expr)
			return nil
		},
		Out:    io.Discard,
		ErrOut: io.Discard,
	})

	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// only r.now() should have been executed; r.table( was aborted
	if len(capturedExprs) != 1 || capturedExprs[0] != "r.now()" {
		t.Errorf("capturedExprs = %v, want [r.now()]", capturedExprs)
	}
	// prompt should have reset to primary after Ctrl+C
	primaryCount := 0
	for _, p := range fr.prompts {
		if p == "r> " {
			primaryCount++
		}
	}
	if primaryCount < 2 {
		t.Errorf("primary prompt should appear at least twice (start + after Ctrl+C); prompts: %v", fr.prompts)
	}
}

func TestIsCompleteEscapeInString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		complete bool
	}{
		{`r.filter({"k": "v\"1"})`, true},
		{`r.filter({"k": "v\"1"`, false},
		{`r.insert({"name": "O\'Brien"})`, true},
		{`r.insert({"name": "O\'Brien"`, false},
	}
	for _, tc := range tests {
		got := isComplete(tc.input)
		if got != tc.complete {
			t.Errorf("isComplete(%q) = %v, want %v", tc.input, got, tc.complete)
		}
	}
}

func TestReplCtrlCDuringExecution(t *testing.T) {
	t.Parallel()

	intCh := make(chan struct{}, 1)
	execStarted := make(chan struct{})

	r := New(&Config{
		// one query then EOF
		Reader: &fakeReader{lines: []string{`r.table("test")`}},
		Exec: func(ctx context.Context, _ string, _ io.Writer) error {
			close(execStarted)
			<-ctx.Done()
			return ctx.Err()
		},
		Out:         io.Discard,
		ErrOut:      io.Discard,
		InterruptCh: intCh,
	})

	done := make(chan error, 1)
	go func() { done <- r.Run(context.Background()) }()

	// wait for exec to start, then send interrupt
	select {
	case <-execStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("exec did not start")
	}
	intCh <- struct{}{}

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("REPL did not exit after interrupt")
	}
}
