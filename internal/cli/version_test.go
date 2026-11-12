package cli

import (
	"strings"
	"testing"

	"github.com/BoatengPeter/smokeforge/internal/version"
)

func TestVersionCommand(t *testing.T) {
	t.Parallel()

	out, _, err := executeCommand(t, "version")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if !strings.Contains(out, "smokeforge") {
		t.Fatalf("version output missing program name: %q", out)
	}

	if !strings.Contains(out, version.String()) {
		t.Fatalf("version output missing version %q: %q", version.String(), out)
	}
}
