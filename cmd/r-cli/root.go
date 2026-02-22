package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

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
	host               string
	port               int
	database           string
	user               string
	password           string
	passwordFile       string
	timeout            time.Duration
	format             string
	profile            bool
	timeFormat         string
	binaryFormat       string
	quiet              bool
	verbose            bool
	tlsCACert          string
	tlsClientCert      string
	tlsKey             string
	insecureSkipVerify bool
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
		Args:          cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return fmt.Errorf("accepts at most 1 arg(s), received %d", len(args))
			}
			if len(args) == 0 && term.IsTerminal(int(os.Stdin.Fd())) { //nolint:gosec
				_ = cmd.Help()
				return nil
			}
			expr, err := readQueryExpr(args, cmd.InOrStdin())
			if err != nil {
				return err
			}
			return runQueryExpr(cmd, cfg, expr)
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// completion subcommands don't need connection config
			if p := cmd.Parent(); p != nil && p.Name() == "completion" {
				return nil
			}
			if err := cfg.resolveEnvVars(cmd.Flags().Changed); err != nil {
				return err
			}
			// -p/--password flag takes precedence over --password-file
			if cmd.Flags().Changed("password") {
				return nil
			}
			return cfg.resolvePassword()
		},
	}
	cmd.SetHelpCommand(&cobra.Command{Hidden: true})
	cmd.AddCommand(newQueryCmd(cfg))
	cmd.AddCommand(newRunCmd(cfg))
	cmd.AddCommand(newDBCmd(cfg))
	cmd.AddCommand(newTableCmd(cfg))
	cmd.AddCommand(newIndexCmd(cfg))
	cmd.AddCommand(newUserCmd(cfg))
	cmd.AddCommand(newGrantCmd(cfg))
	cmd.AddCommand(newInsertCmd(cfg))
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
	f.StringVar(&cfg.tlsCACert, "tls-cert", "", "path to CA certificate PEM file")
	f.StringVar(&cfg.tlsClientCert, "tls-client-cert", "", "path to client certificate PEM file")
	f.StringVar(&cfg.tlsKey, "tls-key", "", "path to client private key PEM file")
	f.BoolVar(&cfg.insecureSkipVerify, "insecure-skip-verify", false, "skip TLS certificate verification (insecure)")

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
	var qe *queryError
	var c *response.ReqlCompileError
	var r *response.ReqlRuntimeError
	var cl *response.ReqlClientError
	var ne *response.ReqlNonExistenceError
	var pe *response.ReqlPermissionError
	return errors.As(err, &qe) || errors.As(err, &c) || errors.As(err, &r) || errors.As(err, &cl) ||
		errors.As(err, &ne) || errors.As(err, &pe)
}

// resolveEnvVars applies env var values for flags not explicitly set via CLI.
func (c *rootConfig) resolveEnvVars(changed func(string) bool) error {
	applyEnvStr(&c.host, changed("host"), "RETHINKDB_HOST")
	applyEnvStr(&c.user, changed("user"), "RETHINKDB_USER")
	applyEnvStr(&c.password, changed("password"), "RETHINKDB_PASSWORD")
	applyEnvStr(&c.database, changed("db"), "RETHINKDB_DATABASE")
	if !changed("port") {
		if v := os.Getenv("RETHINKDB_PORT"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("RETHINKDB_PORT %q: not a valid port number", v)
			}
			c.port = n
		}
	}
	return nil
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

// buildTLSConfig returns a *tls.Config built from TLS flags, or nil for plain TCP.
func (c *rootConfig) buildTLSConfig() (*tls.Config, error) {
	if c.tlsCACert == "" && c.tlsClientCert == "" && c.tlsKey == "" && !c.insecureSkipVerify {
		return nil, nil
	}
	tlsCfg := &tls.Config{
		InsecureSkipVerify: c.insecureSkipVerify, //nolint:gosec
	}
	if c.tlsCACert != "" {
		pool, err := loadCACert(c.tlsCACert)
		if err != nil {
			return nil, err
		}
		tlsCfg.RootCAs = pool
	}
	if c.tlsClientCert != "" || c.tlsKey != "" {
		cert, err := loadClientCert(c.tlsClientCert, c.tlsKey)
		if err != nil {
			return nil, err
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}
	return tlsCfg, nil
}

// loadCACert reads a PEM file and returns an x509.CertPool with the certificate.
func loadCACert(path string) (*x509.CertPool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading CA cert: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(data) {
		return nil, fmt.Errorf("parsing CA cert: no valid PEM certificate found")
	}
	return pool, nil
}

// loadClientCert loads an x509 key pair for mutual TLS authentication.
func loadClientCert(certPath, keyPath string) (tls.Certificate, error) {
	if certPath == "" || keyPath == "" {
		return tls.Certificate{}, fmt.Errorf("--tls-client-cert and --tls-key must be used together")
	}
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("loading client certificate: %w", err)
	}
	return cert, nil
}
