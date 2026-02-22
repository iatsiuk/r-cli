package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"os"

	"github.com/spf13/cobra"

	"r-cli/internal/conn"
	"r-cli/internal/connmgr"
	"r-cli/internal/query"
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
	mgr := connmgr.NewFromConfig(conn.Config{
		Host:     cfg.host,
		Port:     cfg.port,
		User:     cfg.user,
		Password: cfg.password,
	}, (*tls.Config)(nil))
	defer func() { _ = mgr.Close() }()

	exec := query.New(mgr)
	info, err := exec.ServerInfo(ctx)
	if err != nil {
		return err
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
