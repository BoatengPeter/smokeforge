package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/BoatengPeter/smokeforge/internal/version"
)

func TestVersionCommand(t *testing.T) {
	oldVersion, oldCommit, oldDate := version.Version, version.Commit, version.Date
	version.Version = "test-version"
	version.Commit = "test-commit"
	version.Date = "2026-11-12"
	t.Cleanup(func() {
		version.Version = oldVersion
		version.Commit = oldCommit
		version.Date = oldDate
	})

	var out, errOut bytes.Buffer
	cmd := NewRootCmd(&out, &errOut)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "smokeforge") {
		t.Fatalf("version output missing program name: %q", got)
	}
	if !strings.Contains(got, "test-version") {
		t.Fatalf("version output missing version: %q", got)
	}
}
