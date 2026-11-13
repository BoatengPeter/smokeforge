package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BoatengPeter/smokeforge/internal/manifest"
	"github.com/BoatengPeter/smokeforge/internal/resolve"
)

const newManifestYAML = `schemaVersion: 1
name: test-registry
kinds:
  backend: {max: 1}
stacks:
  - id: test/plain
    kind: backend
    name: Test plain
    template: templates/plain
    variables: [module_path, greeting]
presets:
  - id: fullstack
    name: Fullstack
    default: true
    features: [metrics]
  - id: logging-only
    name: Logging only
    features: [logging]
  - id: frontend
    name: Frontend
    features: []
features:
  - id: logging
    name: Logging
    kinds: [backend]
    variants:
      - id: basic
        name: Basic
        default: true
        overlays:
          test/plain: overlays/logging/basic
  - id: metrics
    name: Metrics
    kinds: [backend]
    overlays:
      test/plain: overlays/metrics/basic
`

const newTemplateYAML = `id: test/plain
variables:
  - name: module_path
    prompt: Module path
    default: example.com/test
  - name: greeting
    prompt: Greeting
    default: hello
`

const newMainTemplate = `package main

import (
	"fmt"
	// smokeforge:anchor imports
)

func main() {
	fmt.Println({{quote .greeting}})
}
`

const newMetricsOverlayYAML = `id: metrics/basic
files: files
ops:
  - addFiles:
      - path: metrics.go.tmpl
        dest: internal/metrics/metrics.go
  - env:
      key: METRICS_ENABLED
      example: "1"
`

const newOverlayYAML = `id: logging/basic
files: files
ops:
  - addFiles:
      - path: logger.go.tmpl
        dest: internal/log/logger.go
  - insertAfter:
      file: main.go
      anchor: imports
      marker: logging:basic
      content: '"os"'
  - env:
      key: LOG_LEVEL
      example: info
`

func writeGenerationRegistry(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "manifest.yaml"), newManifestYAML)
	writeTestFile(t, filepath.Join(dir, "templates", "plain", "template.yaml"), newTemplateYAML)
	writeTestFile(t, filepath.Join(dir, "templates", "plain", "files", "go.mod.tmpl"),
		"module {{.module_path}}\n\ngo 1.26\n")
	writeTestFile(t, filepath.Join(dir, "templates", "plain", "files", "main.go.tmpl"), newMainTemplate)
	writeTestFile(t, filepath.Join(dir, "overlays", "logging", "basic", "overlay.yaml"), newOverlayYAML)
	writeTestFile(t, filepath.Join(dir, "overlays", "logging", "basic", "files", "logger.go.tmpl"),
		"package log\n\n// Level is the level for {{.project_slug}}.\nconst Level = \"info\"\n")
	writeTestFile(t, filepath.Join(dir, "overlays", "metrics", "basic", "overlay.yaml"), newMetricsOverlayYAML)
	writeTestFile(t, filepath.Join(dir, "overlays", "metrics", "basic", "files", "metrics.go.tmpl"),
		"package metrics\n\n// Enabled reports metrics for {{.project_slug}}.\nconst Enabled = true\n")

	return dir
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	registryDir := writeGenerationRegistry(t)
	target := filepath.Join(t.TempDir(), "acme")

	out, _, err := executeCommand(
		t,
		"new", target,
		"--registry", registryDir,
		"--kind", "backend",
		"--stack", "test/plain",
		"--name", "Acme App",
		"--feature", "logging",
		"--var", "module_path=example.com/acme",
		"--no-input",
		"--no-git",
	)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	for _, want := range []string{"Acme App", "Project created", target} {
		if !strings.Contains(out, want) {
			t.Fatalf("output %q missing %q", out, want)
		}
	}

	mainContent := mustReadFile(t, filepath.Join(target, "main.go"))

	if !strings.Contains(string(mainContent), "smokeforge:begin logging:basic") {
		t.Fatalf("overlay markers missing:\n%s", mainContent)
	}

	if _, err := os.Stat(filepath.Join(target, ".smokeforge.json")); err != nil {
		t.Fatalf("project state missing: %v", err)
	}

	if _, err := os.Stat(filepath.Join(target, ".git")); err == nil {
		t.Fatal("git repository created despite --no-git")
	}
}

func TestNextStepLines(t *testing.T) {
	t.Parallel()

	plan := &resolve.Plan{
		Target: resolve.TargetPlan{
			Stack: manifest.Stack{ID: "next", Name: "Next.js"},
			Variables: map[string]string{
				"package_manager": "bun",
			},
			Overlays: []resolve.OverlayPlan{
				{FeatureID: "database", VariantID: "postgres"},
			},
		},
	}

	lines := strings.Join(nextStepLines(plan, "my-app", true), "\n")
	for _, want := range []string{
		"cd my-app",
		"make install",
		"bun install",
		"make db-up",
		"make db-migrate",
		"make dev",
	} {
		if !strings.Contains(lines, want) {
			t.Fatalf("next steps missing %q:\n%s", want, lines)
		}
	}
}

func TestNextStepLinesWithoutDocker(t *testing.T) {
	t.Parallel()

	plan := &resolve.Plan{
		Target: resolve.TargetPlan{
			Stack: manifest.Stack{ID: "next", Name: "Next.js"},
			Variables: map[string]string{
				"package_manager": "pnpm",
			},
			Overlays: []resolve.OverlayPlan{
				{FeatureID: "database", VariantID: "postgres"},
			},
		},
	}

	lines := strings.Join(nextStepLines(plan, "my-app", false), "\n")
	for _, want := range []string{
		"Docker is not installed",
		"https://docs.docker.com/get-docker/",
		"DATABASE_URL",
		"make db-migrate",
	} {
		if !strings.Contains(lines, want) {
			t.Fatalf("next steps missing %q:\n%s", want, lines)
		}
	}

	if !strings.Contains(lines, "Install it, then run: make db-up") {
		t.Fatalf("next steps missing Docker install instruction:\n%s", lines)
	}
}

func TestNewUsesDefaultPreset(t *testing.T) {
	t.Parallel()

	registryDir := writeGenerationRegistry(t)
	target := filepath.Join(t.TempDir(), "default-preset")

	if _, _, err := executeCommand(
		t,
		"new", target,
		"--registry", registryDir,
		"--kind", "backend",
		"--stack", "test/plain",
		"--name", "Default Preset",
		"--no-input",
		"--no-git",
	); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if _, err := os.Stat(filepath.Join(target, "internal", "metrics", "metrics.go")); err != nil {
		t.Fatalf("default preset feature missing: %v", err)
	}
}

func TestNewPresetFlag(t *testing.T) {
	t.Parallel()

	registryDir := writeGenerationRegistry(t)
	target := filepath.Join(t.TempDir(), "preset-flag")

	if _, _, err := executeCommand(
		t,
		"new", target,
		"--registry", registryDir,
		"--kind", "backend",
		"--stack", "test/plain",
		"--name", "Logging Preset",
		"--preset", "logging-only",
		"--no-input",
		"--no-git",
	); err != nil {
		t.Fatalf("execute: %v", err)
	}

	mainContent := mustReadFile(t, filepath.Join(target, "main.go"))
	if !strings.Contains(string(mainContent), "smokeforge:begin logging:basic") {
		t.Fatalf("preset feature missing:\n%s", mainContent)
	}
}

func TestNewDryRun(t *testing.T) {
	t.Parallel()

	registryDir := writeGenerationRegistry(t)
	target := filepath.Join(t.TempDir(), "dry-run")

	out, _, err := executeCommand(
		t,
		"new", target,
		"--registry", registryDir,
		"--kind", "backend",
		"--stack", "test/plain",
		"--name", "Dry Run",
		"--no-input",
		"--no-git",
		"--dry-run",
	)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if !strings.Contains(out, "Dry run") {
		t.Fatalf("output %q missing dry-run summary", out)
	}

	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("dry run created target: %v", err)
	}
}

func TestNewRequiresSelection(t *testing.T) {
	t.Parallel()

	registryDir := writeGenerationRegistry(t)

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "missing kind",
			args: []string{"new", filepath.Join(t.TempDir(), "a")},
			want: "kind",
		},
		{
			name: "missing stack",
			args: []string{"new", filepath.Join(t.TempDir(), "b"), "--kind", "backend"},
			want: "stack",
		},
		{
			name: "missing name",
			args: []string{
				"new", filepath.Join(t.TempDir(), "c"),
				"--kind", "backend", "--stack", "test/plain",
			},
			want: "name",
		},
		{
			name: "unknown variable",
			args: []string{
				"new", filepath.Join(t.TempDir(), "d"),
				"--kind", "backend", "--stack", "test/plain", "--name", "App",
				"--var", "typo=x",
			},
			want: "variable",
		},
		{
			name: "invalid variable flag",
			args: []string{
				"new", filepath.Join(t.TempDir(), "e"),
				"--kind", "backend", "--stack", "test/plain", "--name", "App",
				"--var", "novalue",
			},
			want: "key=value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			args := append([]string{}, tt.args...)
			args = append(args, "--registry", registryDir, "--no-input", "--no-git")

			_, _, err := executeCommand(t, args...)
			if err == nil {
				t.Fatal("expected error")
			}

			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error %q does not contain %q", err, tt.want)
			}
		})
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()

	//nolint:gosec // test reads a known generated file under a temporary directory.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func TestWithDefaultPresetUsesKind(t *testing.T) {
	t.Parallel()

	m := &manifest.Manifest{
		Presets: []manifest.Preset{
			{ID: "web", Default: true, Kinds: []string{"frontend"}},
			{ID: "api", Default: true, Kinds: []string{"backend"}},
		},
	}

	selection := withDefaultPreset(resolve.Selection{Kind: "backend"}, m)
	if selection.Preset != "api" {
		t.Fatalf("preset = %q, want api", selection.Preset)
	}
}
