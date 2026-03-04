// Package parselog appends parser error entries to ~/.r-cli/parser-errors.log in JSONL format.
package parselog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	logFileName = "parser-errors.log"
	maxExprLen  = 4096
)

var (
	mu         sync.Mutex
	logDir     string // empty means resolve from os.UserHomeDir()
	logVersion string
)

// SetVersion sets the version string included in each log entry.
func SetVersion(v string) {
	mu.Lock()
	logVersion = v
	mu.Unlock()
}

// SetDir overrides the log directory (for testing).
func SetDir(path string) {
	mu.Lock()
	logDir = path
	mu.Unlock()
}

// Log appends a JSONL entry to the parser error log. No-op if err is nil.
// All failures are silently ignored.
func Log(expr string, err error) {
	if err == nil {
		return
	}

	mu.Lock()
	dir := logDir
	ver := logVersion
	mu.Unlock()

	if dir == "" {
		home, herr := os.UserHomeDir()
		if herr != nil {
			return
		}
		dir = filepath.Join(home, ".r-cli")
	}

	if len(expr) > maxExprLen {
		expr = expr[:maxExprLen]
	}

	entry := struct {
		Ts   string `json:"ts"`
		Ver  string `json:"ver"`
		Err  string `json:"err"`
		Expr string `json:"expr"`
	}{
		Ts:   time.Now().Format(time.RFC3339),
		Ver:  ver,
		Err:  err.Error(),
		Expr: expr,
	}

	data, merr := json.Marshal(entry)
	if merr != nil {
		return
	}
	data = append(data, '\n')

	mu.Lock()
	defer mu.Unlock()

	if mkErr := os.MkdirAll(dir, 0o700); mkErr != nil {
		return
	}

	f, ferr := os.OpenFile(filepath.Join(dir, logFileName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if ferr != nil {
		return
	}

	_, _ = f.Write(data)
	_ = f.Close()
}
