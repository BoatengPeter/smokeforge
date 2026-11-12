package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, StateFile), []byte(`{"schemaVersion":1}`), 0o600); err != nil {
		t.Fatal(err)
	}

	nested := filepath.Join(root, "apps", "api", "internal")
	if err := os.MkdirAll(nested, 0o750); err != nil {
		t.Fatal(err)
	}

	found, err := FindRoot(nested)
	if err != nil {
		t.Fatalf("find root: %v", err)
	}

	if found != root {
		t.Fatalf("root = %q, want %q", found, root)
	}
}

func TestFindRootRejectsOutsideProject(t *testing.T) {
	t.Parallel()

	_, err := FindRoot(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "project") {
		t.Fatalf("error = %v, want not-a-project error", err)
	}
}
