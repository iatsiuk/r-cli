package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
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
}

func newRootCmd() *cobra.Command {
	cfg := &rootConfig{}
	return buildRootCmd(cfg)
}

func buildRootCmd(cfg *rootConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "r-cli",
		Short:             "RethinkDB query CLI",
		Version:           version,
		SilenceUsage:      true,
		SilenceErrors:     true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return cfg.resolvePassword()
		},
	}
	cmd.SetHelpCommand(&cobra.Command{Hidden: true})

	f := cmd.PersistentFlags()
	f.StringVarP(&cfg.host, "host", "H", "localhost", "RethinkDB host")
	f.IntVarP(&cfg.port, "port", "P", 28015, "RethinkDB port")
	f.StringVarP(&cfg.database, "db", "d", "", "default database")
	f.StringVarP(&cfg.user, "user", "u", "admin", "RethinkDB user")
	f.StringVarP(&cfg.password, "password", "p", "", "RethinkDB password (or RETHINKDB_PASSWORD env)")
	f.StringVar(&cfg.passwordFile, "password-file", "", "read password from file")
	f.DurationVarP(&cfg.timeout, "timeout", "t", 30*time.Second, "connection timeout")
	f.StringVarP(&cfg.format, "format", "f", "json", "output format: json, jsonl, raw, table")

	return cmd
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
	c.password = strings.TrimRight(string(data), "\r\n")
	return nil
}
