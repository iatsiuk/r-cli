package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func newStatusCmd(cfg *rootConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show server info and connection status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runStatus(cmd.Context(), cfg, os.Stdout)
		},
	}
}

// statusInfo is the JSON output of the status command.
type statusInfo struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	ServerID string `json:"server_id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
}

func runStatus(ctx context.Context, cfg *rootConfig, w io.Writer) error {
	if cfg.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.timeout)
		defer cancel()
	}

	if cfg.verbose && !cfg.quiet {
		_, _ = fmt.Fprintf(os.Stderr, "connecting to %s:%d\n", cfg.host, cfg.port)
	}

	exec, cleanup := newExecutor(cfg)
	defer cleanup()

	start := time.Now()
	info, err := exec.ServerInfo(ctx)
	if err != nil {
		return err
	}
	if cfg.verbose && !cfg.quiet {
		_, _ = fmt.Fprintf(os.Stderr, "query time: %v\n", time.Since(start))
	}

	si := statusInfo{
		Host:     cfg.host,
		Port:     cfg.port,
		ServerID: info.ID,
		Name:     info.Name,
		Status:   "ok",
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(si)
}
