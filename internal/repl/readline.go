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
func NewReadlineReader(prompt, historyFile string, out, errOut io.Writer) (Reader, error) {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          prompt,
		HistoryFile:     historyFile,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
		Stdout:          out,
		Stderr:          errOut,
	})
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

func (r *readlineReader) Close() error {
	return r.rl.Close()
}
