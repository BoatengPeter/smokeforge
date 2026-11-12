package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sampleState() *State {
	return &State{
		SchemaVersion: 1,
		Name:          "Acme App",
		Kind:          "backend",
		CreatedAt:     "2026-11-12T12:00:00Z",
		Registry: RegistryRef{
			Source: "https://example.com/releases",
			Tag:    "v1.0.0",
			Commit: "abc123",
		},
		Variables: map[string]string{
			"module_path": "example.com/acme",
		},
		Targets: []Target{
			{
				Path:     ".",
				Stack:    "go/chi",
				Features: []string{"database:postgres"},
			},
		},
		RootFeatures: []string{"docker"},
		Applied: []AppliedOverlay{
			{
				Overlay:     "database:postgres",
				Target:      ".",
				RegistryTag: "v1.0.0",
				Files: map[string]string{
					"internal/db/db.go": "sha256:abc",
				},
				Markers: []string{"database:postgres"},
			},
		},
	}
}

func sampleLock() *Lock {
	return &Lock{
		LockVersion: 1,
		Registry: RegistryLock{
			Source:        "https://example.com/releases",
			Tag:           "v1.0.0",
			Commit:        "abc123",
			ArchiveSHA256: "abc",
		},
		Templates: map[string]string{"backend/go": "sha256:def"},
		Overlays:  map[string]string{"database/postgres": "sha256:ghi"},
	}
}

func TestWriteAndLoadState(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	state := sampleState()
	lock := sampleLock()

	if err := WriteState(dir, state, lock); err != nil {
		t.Fatalf("write state: %v", err)
	}

	loadedState, err := LoadState(dir)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}

	if loadedState.Name != state.Name || loadedState.Targets[0].Stack != "go/chi" {
		t.Fatalf("state = %+v", loadedState)
	}

	if loadedState.Applied[0].Files["internal/db/db.go"] != "sha256:abc" {
		t.Fatalf("state hashes = %+v", loadedState.Applied[0].Files)
	}

	loadedLock, err := LoadLock(dir)
	if err != nil {
		t.Fatalf("load lock: %v", err)
	}

	if loadedLock.Registry.Tag != "v1.0.0" || loadedLock.Templates["backend/go"] != "sha256:def" {
		t.Fatalf("lock = %+v", loadedLock)
	}
}

func TestWriteStateFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := WriteState(dir, sampleState(), sampleLock()); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{StateFile, LockFile} {
		path := filepath.Join(dir, name)

		//nolint:gosec // test reads a known state file under t.TempDir().
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}

		if !strings.HasSuffix(string(data), "\n") {
			t.Fatalf("%s does not end with a newline", name)
		}

		var decoded map[string]any
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("%s is not valid json: %v", name, err)
		}
	}
}

func TestLoadStateRejectsUnsupportedSchema(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, StateFile),
		[]byte(`{"schemaVersion": 2}`),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	_, err := LoadState(dir)
	if err == nil || !strings.Contains(err.Error(), "schemaVersion") {
		t.Fatalf("error = %v, want schemaVersion error", err)
	}
}

func TestWriteStateRejectsInvalidState(t *testing.T) {
	t.Parallel()

	state := sampleState()
	state.SchemaVersion = 0

	err := WriteState(t.TempDir(), state, sampleLock())
	if err == nil || !strings.Contains(err.Error(), "schemaVersion") {
		t.Fatalf("error = %v, want schemaVersion error", err)
	}
}
