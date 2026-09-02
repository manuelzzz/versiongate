// Package migrations embeds VersionGate's SQL schema migrations so they
// ship inside the compiled binary — no migrations/ directory needs to be
// present alongside a deployed VersionGate (e.g. inside the Docker
// image from #19), and there's no risk of running against a stale copy
// of the migration files.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
