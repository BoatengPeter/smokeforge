package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidManifest(t *testing.T) {
	m, err := LoadFile(filepath.Join("testdata", "manifest.yaml"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if m.SchemaVersion != 1 {
		t.Fatalf("schemaVersion = %d, want 1", m.SchemaVersion)
	}
	if len(m.Stacks) != 2 || m.Stacks[0].ID != "go/chi" {
		t.Fatalf("unexpected stacks: %+v", m.Stacks)
	}
	if len(m.Features) != 1 || m.Features[0].ID != "database" {
		t.Fatalf("unexpected features: %+v", m.Features)
	}
}

func TestLoadFileRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.yaml")
	data := []byte("schemaVersion: 1\nname: x\nunknown: true\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFile(path); err == nil {
		t.Fatal("expected unknown-field error")
	}
}

func TestParseRejectsMultipleDocuments(t *testing.T) {
	data := []byte("schemaVersion: 1\nname: a\n---\nschemaVersion: 1\nname: b\n")
	if _, err := Parse(data); err == nil {
		t.Fatal("expected error for multiple YAML documents")
	}
}
