package config

import "testing"

func TestLoad(t *testing.T) {
	t.Run("missing DSN is an error", func(t *testing.T) {
		t.Setenv("VERSIONGATE_DATABASE_DSN", "")
		_, err := Load()
		if err == nil {
			t.Fatal("Load() = nil error, want error for missing DSN")
		}
	})

	t.Run("applies default listen address", func(t *testing.T) {
		t.Setenv("VERSIONGATE_DATABASE_DSN", "postgres://localhost/test")
		t.Setenv("VERSIONGATE_LISTEN_ADDR", "")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() returned unexpected error: %v", err)
		}
		if cfg.ListenAddr != defaultListenAddr {
			t.Fatalf("ListenAddr = %q, want %q", cfg.ListenAddr, defaultListenAddr)
		}
	})

	t.Run("respects explicit listen address", func(t *testing.T) {
		t.Setenv("VERSIONGATE_DATABASE_DSN", "postgres://localhost/test")
		t.Setenv("VERSIONGATE_LISTEN_ADDR", ":9090")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() returned unexpected error: %v", err)
		}
		if cfg.ListenAddr != ":9090" {
			t.Fatalf("ListenAddr = %q, want :9090", cfg.ListenAddr)
		}
	})
}
