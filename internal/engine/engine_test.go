package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BoatengPeter/smokeforge/internal/project"
	"github.com/BoatengPeter/smokeforge/internal/registry"
	"github.com/BoatengPeter/smokeforge/internal/resolve"
)

const manifestYAML = `schemaVersion: 1
name: test-registry
kinds:
  backend: {max: 1}
stacks:
  - id: test/plain
    kind: backend
    name: Test plain
    template: templates/plain
    variables: [module_path, greeting]
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
  - id: config
    name: Config
    kinds: [backend]
    variants:
      - id: basic
        name: Basic
        default: true
        overlays:
          test/plain: overlays/config/basic
`

const templateYAML = `id: test/plain
variables:
  - name: module_path
    prompt: Module path
    default: example.com/test
  - name: greeting
    prompt: Greeting
    default: hello
`

const goModTemplate = `module {{.module_path}}

go 1.26
`

const mainTemplate = `package main

import (
	"fmt"
	// smokeforge:anchor imports
)

func main() {
	fmt.Println({{quote .greeting}})
	// smokeforge:anchor after-print
}
`

const loggerTemplate = `package log

// Level is the log level for {{.project_slug}}.
const Level = "info"
`

const metricsOverlayYAML = `id: metrics/basic
name: Basic metrics
files: files
ops:
  - addFiles:
      - path: metrics.go.tmpl
        dest: internal/metrics/metrics.go
  - env:
      key: METRICS_ENABLED
      example: "1"
`

const configOverlayYAML = `id: config/basic
name: Basic config
files: files
ops:
  - addFiles:
      - path: config.yaml
        dest: config.yaml
  - patchYAML:
      file: config.yaml
      op: set
      path: logging.level
      value: info
`

const overlayYAML = `id: logging/basic
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
      content: '"os"'
  - insertAfter:
      file: main.go
      anchor: after-print
      marker: logging:basic-use
      content: '_ = os.Args'
  - env:
      key: LOG_LEVEL
      example: info
`

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func fixture(t *testing.T) (registry.Result, string) {
	t.Helper()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "manifest.yaml"), manifestYAML)
	writeFile(t, filepath.Join(dir, "templates", "plain", "template.yaml"), templateYAML)
	writeFile(t, filepath.Join(dir, "templates", "plain", "files", "go.mod.tmpl"), goModTemplate)
	writeFile(t, filepath.Join(dir, "templates", "plain", "files", "main.go.tmpl"), mainTemplate)
	writeFile(
		t,
		filepath.Join(dir, "overlays", "logging", "basic", "overlay.yaml"),
		overlayYAML,
	)
	writeFile(
		t,
		filepath.Join(dir, "overlays", "logging", "basic", "files", "logger.go.tmpl"),
		loggerTemplate,
	)
	writeFile(
		t,
		filepath.Join(dir, "overlays", "metrics", "basic", "overlay.yaml"),
		metricsOverlayYAML,
	)
	writeFile(
		t,
		filepath.Join(dir, "overlays", "metrics", "basic", "files", "metrics.go.tmpl"),
		"package metrics\n\n// Enabled reports metrics for {{.project_slug}}.\nconst Enabled = true\n",
	)
	writeFile(
		t,
		filepath.Join(dir, "overlays", "config", "basic", "overlay.yaml"),
		configOverlayYAML,
	)
	writeFile(
		t,
		filepath.Join(dir, "overlays", "config", "basic", "files", "config.yaml"),
		"{}\n",
	)

	reg, err := registry.Load(t.Context(), registry.Options{Source: dir})
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}

	return *reg, dir
}

func fixturePlan(t *testing.T, reg registry.Result) *resolve.Plan {
	t.Helper()

	plan, err := resolve.Resolve(reg.Manifest, reg.Dir, resolve.Selection{
		Name:      "Acme App",
		Kind:      "backend",
		Stack:     "test/plain",
		Features:  []string{"logging"},
		Variables: map[string]string{"module_path": "example.com/acme"},
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	return plan
}

func generatedProject(t *testing.T) (string, *Result, []string) {
	t.Helper()

	reg, _ := fixture(t)
	plan := fixturePlan(t, reg)
	target := filepath.Join(t.TempDir(), "acme")

	stages := make([]string, 0, 8)

	result, err := Generate(t.Context(), Request{
		Registry: &reg,
		Plan:     plan,
		Target:   target,
		NoGit:    true,
		Progress: func(stage string) {
			stages = append(stages, stage)
		},
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	return target, result, stages
}

func TestGenerateRendersBaseAndOverlays(t *testing.T) {
	t.Parallel()

	target, result, stages := generatedProject(t)

	if result.Target != target {
		t.Fatalf("target = %q, want %q", result.Target, target)
	}

	if len(stages) == 0 {
		t.Fatal("expected progress reporting")
	}

	mainContent := mustReadFile(t, filepath.Join(target, "main.go"))
	if !strings.Contains(string(mainContent), "smokeforge:begin logging:basic") {
		t.Fatalf("overlay markers missing:\n%s", mainContent)
	}

	goMod := mustReadFile(t, filepath.Join(target, "go.mod"))
	if !strings.Contains(string(goMod), "module example.com/acme") {
		t.Fatalf("go.mod not rendered:\n%s", goMod)
	}

	logger := mustReadFile(t, filepath.Join(target, "internal", "log", "logger.go"))
	if !strings.Contains(string(logger), "acme") {
		t.Fatalf("logger not rendered:\n%s", logger)
	}

	if _, err := os.Stat(filepath.Join(target, ".git")); err == nil {
		t.Fatal("git repository was initialized despite NoGit")
	}
}

func TestGenerateEnvironment(t *testing.T) {
	t.Parallel()

	target, _, _ := generatedProject(t)

	envFile := mustReadFile(t, filepath.Join(target, ".env.example"))
	if !strings.Contains(string(envFile), "LOG_LEVEL=info") {
		t.Fatalf(".env.example missing LOG_LEVEL:\n%s", envFile)
	}

	gitignore := mustReadFile(t, filepath.Join(target, ".gitignore"))
	for _, want := range []string{".env", ".smokeforge/"} {
		if !strings.Contains(string(gitignore), want) {
			t.Fatalf(".gitignore missing %q:\n%s", want, gitignore)
		}
	}
}

func TestGenerateState(t *testing.T) {
	t.Parallel()

	target, _, _ := generatedProject(t)

	state, err := project.LoadState(target)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}

	if state.Name != "Acme App" || state.Targets[0].Stack != "test/plain" {
		t.Fatalf("state = %+v", state)
	}

	if len(state.Applied) != 1 || state.Applied[0].Overlay != "logging:basic" {
		t.Fatalf("applied = %+v", state.Applied)
	}

	if state.Applied[0].Files["internal/log/logger.go"] == "" {
		t.Fatalf("overlay file hash missing: %+v", state.Applied[0].Files)
	}

	lock, err := project.LoadLock(target)
	if err != nil {
		t.Fatalf("load lock: %v", err)
	}

	if lock.Registry.Tag != "local" {
		t.Fatalf("lock tag = %q", lock.Registry.Tag)
	}

	if len(lock.Templates) == 0 || len(lock.Overlays) == 0 {
		t.Fatalf("lock checksums missing: %+v", lock)
	}
}

func TestGenerateDryRun(t *testing.T) {
	t.Parallel()

	reg, _ := fixture(t)
	plan := fixturePlan(t, reg)
	target := filepath.Join(t.TempDir(), "dry-run")

	result, err := Generate(t.Context(), Request{
		Registry: &reg,
		Plan:     plan,
		Target:   target,
		NoGit:    true,
		DryRun:   true,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if !result.DryRun {
		t.Fatal("result must be marked as a dry run")
	}

	if len(result.Files) == 0 {
		t.Fatal("dry run must report generated files")
	}

	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("dry run created target: %v", err)
	}
}

func TestGenerateRejectsNonEmptyTarget(t *testing.T) {
	t.Parallel()

	reg, _ := fixture(t)
	plan := fixturePlan(t, reg)
	target := t.TempDir()

	existing := filepath.Join(target, "keep.txt")
	if err := os.WriteFile(existing, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Generate(t.Context(), Request{
		Registry: &reg,
		Plan:     plan,
		Target:   target,
		NoGit:    true,
	})
	if err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Fatalf("error = %v, want not-empty error", err)
	}

	data := mustReadFile(t, existing)
	if string(data) != "keep" {
		t.Fatalf("existing target was modified: %q", data)
	}
}

func TestGenerateLeavesTargetUntouchedOnFailure(t *testing.T) {
	t.Parallel()

	_, registryDir := fixture(t)
	writeFile(
		t,
		filepath.Join(registryDir, "overlays", "logging", "basic", "overlay.yaml"),
		strings.ReplaceAll(overlayYAML, "anchor: imports", "anchor: missing"),
	)

	loaded, err := registry.Load(t.Context(), registry.Options{Source: registryDir})
	if err != nil {
		t.Fatal(err)
	}

	plan := fixturePlan(t, *loaded)

	target := filepath.Join(t.TempDir(), "broken")

	_, err = Generate(t.Context(), Request{
		Registry: loaded,
		Plan:     plan,
		Target:   target,
		NoGit:    true,
	})
	if err == nil {
		t.Fatal("expected generation error")
	}

	if !strings.Contains(err.Error(), "anchor") {
		t.Fatalf("error = %v, want anchor error", err)
	}

	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("failed generation created target: %v", statErr)
	}
}

func TestGenerateHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	reg, _ := fixture(t)
	plan := fixturePlan(t, reg)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := Generate(ctx, Request{
		Registry: &reg,
		Plan:     plan,
		Target:   filepath.Join(t.TempDir(), "canceled"),
		NoGit:    true,
	})
	if err == nil {
		t.Fatal("expected context error")
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
