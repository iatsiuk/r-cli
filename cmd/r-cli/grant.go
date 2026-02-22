package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"r-cli/internal/reql"
)

func newGrantCmd(cfg *rootConfig) *cobra.Command {
	var (
		tableName string
		read      bool
		write     bool
	)
	c := &cobra.Command{
		Use:   "grant <user>",
		Short: "Grant or revoke permissions for a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			perms := buildGrantPerms(cmd, read, write)
			if len(perms) == 0 {
				return fmt.Errorf("at least one permission flag required (--read, --write)")
			}
			if tableName != "" && cfg.database == "" {
				return fmt.Errorf("--table requires --db")
			}
			user := args[0]
			term := grantTerm(cfg.database, tableName, user, perms)
			return execTerm(cmd.Context(), cfg, term, os.Stdout)
		},
	}
	c.Flags().StringVar(&tableName, "table", "", "target table (requires --db)")
	c.Flags().BoolVar(&read, "read", false, "read permission")
	c.Flags().BoolVar(&write, "write", false, "write permission")
	return c
}

func buildGrantPerms(cmd *cobra.Command, read, write bool) map[string]interface{} {
	perms := map[string]interface{}{}
	if cmd.Flags().Changed("read") {
		perms["read"] = read
	}
	if cmd.Flags().Changed("write") {
		perms["write"] = write
	}
	return perms
}

func grantTerm(dbName, tableName, user string, perms map[string]interface{}) reql.Term {
	if dbName == "" {
		return reql.Grant(user, perms)
	}
	db := reql.DB(dbName)
	if tableName == "" {
		return db.Grant(user, perms)
	}
	return db.Table(tableName).Grant(user, perms)
}
