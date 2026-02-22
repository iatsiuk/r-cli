// Package repl provides an interactive REPL for RethinkDB queries.
package repl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
)

// ErrInterrupt is returned by Reader.Readline when the user presses Ctrl+C.
var ErrInterrupt = errors.New("interrupt")

// Reader abstracts line input for testability.
type Reader interface {
	Readline() (string, error)
	SetPrompt(prompt string)
	AddHistory(line string) error
	Close() error
}

// ExecFunc executes a ReQL expression string and writes output to w.
type ExecFunc func(ctx context.Context, expr string, w io.Writer) error

// Config holds REPL construction options.
type Config struct {
	Reader      Reader
	Exec        ExecFunc
	Out         io.Writer
	ErrOut      io.Writer
	InterruptCh <-chan struct{} // receives when user interrupts during query execution
	Prompt      string
}

// Repl is the interactive REPL.
type Repl struct {
	reader      Reader
	exec        ExecFunc
	out         io.Writer
	errOut      io.Writer
	interruptCh <-chan struct{}
	prompt      string
}

// New creates a Repl from Config.
func New(cfg *Config) *Repl {
	prompt := cfg.Prompt
	if prompt == "" {
		prompt = "r> "
	}
	out := cfg.Out
	if out == nil {
		out = io.Discard
	}
	errOut := cfg.ErrOut
	if errOut == nil {
		errOut = io.Discard
	}
	return &Repl{
		reader:      cfg.Reader,
		exec:        cfg.Exec,
		out:         out,
		errOut:      errOut,
		interruptCh: cfg.InterruptCh,
		prompt:      prompt,
	}
}

// Run starts the REPL loop. Returns nil on clean exit (EOF).
func (r *Repl) Run(ctx context.Context) error {
	r.reader.SetPrompt(r.prompt)
	for {
		line, err := r.reader.Readline()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			if errors.Is(err, ErrInterrupt) {
				continue
			}
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		_ = r.reader.AddHistory(line)
		r.runQuery(ctx, line)
	}
}

func (r *Repl) runQuery(ctx context.Context, expr string) {
	queryCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	done := r.watchInterrupt(queryCtx, cancel)

	if err := r.exec(queryCtx, expr, r.out); err != nil {
		if !errors.Is(err, context.Canceled) {
			_, _ = fmt.Fprintln(r.errOut, err)
		}
	}
	cancel() // unblock watchInterrupt goroutine via queryCtx.Done()
	<-done
}

// watchInterrupt starts a goroutine that cancels queryCtx on interrupt.
// Returns a channel closed when the goroutine exits.
// If interruptCh is nil, returns an already-closed channel.
func (r *Repl) watchInterrupt(queryCtx context.Context, cancel context.CancelFunc) <-chan struct{} {
	done := make(chan struct{})
	if r.interruptCh == nil {
		close(done)
		return done
	}
	go func() {
		defer close(done)
		select {
		case <-r.interruptCh:
			cancel()
		case <-queryCtx.Done():
		}
	}()
	return done
}
