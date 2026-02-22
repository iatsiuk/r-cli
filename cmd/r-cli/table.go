package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"r-cli/internal/reql"
)

func newTableCmd(cfg *rootConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "table",
		Short: "Table management commands",
	}
	cmd.AddCommand(
		newTableListCmd(cfg),
		newTableCreateCmd(cfg),
		newTableDropCmd(cfg),
		newTableInfoCmd(cfg),
		newTableReconfigureCmd(cfg),
		newTableRebalanceCmd(cfg),
		newTableWaitCmd(cfg),
		newTableSyncCmd(cfg),
	)
	return cmd
}

// tableDB returns a DB term for the configured database, or an error if unset.
func tableDB(cfg *rootConfig) (reql.Term, error) {
	if cfg.database == "" {
		return reql.Term{}, fmt.Errorf("table commands require --db flag or RETHINKDB_DATABASE env var")
	}
	return reql.DB(cfg.database), nil
}

func newTableListCmd(cfg *rootConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List tables in current database",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			db, err := tableDB(cfg)
			if err != nil {
				return err
			}
			return execTerm(cmd.Context(), cfg, db.TableList(), os.Stdout)
		},
	}
}

func newTableCreateCmd(cfg *rootConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Create a table",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := tableDB(cfg)
			if err != nil {
				return err
			}
			return execTerm(cmd.Context(), cfg, db.TableCreate(args[0]), os.Stdout)
		},
	}
}

func newTableDropCmd(cfg *rootConfig) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "drop <name>",
		Short: "Drop a table",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := tableDB(cfg)
			if err != nil {
				return err
			}
			if !yes {
				if err := confirmDrop("table", args[0], os.Stdin, cfg.quiet); err != nil {
					return err
				}
			}
			return execTerm(cmd.Context(), cfg, db.TableDrop(args[0]), os.Stdout)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}

func newTableInfoCmd(cfg *rootConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "info <name>",
		Short: "Show table status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := tableDB(cfg)
			if err != nil {
				return err
			}
			return execTerm(cmd.Context(), cfg, db.Table(args[0]).Status(), os.Stdout)
		},
	}
}

func newTableReconfigureCmd(cfg *rootConfig) *cobra.Command {
	var shards, replicas int
	var dryRun bool
	c := &cobra.Command{
		Use:   "reconfigure <name>",
		Short: "Reconfigure table shards and replicas",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if shards == 0 && replicas == 0 {
				return fmt.Errorf("reconfigure requires --shards and/or --replicas")
			}
			db, err := tableDB(cfg)
			if err != nil {
				return err
			}
			opts := reql.OptArgs{}
			if shards > 0 {
				opts["shards"] = shards
			}
			if replicas > 0 {
				opts["replicas"] = replicas
			}
			if dryRun {
				opts["dry_run"] = true
			}
			return execTerm(cmd.Context(), cfg, db.Table(args[0]).Reconfigure(opts), os.Stdout)
		},
	}
	c.Flags().IntVar(&shards, "shards", 0, "number of shards")
	c.Flags().IntVar(&replicas, "replicas", 0, "number of replicas")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "preview reconfiguration without applying")
	return c
}

func newTableRebalanceCmd(cfg *rootConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "rebalance <name>",
		Short: "Rebalance table shards",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := tableDB(cfg)
			if err != nil {
				return err
			}
			return execTerm(cmd.Context(), cfg, db.Table(args[0]).Rebalance(), os.Stdout)
		},
	}
}

func newTableWaitCmd(cfg *rootConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "wait <name>",
		Short: "Wait for table to be ready",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := tableDB(cfg)
			if err != nil {
				return err
			}
			return execTerm(cmd.Context(), cfg, db.Table(args[0]).Wait(), os.Stdout)
		},
	}
}

func newTableSyncCmd(cfg *rootConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "sync <name>",
		Short: "Sync table to disk",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := tableDB(cfg)
			if err != nil {
				return err
			}
			return execTerm(cmd.Context(), cfg, db.Table(args[0]).Sync(), os.Stdout)
		},
	}
}
