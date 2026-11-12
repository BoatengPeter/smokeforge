package registry

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const localManifest = `schemaVersion: 1
name: local
kinds:
  backend: {max: 1}
stacks:
  - {id: "go/chi", kind: backend, name: Go, template: templates/go, variables: [module_path]}
features: []
`

func writeLocalRegistry(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yaml")
	if err := os.WriteFile(path, []byte(localManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestLoadLocalDir(t *testing.T) {
	dir := writeLocalRegistry(t)

	res, err := Load(context.Background(), Options{Source: dir})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if res.Manifest.Name != "local" {
		t.Fatalf("name = %q, want local", res.Manifest.Name)
	}
	if res.Tag != "local" {
		t.Fatalf("tag = %q, want local", res.Tag)
	}
	if res.Dir != dir {
		t.Fatalf("dir = %q, want %q", res.Dir, dir)
	}
	if res.FromCache {
		t.Fatal("local registry must not be marked as cached")
	}
}

func TestLoadLocalDirRejectsInvalidManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yaml")
	if err := os.WriteFile(path, []byte("schemaVersion: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(context.Background(), Options{Source: dir})
	if err == nil || !strings.Contains(err.Error(), "schemaVersion") {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestLoadLocalDirMissingManifest(t *testing.T) {
	_, err := Load(context.Background(), Options{Source: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "manifest") {
		t.Fatalf("expected missing manifest error, got %v", err)
	}
}
