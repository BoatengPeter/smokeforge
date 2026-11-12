package cli

import (
	"strings"
	"testing"
)

func TestDoctor(t *testing.T) {
	t.Parallel()

	out, _, err := executeCommand(
		t,
		"doctor",
		"--registry", writeTestRegistry(t),
		"--cache-dir", t.TempDir(),
	)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	for _, want := range []string{"registry: ok", "git: ok", "cache: ok"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output %q missing %q", out, want)
		}
	}
}
