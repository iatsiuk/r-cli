package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"r-cli/internal/output"
	"r-cli/internal/query"
	"r-cli/internal/repl"
	"r-cli/internal/reql"
	"r-cli/internal/reql/parser"
)

// replStart is the function used to launch the REPL; replaced in tests.
var replStart = runREPL

func newReplCmd(cfg *rootConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "repl",
		Short: "Start an interactive REPL",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return replStart(cmd.Context(), cfg, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
}

// runREPL creates a readline reader, connects to RethinkDB, and runs the REPL loop.
func runREPL(ctx context.Context, cfg *rootConfig, out, errOut io.Writer) error {
	exec, cleanup, err := newExecutor(cfg)
	if err != nil {
		return err
	}
	defer cleanup()

	localCfg := *cfg
	completer := &repl.Completer{
		FetchDBs:    makeFetchDBs(exec),
		FetchTables: makeFetchTables(exec, &localCfg),
	}
	completer.SetCurrentDB(cfg.database)

	historyFile := replHistoryFile()
	interruptCh := make(chan struct{}, 1)
	notifyInterrupt := func() {
		select {
		case interruptCh <- struct{}{}:
		default:
		}
	}
	reader, err := repl.NewReadlineReader("r> ", historyFile, out, errOut, notifyInterrupt, completer)
	if err != nil {
		return err
	}

	var once sync.Once
	closeReader := func() { once.Do(func() { _ = reader.Close() }) }
	defer closeReader()

	// replCtx is independent of ctx so that OS SIGINT during query execution
	// cancels only the current query (via interruptCh) without killing the REPL loop.
	replCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// route OS SIGINT to interruptCh; readline handles Ctrl+C in raw mode itself.
	sigIntCh := make(chan os.Signal, 1)
	signal.Notify(sigIntCh, os.Interrupt)
	defer signal.Stop(sigIntCh)

	// close readline on SIGTERM for graceful exit.
	sigTermCh := make(chan os.Signal, 1)
	signal.Notify(sigTermCh, syscall.SIGTERM)
	defer signal.Stop(sigTermCh)

	go func() {
		for {
			select {
			case <-sigIntCh:
				notifyInterrupt()
			case <-sigTermCh:
				closeReader()
				return
			case <-replCtx.Done():
				return
			}
		}
	}()

	r := repl.New(&repl.Config{
		Reader:      reader,
		Exec:        makeReplExec(exec, &localCfg),
		Out:         out,
		ErrOut:      errOut,
		InterruptCh: interruptCh,
		OnUseDB: func(db string) {
			localCfg.database = db
			completer.SetCurrentDB(db)
		},
		OnFormat: func(format string) {
			localCfg.format = format
		},
	})
	return r.Run(replCtx)
}

// makeReplExec returns an ExecFunc that parses and executes a ReQL expression.
func makeReplExec(exec *query.Executor, cfg *rootConfig) repl.ExecFunc {
	return func(ctx context.Context, expr string, w io.Writer) error {
		term, err := parser.Parse(expr)
		if err != nil {
			return err
		}
		start := time.Now()
		profile, cur, err := exec.Run(ctx, term, buildQueryOpts(cfg))
		if err != nil {
			return err
		}
		writeQueryMeta(cfg, profile, time.Since(start))
		if cur == nil {
			return nil
		}
		defer func() { _ = cur.Close() }()
		return writeOutput(w, output.DetectFormat(os.Stdout, cfg.format), makeIter(cur, cfg))
	}
}

func makeFetchDBs(exec *query.Executor) func(context.Context) ([]string, error) {
	return func(ctx context.Context) ([]string, error) {
		_, cur, err := exec.Run(ctx, reql.DBList(), reql.OptArgs{})
		if err != nil {
			return nil, err
		}
		defer func() { _ = cur.Close() }()
		rows, err := cur.All()
		if err != nil {
			return nil, err
		}
		return jsonRowsToStrings(rows), nil
	}
}

func makeFetchTables(exec *query.Executor, cfg *rootConfig) func(context.Context, string) ([]string, error) {
	return func(ctx context.Context, db string) ([]string, error) {
		if db == "" {
			db = cfg.database
		}
		if db == "" {
			return nil, nil
		}
		_, cur, err := exec.Run(ctx, reql.DB(db).TableList(), reql.OptArgs{})
		if err != nil {
			return nil, err
		}
		defer func() { _ = cur.Close() }()
		rows, err := cur.All()
		if err != nil {
			return nil, err
		}
		return jsonRowsToStrings(rows), nil
	}
}

// jsonRowsToStrings unmarshals each JSON row as a string, skipping failures.
func jsonRowsToStrings(rows []json.RawMessage) []string {
	var names []string
	for _, row := range rows {
		var name string
		if json.Unmarshal(row, &name) == nil {
			names = append(names, name)
		}
	}
	return names
}

// replHistoryFile returns the path to the REPL history file in the user's home dir.
func replHistoryFile() string {
	u, err := user.Current()
	if err != nil {
		return ""
	}
	return filepath.Join(u.HomeDir, ".r-cli_history")
}
