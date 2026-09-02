package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	root := NewRootCommand()

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() returned unexpected error: %v", err)
	}

	if got := strings.TrimSpace(out.String()); got != buildVersion {
		t.Fatalf("output = %q, want %q", got, buildVersion)
	}
}

func TestRootHelp(t *testing.T) {
	root := NewRootCommand()

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() returned unexpected error: %v", err)
	}

	if !strings.Contains(out.String(), "versiongate") {
		t.Fatalf("--help output = %q, want it to mention versiongate", out.String())
	}
}
