package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// buildVersion identifies the versiongate CLI/server release. It is
// overridable at build time via:
//
//	-ldflags "-X github.com/manuelzzz/versiongate/internal/cli.buildVersion=1.2.3"
//
// This is VersionGate's own build version, unrelated to the
// internal/version package, which models the version of a published
// Release the service manages.
var buildVersion = "dev"

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the versiongate CLI version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), buildVersion)
			return nil
		},
	}
}
