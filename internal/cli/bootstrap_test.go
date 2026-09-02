package cli

import (
	"bytes"
	"strings"
	"testing"
)

// TestBootstrapCommand_RequiresName exercises the command's flag
// wiring without a database: --name is required, so invoking bootstrap
// without it must fail before ever attempting to connect.
func TestBootstrapCommand_RequiresName(t *testing.T) {
	root := NewRootCommand()

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"bootstrap"})

	err := root.Execute()
	if err == nil {
		t.Fatal("Execute() with no --name = nil error, want error")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Fatalf("error = %q, want it to mention the missing --name flag", err.Error())
	}
}
