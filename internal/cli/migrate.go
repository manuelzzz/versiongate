package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/manuelzzz/versiongate/internal/postgres"
)

func newMigrateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Manage the database schema",
	}

	cmd.AddCommand(newMigrateUpCommand())
	cmd.AddCommand(newMigrateDownCommand())
	cmd.AddCommand(newMigrateStatusCommand())

	return cmd
}

func newMigrateUpCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Apply all pending migrations",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			db, err := openDB(ctx)
			if err != nil {
				return fmt.Errorf("connect: %w", err)
			}
			defer db.Close()

			if err := postgres.MigrateUp(ctx, db); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "migrations applied")
			return nil
		},
	}
}

func newMigrateDownCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Roll back the most recently applied migration",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			db, err := openDB(ctx)
			if err != nil {
				return fmt.Errorf("connect: %w", err)
			}
			defer db.Close()

			if err := postgres.MigrateDown(ctx, db); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "last migration rolled back")
			return nil
		},
	}
}

func newMigrateStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show which migrations have been applied",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			db, err := openDB(ctx)
			if err != nil {
				return fmt.Errorf("connect: %w", err)
			}
			defer db.Close()

			status, err := postgres.MigrationStatus(ctx, db)
			if err != nil {
				return err
			}
			for _, m := range status {
				state := string(m.State)
				if !m.AppliedAt.IsZero() {
					state = fmt.Sprintf("%s at %s", state, m.AppliedAt)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", m.Source.Path, state)
			}
			return nil
		},
	}
}
