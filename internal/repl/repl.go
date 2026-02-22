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
	OnUseDB     func(db string)     // called when .use <db> is executed
	OnFormat    func(format string) // called when .format <fmt> is executed
}

// Repl is the interactive REPL.
type Repl struct {
	reader      Reader
	exec        ExecFunc
	out         io.Writer
	errOut      io.Writer
	interruptCh <-chan struct{}
	prompt      string
	onUseDB     func(db string)
	onFormat    func(format string)
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
	onUseDB := cfg.OnUseDB
	if onUseDB == nil {
		onUseDB = func(string) {}
	}
	onFormat := cfg.OnFormat
	if onFormat == nil {
		onFormat = func(string) {}
	}
	return &Repl{
		reader:      cfg.Reader,
		exec:        cfg.Exec,
		out:         out,
		errOut:      errOut,
		interruptCh: cfg.InterruptCh,
		prompt:      prompt,
		onUseDB:     onUseDB,
		onFormat:    onFormat,
	}
}

const contPrompt = "... "

// Run starts the REPL loop. Returns nil on clean exit (EOF).
func (r *Repl) Run(ctx context.Context) error {
	r.reader.SetPrompt(r.prompt)
	var lines []string
	for {
		line, err := r.reader.Readline()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			if errors.Is(err, ErrInterrupt) {
				lines = lines[:0]
				r.reader.SetPrompt(r.prompt)
				continue
			}
			return err
		}

		if len(lines) == 0 {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, ".") {
				if r.dotCommand(line) {
					return nil
				}
				continue
			}
		}

		lines = append(lines, line)
		input := strings.Join(lines, "\n")

		if !isComplete(input) {
			r.reader.SetPrompt(contPrompt)
			continue
		}

		r.reader.SetPrompt(r.prompt)
		lines = lines[:0]

		expr := strings.TrimSpace(input)
		_ = r.reader.AddHistory(expr)
		r.runQuery(ctx, expr)
	}
}

// isComplete returns true when all parentheses, braces, and brackets are balanced.
// Bracket characters inside string literals are ignored.
func isComplete(s string) bool {
	depth := 0
	inStr := false
	strChar := byte(0)
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inStr {
			if ch == '\\' {
				i++
				continue
			}
			if ch == strChar {
				inStr = false
			}
			continue
		}
		switch ch {
		case '"', '\'':
			inStr = true
			strChar = ch
		case '(', '{', '[':
			depth++
		case ')', '}', ']':
			depth--
		}
	}
	return depth <= 0 && !inStr
}

// dotCommand dispatches a REPL dot-command. Returns true if the REPL should exit.
func (r *Repl) dotCommand(line string) bool {
	parts := strings.Fields(line)
	switch parts[0] {
	case ".exit", ".quit":
		return true
	case ".use":
		if len(parts) < 2 {
			_, _ = fmt.Fprintln(r.errOut, "usage: .use <database>")
			return false
		}
		r.onUseDB(parts[1])
	case ".format":
		if len(parts) < 2 {
			_, _ = fmt.Fprintln(r.errOut, "usage: .format <json|jsonl|raw|table>")
			return false
		}
		r.onFormat(parts[1])
	case ".help":
		_, _ = fmt.Fprintln(r.out, "Available commands:")
		_, _ = fmt.Fprintln(r.out, "  .exit, .quit          exit the REPL")
		_, _ = fmt.Fprintln(r.out, "  .use <database>       change current database")
		_, _ = fmt.Fprintln(r.out, "  .format <fmt>         set output format (json|jsonl|raw|table)")
		_, _ = fmt.Fprintln(r.out, "  .help                 show this help")
	default:
		_, _ = fmt.Fprintf(r.errOut, "unknown command: %s\n", parts[0])
	}
	return false
}

func (r *Repl) runQuery(ctx context.Context, expr string) {
	// drain any stale interrupt signal queued while readline was waiting for input
	for len(r.interruptCh) > 0 {
		<-r.interruptCh
	}
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
