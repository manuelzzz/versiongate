package postgres

import (
	"context"
	"testing"
)

// TestOpen_ConnectionFailure exercises Open's fail-fast behavior without
// requiring a live Postgres instance: a pre-cancelled context makes the
// ping fail immediately, proving Open surfaces connection failures as a
// wrapped error rather than succeeding or hanging. Behavior against a
// real, unreachable Postgres instance is exercised manually (see #20);
// domain logic has no dependency on this package at all.
func TestOpen_ConnectionFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Open(ctx, "postgres://user:pass@localhost:5432/db?sslmode=disable")
	if err == nil {
		t.Fatal("Open() with a cancelled context = nil error, want error")
	}
}
