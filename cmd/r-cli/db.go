package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"r-cli/internal/reql"
)

func newDBCmd(cfg *rootConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Database management commands",
	}
	cmd.AddCommand(
		newDBListCmd(cfg),
		newDBCreateCmd(cfg),
		newDBDropCmd(cfg),
	)
	return cmd
}

func newDBListCmd(cfg *rootConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List databases",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return execTerm(cmd.Context(), cfg, reql.DBList(), os.Stdout)
		},
	}
}

func newDBCreateCmd(cfg *rootConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Create a database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return execTerm(cmd.Context(), cfg, reql.DBCreate(args[0]), os.Stdout)
		},
	}
}

func newDBDropCmd(cfg *rootConfig) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "drop <name>",
		Short: "Drop a database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				if err := confirmDrop("database", args[0], os.Stdin, cfg.quiet); err != nil {
					return err
				}
			}
			return execTerm(cmd.Context(), cfg, reql.DBDrop(args[0]), os.Stdout)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}

var errAborted = errors.New("aborted")

// confirmDrop prompts the user to confirm a destructive drop operation.
// When quiet is true, skips the prompt and returns errAborted (use --yes to proceed in quiet mode).
func confirmDrop(kind, name string, r io.Reader, quiet bool) error {
	if quiet {
		return errAborted
	}
	fmt.Fprintf(os.Stderr, "Drop %s %q? [y/N] ", kind, name)
	scanner := bufio.NewScanner(r)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer == "y" || answer == "yes" {
			return nil
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading input: %w", err)
	}
	return errAborted
}
