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

func (f *fakeReader) SetPrompt(prompt string)      { f.prompt = prompt }
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
