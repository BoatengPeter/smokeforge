package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHashBytes(t *testing.T) {
	t.Parallel()

	got := HashBytes([]byte("hello"))

	want := "sha256:2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got != want {
		t.Fatalf("hash = %q, want %q", got, want)
	}
}

func TestHashDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("world"), 0o600); err != nil {
		t.Fatal(err)
	}

	first, err := HashDir(dir)
	if err != nil {
		t.Fatalf("hash dir: %v", err)
	}

	second, err := HashDir(dir)
	if err != nil {
		t.Fatalf("hash dir again: %v", err)
	}

	if first != second {
		t.Fatalf("hash dir is not deterministic: %q != %q", first, second)
	}

	if !strings.HasPrefix(first, "sha256:") {
		t.Fatalf("hash = %q, want sha256 prefix", first)
	}
}
