package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"r-cli/internal/conn"
	"r-cli/internal/connmgr"
	"r-cli/internal/output"
	"r-cli/internal/query"
	"r-cli/internal/reql"
	"r-cli/internal/response"
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
			return execTerm(cmd.Context(), cfg, reql.Datum(json.RawMessage(termBytes)), os.Stdout)
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

// newExecutor creates a connection manager and query executor from the given config.
// The returned cleanup func must be called to close the manager.
func newExecutor(cfg *rootConfig) (exec *query.Executor, cleanup func()) {
	mgr := connmgr.NewFromConfig(conn.Config{
		Host:     cfg.host,
		Port:     cfg.port,
		User:     cfg.user,
		Password: cfg.password,
	}, (*tls.Config)(nil))
	return query.New(mgr), func() { _ = mgr.Close() }
}

// execTerm builds a connection, runs the given ReQL term, and writes output.
func execTerm(ctx context.Context, cfg *rootConfig, term reql.Term, w io.Writer) error {
	if cfg.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.timeout)
		defer cancel()
	}

	if cfg.verbose {
		fmt.Fprintf(os.Stderr, "connecting to %s:%d\n", cfg.host, cfg.port)
	}

	exec, cleanup := newExecutor(cfg)
	defer cleanup()

	opts := reql.OptArgs{}
	if cfg.database != "" {
		opts["db"] = cfg.database
	}
	if cfg.profile {
		opts["profile"] = true
	}

	start := time.Now()
	cur, err := exec.Run(ctx, term, opts)
	if err != nil {
		return err
	}
	if cfg.verbose {
		fmt.Fprintf(os.Stderr, "query time: %v\n", time.Since(start))
	}
	if cur == nil {
		return nil
	}
	defer func() { _ = cur.Close() }()

	var iter output.RowIterator = cur
	if cfg.timeFormat == "native" || cfg.binaryFormat == "native" {
		iter = &convertingIter{
			inner:         cur,
			convertTime:   cfg.timeFormat == "native",
			convertBinary: cfg.binaryFormat == "native",
		}
	}
	return writeOutput(w, output.DetectFormat(os.Stdout, cfg.format), iter)
}

// convertingIter wraps a RowIterator, applying selective pseudo-type conversion to each row.
type convertingIter struct {
	inner         output.RowIterator
	convertTime   bool
	convertBinary bool
}

func (c *convertingIter) Next() (json.RawMessage, error) {
	raw, err := c.inner.Next()
	if err != nil {
		return nil, err
	}
	return convertRow(raw, c.convertTime, c.convertBinary), nil
}

// convertRow applies selective pseudo-type conversion to raw JSON.
// Returns raw unchanged on any error or when no conversion is needed.
func convertRow(raw json.RawMessage, convertTime, convertBinary bool) json.RawMessage {
	if !convertTime && !convertBinary {
		return raw
	}
	var v interface{}
	if json.Unmarshal(raw, &v) != nil {
		return raw
	}
	out, err := json.Marshal(selectiveConvert(v, convertTime, convertBinary))
	if err != nil {
		return raw
	}
	return out
}

// selectiveConvert recursively converts TIME and/or BINARY pseudo-types based on flags.
func selectiveConvert(v interface{}, convertTime, convertBinary bool) interface{} {
	if convertTime && convertBinary {
		return response.ConvertPseudoTypes(v)
	}
	switch val := v.(type) {
	case map[string]interface{}:
		return convertMap(val, convertTime, convertBinary)
	case []interface{}:
		out := make([]interface{}, len(val))
		for i, item := range val {
			out[i] = selectiveConvert(item, convertTime, convertBinary)
		}
		return out
	}
	return v
}

// convertMap handles pseudo-type detection and selective conversion for map values.
func convertMap(m map[string]interface{}, convertTime, convertBinary bool) interface{} {
	reqlType, isReql := m["$reql_type$"].(string)
	if isReql {
		switch reqlType {
		case "TIME":
			if convertTime {
				return response.ConvertPseudoTypes(m)
			}
			return m
		case "BINARY":
			if convertBinary {
				return response.ConvertPseudoTypes(m)
			}
			return m
		}
	}
	out := make(map[string]interface{}, len(m))
	for k, item := range m {
		out[k] = selectiveConvert(item, convertTime, convertBinary)
	}
	return out
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
