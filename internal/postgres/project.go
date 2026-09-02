package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/manuelzzz/versiongate/internal/project"
)

// ProjectRepository is a PostgreSQL-backed implementation of
// project.Repository.
type ProjectRepository struct {
	db *sql.DB
}

// NewProjectRepository builds a ProjectRepository backed by db.
func NewProjectRepository(db *sql.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

func (r *ProjectRepository) Create(ctx context.Context, name string) (project.Project, error) {
	const query = `
		INSERT INTO projects (name)
		VALUES ($1)
		RETURNING id, name, active, created_at, updated_at`

	var p project.Project
	err := r.db.QueryRowContext(ctx, query, name).
		Scan(&p.ID, &p.Name, &p.Active, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return project.Project{}, fmt.Errorf("postgres: create project: %w", err)
	}
	return p, nil
}

func (r *ProjectRepository) Get(ctx context.Context, id project.ID) (project.Project, error) {
	const query = `
		SELECT id, name, active, created_at, updated_at
		FROM projects
		WHERE id = $1`

	var p project.Project
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(&p.ID, &p.Name, &p.Active, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return project.Project{}, project.ErrNotFound
	}
	if err != nil {
		return project.Project{}, fmt.Errorf("postgres: get project: %w", err)
	}
	return p, nil
}

func (r *ProjectRepository) Deactivate(ctx context.Context, id project.ID) (project.Project, error) {
	const query = `
		UPDATE projects
		SET active = FALSE, updated_at = now()
		WHERE id = $1
		RETURNING id, name, active, created_at, updated_at`

	var p project.Project
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(&p.ID, &p.Name, &p.Active, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return project.Project{}, project.ErrNotFound
	}
	if err != nil {
		return project.Project{}, fmt.Errorf("postgres: deactivate project: %w", err)
	}
	return p, nil
}
