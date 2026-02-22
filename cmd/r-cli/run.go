package main

import (
	"bytes"
	"context"
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
func newExecutor(cfg *rootConfig) (*query.Executor, func(), error) {
	tlsCfg, err := cfg.buildTLSConfig()
	if err != nil {
		return nil, func() {}, err
	}
	mgr := connmgr.NewFromConfig(conn.Config{
		Host:     cfg.host,
		Port:     cfg.port,
		User:     cfg.user,
		Password: cfg.password,
	}, tlsCfg)
	return query.New(mgr), func() { _ = mgr.Close() }, nil
}

// execTerm builds a connection, runs the given ReQL term, and writes output.
func execTerm(ctx context.Context, cfg *rootConfig, term reql.Term, w io.Writer) error {
	if cfg.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.timeout)
		defer cancel()
	}

	if cfg.verbose && !cfg.quiet {
		_, _ = fmt.Fprintf(os.Stderr, "connecting to %s:%d\n", cfg.host, cfg.port)
	}

	exec, cleanup, err := newExecutor(cfg)
	if err != nil {
		return err
	}
	defer cleanup()

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

// buildQueryOpts constructs the ReQL query options from the root config.
func buildQueryOpts(cfg *rootConfig) reql.OptArgs {
	opts := reql.OptArgs{}
	if cfg.database != "" {
		opts["db"] = cfg.database
	}
	if cfg.profile {
		opts["profile"] = true
	}
	return opts
}

// writeQueryMeta writes verbose timing and profile data to stderr.
func writeQueryMeta(cfg *rootConfig, profile json.RawMessage, elapsed time.Duration) {
	if cfg.verbose && !cfg.quiet {
		_, _ = fmt.Fprintf(os.Stderr, "query time: %v\n", elapsed)
	}
	if cfg.profile && !cfg.quiet && len(profile) > 0 {
		_, _ = fmt.Fprintf(os.Stderr, "profile: %s\n", profile)
	}
}

// makeIter wraps cur in a convertingIter when pseudo-type conversion is requested.
func makeIter(cur output.RowIterator, cfg *rootConfig) output.RowIterator {
	if cfg.timeFormat == "native" || cfg.binaryFormat == "native" {
		return &convertingIter{
			inner:         cur,
			convertTime:   cfg.timeFormat == "native",
			convertBinary: cfg.binaryFormat == "native",
		}
	}
	return cur
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
