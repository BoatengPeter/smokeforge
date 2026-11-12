package cli

import (
	"strings"
	"testing"
)

func TestRegistry(t *testing.T) {
	t.Parallel()

	t.Run("info", func(t *testing.T) {
		t.Parallel()

		out, _, err := executeCommand(t, "registry", "info", "--registry", writeTestRegistry(t))
		if err != nil {
			t.Fatalf("execute: %v", err)
		}

		wants := []string{
			"source:",
			"tag: local",
			"schema: 1",
			"stacks: 1",
			"features: 1",
		}
		for _, want := range wants {
			if !strings.Contains(out, want) {
				t.Fatalf("output %q missing %q", out, want)
			}
		}
	})

	t.Run("update local", func(t *testing.T) {
		t.Parallel()

		out, _, err := executeCommand(t, "registry", "update", "--registry", writeTestRegistry(t))
		if err != nil {
			t.Fatalf("execute: %v", err)
		}

		if !strings.Contains(out, "updated registry") {
			t.Fatalf("output %q missing update confirmation", out)
		}
	})

	t.Run("rejects unknown subcommand", func(t *testing.T) {
		t.Parallel()

		_, _, err := executeCommand(t, "registry", "nope")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
