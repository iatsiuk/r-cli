package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"r-cli/internal/conn"
	"r-cli/internal/response"
)

// exit codes
const (
	exitOK         = 0
	exitConnection = 1
	exitQuery      = 2
	exitAuth       = 3
	exitINT        = 130
)

type rootConfig struct {
	host         string
	port         int
	database     string
	user         string
	password     string
	passwordFile string
	timeout      time.Duration
	format       string
	profile      bool
	timeFormat   string
	binaryFormat string
	quiet        bool
	verbose      bool
}

func newRootCmd() *cobra.Command {
	cfg := &rootConfig{}
	return buildRootCmd(cfg)
}

func buildRootCmd(cfg *rootConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "r-cli",
		Short:         "RethinkDB query CLI",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg.resolveEnvVars(cmd.Flags().Changed)
			return cfg.resolvePassword()
		},
	}
	cmd.SetHelpCommand(&cobra.Command{Hidden: true})
	cmd.AddCommand(newRunCmd(cfg))
	cmd.AddCommand(newDBCmd(cfg))
	cmd.AddCommand(newTableCmd(cfg))
	cmd.AddCommand(newStatusCmd(cfg))

	f := cmd.PersistentFlags()
	f.StringVarP(&cfg.host, "host", "H", "localhost", "RethinkDB host")
	f.IntVarP(&cfg.port, "port", "P", 28015, "RethinkDB port")
	f.StringVarP(&cfg.database, "db", "d", "", "default database")
	f.StringVarP(&cfg.user, "user", "u", "admin", "RethinkDB user")
	f.StringVarP(&cfg.password, "password", "p", "", "RethinkDB password (or RETHINKDB_PASSWORD env)")
	f.StringVar(&cfg.passwordFile, "password-file", "", "read password from file")
	f.DurationVarP(&cfg.timeout, "timeout", "t", 30*time.Second, "connection timeout")
	f.StringVarP(&cfg.format, "format", "f", "", "output format: json, jsonl, raw, table (default: json on TTY, jsonl when piped)")
	f.BoolVar(&cfg.profile, "profile", false, "enable query profiling output")
	f.StringVar(&cfg.timeFormat, "time-format", "native", "time format: native (convert pseudo-types), raw (pass-through)")
	f.StringVar(&cfg.binaryFormat, "binary-format", "native", "binary format: native (convert pseudo-types), raw (pass-through)")
	f.BoolVar(&cfg.quiet, "quiet", false, "suppress non-data output to stderr")
	f.BoolVar(&cfg.verbose, "verbose", false, "show connection info and query timing to stderr")

	return cmd
}

// exitCode maps an error to the appropriate process exit code.
func exitCode(err error) int {
	if err == nil {
		return exitOK
	}
	if errors.Is(err, conn.ErrReqlAuth) {
		return exitAuth
	}
	if isQueryError(err) {
		return exitQuery
	}
	return exitConnection
}

func isQueryError(err error) bool {
	var c *response.ReqlCompileError
	var r *response.ReqlRuntimeError
	var cl *response.ReqlClientError
	var ne *response.ReqlNonExistenceError
	var pe *response.ReqlPermissionError
	return errors.As(err, &c) || errors.As(err, &r) || errors.As(err, &cl) ||
		errors.As(err, &ne) || errors.As(err, &pe)
}

// resolveEnvVars applies env var values for flags not explicitly set via CLI.
func (c *rootConfig) resolveEnvVars(changed func(string) bool) {
	applyEnvStr(&c.host, changed("host"), "RETHINKDB_HOST")
	applyEnvStr(&c.user, changed("user"), "RETHINKDB_USER")
	applyEnvStr(&c.password, changed("password"), "RETHINKDB_PASSWORD")
	applyEnvStr(&c.database, changed("db"), "RETHINKDB_DATABASE")
	if !changed("port") {
		if v := os.Getenv("RETHINKDB_PORT"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				c.port = n
			}
		}
	}
}

// applyEnvStr sets *dst to the env var value when the flag was not explicitly set.
func applyEnvStr(dst *string, flagChanged bool, key string) {
	if flagChanged {
		return
	}
	if v := os.Getenv(key); v != "" {
		*dst = v
	}
}

// resolvePassword loads the password from --password-file if set.
func (c *rootConfig) resolvePassword() error {
	if c.passwordFile == "" {
		return nil
	}
	data, err := os.ReadFile(c.passwordFile)
	if err != nil {
		return fmt.Errorf("reading password file: %w", err)
	}
	c.password = strings.TrimSpace(string(data))
	return nil
}
