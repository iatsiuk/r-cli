package main

import (
	"os"

	"github.com/spf13/cobra"

	"r-cli/internal/reql"
)

func newIndexCmd(cfg *rootConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "index",
		Short: "Index management commands",
	}
	cmd.AddCommand(
		newIndexListCmd(cfg),
		newIndexCreateCmd(cfg),
		newIndexDropCmd(cfg),
		newIndexRenameCmd(cfg),
		newIndexStatusCmd(cfg),
		newIndexWaitCmd(cfg),
	)
	return cmd
}

// indexTable returns a Table term for the configured database and given table name.
func indexTable(cfg *rootConfig, table string) (reql.Term, error) {
	db, err := tableDB(cfg)
	if err != nil {
		return reql.Term{}, err
	}
	return db.Table(table), nil
}

func newIndexListCmd(cfg *rootConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "list <table>",
		Short: "List secondary indexes on a table",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tbl, err := indexTable(cfg, args[0])
			if err != nil {
				return err
			}
			return execTerm(cmd.Context(), cfg, tbl.IndexList(), os.Stdout)
		},
	}
}

func newIndexCreateCmd(cfg *rootConfig) *cobra.Command {
	var geo, multi bool
	c := &cobra.Command{
		Use:   "create <table> <name>",
		Short: "Create a secondary index",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			tbl, err := indexTable(cfg, args[0])
			if err != nil {
				return err
			}
			opts := reql.OptArgs{}
			if geo {
				opts["geo"] = true
			}
			if multi {
				opts["multi"] = true
			}
			if len(opts) > 0 {
				return execTerm(cmd.Context(), cfg, tbl.IndexCreate(args[1], opts), os.Stdout)
			}
			return execTerm(cmd.Context(), cfg, tbl.IndexCreate(args[1]), os.Stdout)
		},
	}
	c.Flags().BoolVar(&geo, "geo", false, "create a geo index")
	c.Flags().BoolVar(&multi, "multi", false, "create a multi-value index")
	return c
}

func newIndexDropCmd(cfg *rootConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "drop <table> <name>",
		Short: "Drop a secondary index",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			tbl, err := indexTable(cfg, args[0])
			if err != nil {
				return err
			}
			return execTerm(cmd.Context(), cfg, tbl.IndexDrop(args[1]), os.Stdout)
		},
	}
}

func newIndexRenameCmd(cfg *rootConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "rename <table> <old> <new>",
		Short: "Rename a secondary index",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			tbl, err := indexTable(cfg, args[0])
			if err != nil {
				return err
			}
			return execTerm(cmd.Context(), cfg, tbl.IndexRename(args[1], args[2]), os.Stdout)
		},
	}
}

func newIndexStatusCmd(cfg *rootConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "status <table> [name]",
		Short: "Show index status",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			tbl, err := indexTable(cfg, args[0])
			if err != nil {
				return err
			}
			return execTerm(cmd.Context(), cfg, tbl.IndexStatus(args[1:]...), os.Stdout)
		},
	}
}

func newIndexWaitCmd(cfg *rootConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "wait <table> [name]",
		Short: "Wait for indexes to be ready",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			tbl, err := indexTable(cfg, args[0])
			if err != nil {
				return err
			}
			return execTerm(cmd.Context(), cfg, tbl.IndexWait(args[1:]...), os.Stdout)
		},
	}
}
