package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"r-cli/internal/query"
	"r-cli/internal/reql"
)

type insertConfig struct {
	file      string
	batchSize int
	conflict  string
}

type insertResult struct {
	Inserted int64 `json:"inserted"`
	Errors   int64 `json:"errors"`
}

func newInsertCmd(cfg *rootConfig) *cobra.Command {
	ic := &insertConfig{}
	cmd := &cobra.Command{
		Use:   "insert <db.table>",
		Short: "Bulk insert documents into a table",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dbName, tableName, err := parseTableRef(args[0])
			if err != nil {
				return err
			}
			src, closer, err := openInputSource(ic.file, os.Stdin)
			if err != nil {
				return err
			}
			defer closer()
			return runInsert(cmd.Context(), cfg, ic, dbName, tableName, src, os.Stdout)
		},
	}
	cmd.Flags().StringVarP(&ic.file, "file", "F", "", "input file (default: stdin)")
	cmd.Flags().IntVar(&ic.batchSize, "batch-size", 200, "documents per insert batch")
	cmd.Flags().StringVar(&ic.conflict, "conflict", "error", "conflict strategy: error, replace, update")
	return cmd
}

// parseTableRef splits "db.table" into db and table names.
func parseTableRef(ref string) (db, table string, err error) {
	parts := strings.SplitN(ref, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid table reference %q: expected db.table", ref)
	}
	return parts[0], parts[1], nil
}

// openInputSource returns a reader for the named file, or stdin if file is empty.
func openInputSource(file string, stdin io.Reader) (io.Reader, func(), error) {
	if file == "" {
		return stdin, func() {}, nil
	}
	f, err := os.Open(file)
	if err != nil {
		return nil, nil, fmt.Errorf("opening input file: %w", err)
	}
	return f, func() { _ = f.Close() }, nil
}

// detectInputFormat infers format from the --format flag or file extension; defaults to jsonl.
func detectInputFormat(file, flagFormat string) string {
	if flagFormat == "json" || flagFormat == "jsonl" {
		return flagFormat
	}
	if filepath.Ext(file) == ".json" {
		return "json"
	}
	return "jsonl"
}

// runInsert reads documents from r and bulk-inserts them into db.table.
func runInsert(ctx context.Context, cfg *rootConfig, ic *insertConfig, dbName, tableName string, r io.Reader, out io.Writer) error {
	if ic.batchSize < 1 {
		return fmt.Errorf("--batch-size must be >= 1")
	}
	if cfg.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.timeout)
		defer cancel()
	}

	format := detectInputFormat(ic.file, cfg.format)
	opts := reql.OptArgs{"conflict": ic.conflict}
	tbl := reql.DB(dbName).Table(tableName)

	exec, cleanup := newExecutor(cfg)
	defer cleanup()

	var total insertResult
	var err error
	if format == "json" {
		err = insertJSON(ctx, exec, cfg, tbl, opts, ic.batchSize, r, &total)
	} else {
		err = insertJSONL(ctx, exec, cfg, tbl, opts, ic.batchSize, r, &total)
	}
	if err != nil {
		return err
	}

	data, _ := json.Marshal(total)
	_, err = fmt.Fprintf(out, "%s\n", data)
	return err
}

// insertJSONL reads JSONL (one doc per line) and bulk-inserts in batches.
func insertJSONL(ctx context.Context, exec *query.Executor, cfg *rootConfig, tbl reql.Term, opts reql.OptArgs, batchSize int, r io.Reader, total *insertResult) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var batch []json.RawMessage
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		batch = append(batch, json.RawMessage(string(line)))
		if len(batch) >= batchSize {
			if err := execInsertBatch(ctx, exec, cfg, tbl, opts, batch, total); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading input: %w", err)
	}
	if len(batch) > 0 {
		return execInsertBatch(ctx, exec, cfg, tbl, opts, batch, total)
	}
	return nil
}

// insertJSON reads a JSON array of documents and bulk-inserts in batches.
func insertJSON(ctx context.Context, exec *query.Executor, cfg *rootConfig, tbl reql.Term, opts reql.OptArgs, batchSize int, r io.Reader, total *insertResult) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}
	var docs []json.RawMessage
	if err := json.Unmarshal(data, &docs); err != nil {
		return fmt.Errorf("parsing JSON input: %w", err)
	}
	for i := 0; i < len(docs); i += batchSize {
		end := i + batchSize
		if end > len(docs) {
			end = len(docs)
		}
		if err := execInsertBatch(ctx, exec, cfg, tbl, opts, docs[i:end], total); err != nil {
			return err
		}
	}
	return nil
}

// execInsertBatch runs a single batch insert and accumulates totals.
func execInsertBatch(ctx context.Context, exec *query.Executor, cfg *rootConfig, tbl reql.Term, opts reql.OptArgs, batch []json.RawMessage, total *insertResult) error {
	docs := make([]interface{}, len(batch))
	for i, d := range batch {
		docs[i] = d
	}
	term := tbl.Insert(docs, opts)
	_, cur, err := exec.Run(ctx, term, buildQueryOpts(cfg))
	if err != nil {
		return err
	}
	if cur == nil {
		return nil
	}
	defer func() { _ = cur.Close() }()

	rows, err := cur.All()
	if err != nil {
		return err
	}
	if len(rows) > 0 {
		var res struct {
			Inserted int64 `json:"inserted"`
			Errors   int64 `json:"errors"`
		}
		if err := json.Unmarshal(rows[0], &res); err != nil {
			return fmt.Errorf("parsing insert response: %w", err)
		}
		total.Inserted += res.Inserted
		total.Errors += res.Errors
	}
	return nil
}
