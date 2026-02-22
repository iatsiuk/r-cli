package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"r-cli/internal/conn"
	"r-cli/internal/connmgr"
	"r-cli/internal/output"
	"r-cli/internal/query"
	"r-cli/internal/reql"
)

func newRunCmd(cfg *rootConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "run [term]",
		Short: "Execute a raw ReQL JSON term",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			termBytes, err := readTerm(args, os.Stdin)
			if err != nil {
				return err
			}
			return runQuery(cmd.Context(), cfg, termBytes, os.Stdout)
		},
	}
}

// readTerm reads the ReQL JSON term from args (first element) or stdin.
// Returns an error if the JSON is invalid.
func readTerm(args []string, stdin io.Reader) ([]byte, error) {
	if len(args) == 1 {
		b := []byte(args[0])
		if !json.Valid(b) {
			return nil, fmt.Errorf("run: invalid JSON term")
		}
		return b, nil
	}
	data, err := io.ReadAll(stdin)
	if err != nil {
		return nil, fmt.Errorf("run: reading stdin: %w", err)
	}
	data = bytes.TrimSpace(data)
	if !json.Valid(data) {
		return nil, fmt.Errorf("run: invalid JSON term")
	}
	return data, nil
}

func runQuery(ctx context.Context, cfg *rootConfig, termJSON []byte, w io.Writer) error {
	mgr := connmgr.NewFromConfig(conn.Config{
		Host:     cfg.host,
		Port:     cfg.port,
		User:     cfg.user,
		Password: cfg.password,
	}, (*tls.Config)(nil))
	defer func() { _ = mgr.Close() }()

	exec := query.New(mgr)
	opts := reql.OptArgs{}
	if cfg.database != "" {
		opts["db"] = cfg.database
	}

	cur, err := exec.Run(ctx, reql.Datum(json.RawMessage(termJSON)), opts)
	if err != nil {
		return err
	}
	if cur == nil {
		return nil
	}
	defer func() { _ = cur.Close() }()

	return writeOutput(w, cfg.format, cur)
}

func writeOutput(w io.Writer, format string, iter output.RowIterator) error {
	switch format {
	case "jsonl":
		return output.JSONL(w, iter)
	case "raw":
		return output.Raw(w, iter)
	case "table":
		return output.Table(w, iter)
	default:
		return output.JSON(w, iter)
	}
}
