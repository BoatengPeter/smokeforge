package patch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BoatengPeter/smokeforge/internal/render"
)

const validOverlay = `id: logging/basic
name: Basic logging
files: files
ops:
  - addFiles:
      - path: logger.go.tmpl
        dest: internal/log/logger.go
  - insertAfter:
      file: main.go
      anchor: imports
      marker: logging:basic
      content: '"example.com/app/internal/log"'
  - env:
      key: LOG_LEVEL
      example: info
`

func writeOverlayFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "overlay.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestLoadOverlayFile(t *testing.T) {
	t.Parallel()

	overlay, err := LoadOverlayFile(writeOverlayFile(t, validOverlay))
	if err != nil {
		t.Fatalf("load overlay: %v", err)
	}

	if overlay.ID != "logging/basic" {
		t.Fatalf("id = %q, want logging/basic", overlay.ID)
	}

	if len(overlay.Ops) != 3 {
		t.Fatalf("ops = %d, want 3", len(overlay.Ops))
	}
}

func TestLoadOverlayFileRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	path := writeOverlayFile(t, "id: x\nunknown: true\n")
	if _, err := LoadOverlayFile(path); err == nil {
		t.Fatal("expected unknown-field error")
	}
}

func TestOverlay_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Overlay)
		want   string
	}{
		{
			name:   "empty id",
			mutate: func(o *Overlay) { o.ID = "" },
			want:   "id",
		},
		{
			name:   "no ops",
			mutate: func(o *Overlay) { o.Ops = nil },
			want:   "ops",
		},
		{
			name: "empty op",
			mutate: func(o *Overlay) {
				o.Ops = append(o.Ops, Op{})
			},
			want: "exactly one",
		},
		{
			name: "two ops in one item",
			mutate: func(o *Overlay) {
				o.Ops[0].Env = &Env{Key: "X"}
			},
			want: "exactly one",
		},
		{
			name: "unsafe destination",
			mutate: func(o *Overlay) {
				o.Ops[0].AddFiles = &AddFiles{{Path: "x", Dest: "../escape"}}
			},
			want: "safe relative",
		},
		{
			name: "missing dependency module",
			mutate: func(o *Overlay) {
				o.Ops = append(o.Ops, Op{AddDependency: &Dependency{
					File:    "go.mod",
					Version: "v1.0.0",
				}})
			},
			want: "module",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			overlay, err := LoadOverlayFile(writeOverlayFile(t, validOverlay))
			if err != nil {
				t.Fatal(err)
			}

			tt.mutate(overlay)

			err = overlay.Validate()
			if err == nil {
				t.Fatal("expected error")
			}

			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error %q does not contain %q", err, tt.want)
			}
		})
	}
}

func TestApply(t *testing.T) {
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

	logger := "package log\n\n// Level is {{.project_slug}} log level.\nconst Level = \"info\"\n"
	if err := os.WriteFile(
		filepath.Join(overlayDir, "files", "logger.go.tmpl"),
		[]byte(logger),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	overlay, err := LoadOverlayFile(writeOverlayFile(t, validOverlay))
	if err != nil {
		t.Fatal(err)
	}

	applied, err := Apply(t.Context(), overlay, ApplyOptions{
		Root:       root,
		OverlayDir: overlayDir,
		Renderer:   render.New("My App", nil),
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	if len(applied.Env) != 1 || applied.Env[0].Key != "LOG_LEVEL" {
		t.Fatalf("env = %+v, want LOG_LEVEL", applied.Env)
	}

	mainContent := mustReadFile(t, filepath.Join(root, "main.go"))

	if !strings.Contains(string(mainContent), "smokeforge:begin logging:basic") {
		t.Fatalf("main.go missing begin marker:\n%s", mainContent)
	}

	if !strings.Contains(string(mainContent), `"example.com/app/internal/log"`) {
		t.Fatalf("main.go missing inserted import:\n%s", mainContent)
	}

	loggerContent := mustReadFile(t, filepath.Join(root, "internal", "log", "logger.go"))

	if !strings.Contains(string(loggerContent), "my-app log level") {
		t.Fatalf("logger.go was not rendered:\n%s", loggerContent)
	}
}

func TestApplyRejectsUnsafeDestinationBeforeWriting(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	overlayDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(overlayDir, "files"), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(overlayDir, "files", "x.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	overlay := &Overlay{
		ID: "unsafe",
		Ops: []Op{
			{AddFiles: &AddFiles{{Path: "x.txt", Dest: "../escape.txt"}}},
		},
	}

	_, err := Apply(t.Context(), overlay, ApplyOptions{
		Root:       root,
		OverlayDir: overlayDir,
		Renderer:   render.New("App", nil),
	})
	if err == nil {
		t.Fatal("expected unsafe path error")
	}

	if _, statErr := os.Stat(filepath.Join(filepath.Dir(root), "escape.txt")); statErr == nil {
		t.Fatal("unsafe file was written outside the root")
	}
}
