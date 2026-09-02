// Package project implements VersionGate's Project domain concept: the
// top-level organizational boundary that Applications, Releases, and
// API Tokens are scoped to (specs/domain/project.md). This package has
// no dependency on any specific storage technology — persistence is
// expressed as the Repository interface below, per
// .rules/architecture.md's dependency-direction rule.
package project

import (
	"context"
	"errors"
	"strings"
	"time"
)

// ID identifies a Project. Projects are assigned an ID by their
// Repository at creation time (e.g. a database-generated UUID) —
// nothing in this package constructs one directly.
type ID string

// Project is the top-level organizational boundary everything else in
// the domain is scoped to.
type Project struct {
	ID        ID
	Name      string
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ErrNotFound is returned by a Repository when no Project matches the
// requested ID.
var ErrNotFound = errors.New("project: not found")

// ErrNameRequired is returned by Create when name is empty — a Project
// cannot exist without a name assigned at creation
// (specs/domain/project.md's Invariants).
var ErrNameRequired = errors.New("project: name is required")

// Repository persists and retrieves Projects. Infrastructure provides
// the implementation; this package only declares what it needs from it
// (.rules/architecture.md's Domain independence).
type Repository interface {
	Create(ctx context.Context, name string) (Project, error)
	Get(ctx context.Context, id ID) (Project, error)
	Deactivate(ctx context.Context, id ID) (Project, error)
}

// Create validates name and persists a new, active Project via repo.
// This is the one place the "a Project cannot exist without a name"
// invariant is enforced, so every caller (CLI, future HTTP handlers)
// gets it for free rather than re-checking it themselves.
func Create(ctx context.Context, repo Repository, name string) (Project, error) {
	if strings.TrimSpace(name) == "" {
		return Project{}, ErrNameRequired
	}
	return repo.Create(ctx, name)
}

// Deactivate stops a Project (and everything it owns) from serving
// policy evaluations, without deleting its data
// (specs/domain/project.md's Lifecycle considerations). Deactivating an
// already-inactive Project is not an error — it is idempotent.
func Deactivate(ctx context.Context, repo Repository, id ID) (Project, error) {
	return repo.Deactivate(ctx, id)
}
