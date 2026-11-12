package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackupRestore(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeDriftFile(t, root, "keep.go", "original keep")
	writeDriftFile(t, root, "delete.go", "original delete")

	backup, err := CreateBackup(root, []string{"keep.go", "delete.go", "created.go"})
	if err != nil {
		t.Fatalf("backup: %v", err)
	}

	writeDriftFile(t, root, "keep.go", "changed")

	if err := os.Remove(filepath.Join(root, "delete.go")); err != nil {
		t.Fatal(err)
	}

	writeDriftFile(t, root, "created.go", "new")

	if err := backup.Restore(); err != nil {
		t.Fatalf("restore: %v", err)
	}

	keep := mustReadProjectFile(t, filepath.Join(root, "keep.go"))
	if string(keep) != "original keep" {
		t.Fatalf("keep.go = %q", keep)
	}

	deleted := mustReadProjectFile(t, filepath.Join(root, "delete.go"))
	if string(deleted) != "original delete" {
		t.Fatalf("delete.go = %q", deleted)
	}

	if _, err := os.Stat(filepath.Join(root, "created.go")); !os.IsNotExist(err) {
		t.Fatalf("created.go should have been removed: %v", err)
	}
}

func TestBackupDiscard(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeDriftFile(t, root, "main.go", "package main\n")

	backup, err := CreateBackup(root, []string{"main.go"})
	if err != nil {
		t.Fatalf("backup: %v", err)
	}

	dir := backup.Dir()
	if err := backup.Discard(); err != nil {
		t.Fatalf("discard: %v", err)
	}

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("backup directory still exists: %v", err)
	}
}

func TestCreateBackupRejectsUnsafePath(t *testing.T) {
	t.Parallel()

	_, err := CreateBackup(t.TempDir(), []string{"../escape"})
	if err == nil || !strings.Contains(err.Error(), "safe relative") {
		t.Fatalf("error = %v, want unsafe path error", err)
	}
}

func mustReadProjectFile(t *testing.T, path string) []byte {
	t.Helper()

	//nolint:gosec // test reads a known file under t.TempDir().
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return data
}
