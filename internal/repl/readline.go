package repl

import (
	"errors"
	"io"

	"github.com/chzyer/readline"
)

// readlineReader wraps *readline.Instance to implement Reader.
type readlineReader struct {
	rl *readline.Instance
}

// NewReadlineReader creates a Reader backed by github.com/chzyer/readline.
// interruptHook is called (non-blocking) when Ctrl+C is pressed; pass nil to disable.
// An optional TabCompleter may be passed to enable tab completion.
func NewReadlineReader(prompt, historyFile string, out, errOut io.Writer, interruptHook func(), completer ...TabCompleter) (Reader, error) {
	var ac readline.AutoCompleter
	if len(completer) > 0 && completer[0] != nil {
		ac = completer[0]
	}
	cfg := &readline.Config{
		Prompt:                 prompt,
		HistoryFile:            historyFile,
		DisableAutoSaveHistory: true,
		InterruptPrompt:        "^C",
		EOFPrompt:              "exit",
		Stdout:                 out,
		Stderr:                 errOut,
		AutoComplete:           ac,
	}
	if interruptHook != nil {
		cfg.FuncFilterInputRune = func(r rune) (rune, bool) {
			if r == readline.CharInterrupt {
				interruptHook()
			}
			return r, true
		}
	}
	rl, err := readline.NewEx(cfg)
	if err != nil {
		return nil, err
	}
	return &readlineReader{rl: rl}, nil
}

func (r *readlineReader) Readline() (string, error) {
	line, err := r.rl.Readline()
	if errors.Is(err, readline.ErrInterrupt) {
		return "", ErrInterrupt
	}
	return line, err
}

func (r *readlineReader) SetPrompt(prompt string) {
	r.rl.SetPrompt(prompt)
}

func (r *readlineReader) AddHistory(line string) error {
	return r.rl.SaveHistory(line)
}

func (r *readlineReader) Close() error {
	return r.rl.Close()
}
