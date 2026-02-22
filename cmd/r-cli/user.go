package main

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"r-cli/internal/reql"
)

func newUserCmd(cfg *rootConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "User management commands",
	}
	cmd.AddCommand(
		newUserListCmd(cfg),
		newUserCreateCmd(cfg),
		newUserDeleteCmd(cfg),
		newUserSetPasswordCmd(cfg),
	)
	return cmd
}

func newUserListCmd(cfg *rootConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List users",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return execTerm(cmd.Context(), cfg, reql.DB("rethinkdb").Table("users"), os.Stdout)
		},
	}
}

func newUserCreateCmd(cfg *rootConfig) *cobra.Command {
	var password string
	c := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pwd := password
			if !cmd.Flags().Changed("password") {
				var err error
				pwd, err = promptPassword(os.Stderr, os.Stdin)
				if err != nil {
					return err
				}
			}
			doc := map[string]interface{}{"id": args[0], "password": pwd}
			return execTerm(cmd.Context(), cfg, reql.DB("rethinkdb").Table("users").Insert(doc), os.Stdout)
		},
	}
	c.Flags().StringVar(&password, "password", "", "user password")
	return c
}

func newUserDeleteCmd(cfg *rootConfig) *cobra.Command {
	var yes bool
	c := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				if err := confirmDrop("user", args[0], os.Stdin, cfg.quiet); err != nil {
					return err
				}
			}
			return execTerm(cmd.Context(), cfg, reql.DB("rethinkdb").Table("users").Get(args[0]).Delete(), os.Stdout)
		},
	}
	c.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return c
}

func newUserSetPasswordCmd(cfg *rootConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "set-password <name>",
		Short: "Update user password",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pwd, err := promptPassword(os.Stderr, os.Stdin)
			if err != nil {
				return err
			}
			return execTerm(cmd.Context(), cfg,
				reql.DB("rethinkdb").Table("users").Get(args[0]).Update(map[string]interface{}{"password": pwd}),
				os.Stdout)
		},
	}
}

// promptPassword reads a password without echo when stdin is a terminal.
// Falls back to plain line reading for non-TTY input (tests, piped input).
func promptPassword(w io.Writer, r io.Reader) (string, error) {
	_, _ = fmt.Fprint(w, "Password: ")
	if f, ok := r.(*os.File); ok && term.IsTerminal(int(f.Fd())) { //nolint:gosec
		pwd, err := term.ReadPassword(int(f.Fd())) //nolint:gosec
		_, _ = fmt.Fprintln(w)
		if err != nil {
			return "", fmt.Errorf("reading password: %w", err)
		}
		return string(pwd), nil
	}
	// non-TTY: read one line
	scanner := bufio.NewScanner(r)
	if scanner.Scan() {
		if text := scanner.Text(); text != "" {
			return text, nil
		}
		return "", fmt.Errorf("password cannot be empty")
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading password: %w", err)
	}
	return "", fmt.Errorf("password cannot be empty")
}
