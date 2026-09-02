package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/manuelzzz/versiongate/internal/postgres"
	"github.com/manuelzzz/versiongate/internal/project"
	"github.com/manuelzzz/versiongate/internal/token"
)

// newBootstrapCommand resolves VersionGate's bootstrap chicken-and-egg
// problem (specs/decisions/bootstrap-mechanism.md): creating a Project
// and its first API Token is itself a write operation, but every write
// operation requires a token to already exist. This command is the one
// operator-invoked entry point that creates both, going through the
// same domain layer (internal/project, internal/token) the HTTP API
// will — it never writes to storage directly.
func newBootstrapCommand() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Create the initial Project and its first API Token",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			db, err := openDB(ctx)
			if err != nil {
				return fmt.Errorf("connect: %w", err)
			}
			defer db.Close()

			projects := postgres.NewProjectRepository(db)
			tokens := postgres.NewTokenRepository(db)

			p, err := project.Create(ctx, projects, name)
			if err != nil {
				return fmt.Errorf("create project: %w", err)
			}

			_, raw, err := token.Issue(ctx, tokens, p.ID)
			if err != nil {
				return fmt.Errorf("issue token: %w", err)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Project created: %s (%s)\n\n", p.Name, p.ID)
			fmt.Fprintln(out, "API Token (save this now — it will not be shown again):")
			fmt.Fprintln(out, raw)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "name for the initial Project (required)")
	// Only errors if "name" isn't a registered flag, which would be a
	// bug in this function itself, not a runtime condition — safe to
	// ignore.
	_ = cmd.MarkFlagRequired("name")

	return cmd
}
