// Package cli implements the versiongate operator CLI. Commands here
// call the same application/domain layer the HTTP server uses — never
// storage directly — per specs/decisions/bootstrap-mechanism.md.
package cli

import (
	"context"
	"database/sql"

	"github.com/spf13/cobra"

	"github.com/manuelzzz/versiongate/internal/config"
	"github.com/manuelzzz/versiongate/internal/postgres"
)

// NewRootCommand builds the versiongate root command with all
// subcommands registered. cmd/versiongate/main.go's only job is to call
// this and Execute() it.
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "versiongate",
		Short:         "Operate a VersionGate instance",
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	root.AddCommand(newVersionCommand())
	root.AddCommand(newMigrateCommand())
	// future: root.AddCommand(newBootstrapCommand()), etc.

	return root
}

// loadConfig is the one place every subcommand goes through to read
// VersionGate's environment configuration, so adding a command never
// means re-deriving config/DB wiring (issue #17's acceptance criterion).
func loadConfig() (config.Config, error) {
	return config.Load()
}

// openDB is the one place every subcommand goes through to connect to
// Postgres, for the same reason loadConfig exists: a new command should
// never need to re-derive connection/pool setup.
func openDB(ctx context.Context) (*sql.DB, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	return postgres.Open(ctx, cfg.DatabaseDSN)
}
