package main

import (
	"os"

	"github.com/manuelzzz/versiongate/internal/cli"
)

func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
