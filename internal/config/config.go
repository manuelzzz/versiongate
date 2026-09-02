package config

import (
	"errors"
	"os"
)

var ErrMissingDatabaseDsn = errors.New("config: VERSIONGATE_DATABASE_DSN is required")

const defaultListenAddr = ":8888"

type Config struct {
	ListenAddr  string
	DatabaseDSN string
}

func Load() (Config, error) {
	dsn := os.Getenv("VERSIONGATE_DATABASE_DSN")
	if dsn == "" {
		return Config{}, ErrMissingDatabaseDsn
	}

	addr := os.Getenv("VERSIONGATE_LISTEN_ADDR")
	if addr == "" {
		addr = defaultListenAddr
	}

	return Config{ListenAddr: addr, DatabaseDSN: dsn}, nil
}
