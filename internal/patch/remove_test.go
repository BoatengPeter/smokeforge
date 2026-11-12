package patch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BoatengPeter/smokeforge/internal/render"
)

func TestRemoveBlock(t *testing.T) {
	t.Parallel()

	content := "package main\n\n// smokeforge:begin logging\n\"os\"\n// smokeforge:end logging\n"

	got, err := removeBlock(content, "logging")
	if err != nil {
		t.Fatalf("remove block: %v", err)
	}

	if strings.Contains(got, "smokeforge") || strings.Contains(got, `"os"`) {
		t.Fatalf("block was not removed:\n%s", got)
	}

	if got != "package main\n\n" {
		t.Fatalf("content = %q", got)
	}
}

func TestRemoveBlockMissingIsNoop(t *testing.T) {
	t.Parallel()

	content := "package main\n"

	got, err := removeBlock(content, "logging")
	if err != nil {
		t.Fatalf("remove block: %v", err)
	}

	if got != content {
		t.Fatalf("content changed: %q", got)
	}
}

func TestRemoveBlockMalformed(t *testing.T) {
	t.Parallel()

	_, err := removeBlock("// smokeforge:begin logging\n", "logging")
	if err == nil || !strings.Contains(err.Error(), "malformed") {
		t.Fatalf("error = %v, want malformed", err)
	}
}

func TestCheckRemovable(t *testing.T) {
	t.Parallel()

	t.Run("supported ops", func(t *testing.T) {
		t.Parallel()

		overlay, err := LoadOverlayFile(writeOverlayFile(t, validOverlay))
		if err != nil {
			t.Fatal(err)
		}

		if err := CheckRemovable(overlay); err != nil {
			t.Fatalf("check removable: %v", err)
		}
	})

	t.Run("structured patch", func(t *testing.T) {
		t.Parallel()

		overlay := &Overlay{
			ID: "patch",
			Ops: []Op{
				{PatchJSON: &JSONPatch{File: "package.json", Op: "set", Path: "x", Value: 1}},
			},
		}
		if err := CheckRemovable(overlay); err == nil || !strings.Contains(err.Error(), "non-reversible") {
			t.Fatalf("error = %v, want non-reversible", err)
		}
	})

	t.Run("dependency", func(t *testing.T) {
		t.Parallel()

		overlay := &Overlay{
			ID: "dep",
			Ops: []Op{
				{AddDependency: &Dependency{File: "go.mod", Module: "x", Version: "v1"}},
			},
		}
		if err := CheckRemovable(overlay); err == nil || !strings.Contains(err.Error(), "non-reversible") {
			t.Fatalf("error = %v, want non-reversible", err)
		}
	})
}

func TestApplyRemoveRoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	overlayDir := t.TempDir()

	base := "package main\n\nimport (\n\t\"fmt\"\n\t// smokeforge:anchor imports\n)\n"
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte(base), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(overlayDir, "files"), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(overlayDir, "files", "logger.go.tmpl"),
		[]byte("package log\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	overlay, err := LoadOverlayFile(writeOverlayFile(t, validOverlay))
	if err != nil {
		t.Fatal(err)
	}

	renderer := render.New("My App", nil)
	options := ApplyOptions{Root: root, OverlayDir: overlayDir, Renderer: renderer}

	if _, err := Apply(t.Context(), overlay, options); err != nil {
		t.Fatalf("apply: %v", err)
	}

	if err := Remove(t.Context(), overlay, RemoveOptions(options)); err != nil {
		t.Fatalf("remove: %v", err)
	}

	mainContent := mustReadFile(t, filepath.Join(root, "main.go"))
	if string(mainContent) != base {
		t.Fatalf("main.go was not restored:\nwant:\n%s\ngot:\n%s", base, mainContent)
	}

	if _, err := os.Stat(filepath.Join(root, "internal", "log", "logger.go")); !os.IsNotExist(err) {
		t.Fatalf("overlay file still exists: %v", err)
	}

	if err := Remove(t.Context(), overlay, RemoveOptions(options)); err != nil {
		t.Fatalf("second remove should be idempotent: %v", err)
	}
}
