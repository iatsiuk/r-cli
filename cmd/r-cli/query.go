package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"r-cli/internal/reql/parser"
)

// queryError wraps errors that should map to exitQuery (2) exit code.
type queryError struct{ err error }

func (e *queryError) Error() string { return e.err.Error() }
func (e *queryError) Unwrap() error { return e.err }

func newQueryCmd(cfg *rootConfig) *cobra.Command {
	var filePath string
	var stopOnError bool

	cmd := &cobra.Command{
		Use:   "query [expression]",
		Short: "Execute a ReQL expression",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if filePath != "" && len(args) > 0 {
				return fmt.Errorf("query: --file and expression argument are mutually exclusive")
			}
			if filePath != "" {
				return runQueryFile(cmd, cfg, filePath, stopOnError)
			}
			expr, err := readQueryExpr(args, cmd.InOrStdin())
			if err != nil {
				return err
			}
			return runQueryExpr(cmd, cfg, expr)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&filePath, "file", "F", "", "read query from file (use --- to separate multiple queries)")
	f.BoolVar(&stopOnError, "stop-on-error", false, "stop on first error when executing multiple queries")
	return cmd
}

// readQueryExpr returns the expression from args[0] or by reading stdin.
func readQueryExpr(args []string, stdin io.Reader) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	data, err := io.ReadAll(stdin)
	if err != nil {
		return "", fmt.Errorf("query: reading stdin: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// runQueryExpr parses expr and executes it, writing results to cmd's output.
func runQueryExpr(cmd *cobra.Command, cfg *rootConfig, expr string) error {
	term, err := parser.Parse(expr)
	if err != nil {
		return &queryError{err: fmt.Errorf("query: %w", err)}
	}
	return execTerm(cmd.Context(), cfg, term, cmd.OutOrStdout())
}

// runQueryFile reads queries from path, splits on "---", and executes each.
func runQueryFile(cmd *cobra.Command, cfg *rootConfig, path string, stopOnError bool) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer func() { _ = f.Close() }()

	queries, err := splitQueries(f)
	if err != nil {
		return fmt.Errorf("query: reading file: %w", err)
	}

	var firstErr error
	for _, q := range queries {
		if err := runQueryExpr(cmd, cfg, q); err != nil {
			if stopOnError {
				return err
			}
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "query error: %v\n", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	if firstErr != nil {
		// individual errors already printed to stderr; return summary to signal non-zero exit
		return &queryError{err: fmt.Errorf("query: one or more queries failed")}
	}
	return nil
}

// splitQueries reads r and splits on lines containing only "---".
func splitQueries(r io.Reader) ([]string, error) {
	var queries []string
	var cur strings.Builder
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "---" {
			if q := strings.TrimSpace(cur.String()); q != "" {
				queries = append(queries, q)
			}
			cur.Reset()
		} else {
			cur.WriteString(line)
			cur.WriteByte('\n')
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if q := strings.TrimSpace(cur.String()); q != "" {
		queries = append(queries, q)
	}
	return queries, nil
}
