package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeDriftFile(t *testing.T, root, name, content string) {
	t.Helper()

	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestCheckDrift(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeDriftFile(t, root, "clean.go", "clean")
	writeDriftFile(t, root, "edited.go", "edited")

	state := &State{
		Applied: []AppliedOverlay{
			{
				Overlay: "logging:basic",
				Files: map[string]string{
					"clean.go":   HashBytes([]byte("clean")),
					"edited.go":  HashBytes([]byte("original")),
					"missing.go": HashBytes([]byte("gone")),
				},
			},
		},
	}

	drift, err := CheckDrift(root, state)
	if err != nil {
		t.Fatalf("check drift: %v", err)
	}

	if len(drift) != 2 {
		t.Fatalf("drift = %+v, want 2 entries", drift)
	}

	if drift[0].Path != "edited.go" || drift[1].Path != "missing.go" {
		t.Fatalf("drift order = %+v", drift)
	}

	if drift[1].Actual != "missing" {
		t.Fatalf("missing actual = %q", drift[1].Actual)
	}
}

func TestCheckDriftIgnoresCleanProject(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeDriftFile(t, root, "main.go", "package main\n")

	state := &State{
		Applied: []AppliedOverlay{
			{Overlay: "logging", Files: map[string]string{"main.go": HashBytes([]byte("package main\n"))}},
		},
	}

	drift, err := CheckDrift(root, state)
	if err != nil {
		t.Fatalf("check drift: %v", err)
	}

	if len(drift) != 0 {
		t.Fatalf("drift = %+v, want none", drift)
	}
}

func TestRehash(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeDriftFile(t, root, "main.go", "package main\n")

	state := &State{
		Applied: []AppliedOverlay{
			{
				Overlay: "logging",
				Files: map[string]string{
					"main.go":    "sha256:stale",
					"removed.go": "sha256:gone",
				},
			},
		},
	}

	if err := Rehash(root, state); err != nil {
		t.Fatalf("rehash: %v", err)
	}

	want := HashBytes([]byte("package main\n"))
	if state.Applied[0].Files["main.go"] != want {
		t.Fatalf("hash = %q, want %q", state.Applied[0].Files["main.go"], want)
	}

	if _, exists := state.Applied[0].Files["removed.go"]; exists {
		t.Fatal("missing file still present in hash map")
	}
}

func TestCheckDriftRejectsUnsafePath(t *testing.T) {
	t.Parallel()

	state := &State{
		Applied: []AppliedOverlay{
			{Overlay: "bad", Files: map[string]string{"../escape": "sha256:x"}},
		},
	}

	_, err := CheckDrift(t.TempDir(), state)
	if err == nil || !strings.Contains(err.Error(), "safe relative") {
		t.Fatalf("error = %v, want unsafe path error", err)
	}
}
