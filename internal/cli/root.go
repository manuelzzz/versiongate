// Package cli implements the versiongate operator CLI. Commands here
// call the same application/domain layer the HTTP server uses — never
// storage directly — per specs/decisions/bootstrap-mechanism.md.
package cli

import (
	"github.com/spf13/cobra"

	"github.com/manuelzzz/versiongate/internal/config"
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
	// future: root.AddCommand(newBootstrapCommand()), etc.

	return root
}

// loadConfig is the one place every subcommand goes through to read
// VersionGate's environment configuration, so adding a command never
// means re-deriving config/DB wiring (issue #17's acceptance criterion).
func loadConfig() (config.Config, error) {
	return config.Load()
}
