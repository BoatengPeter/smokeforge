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

const cliFullstackManifestYAML = `schemaVersion: 1
name: fullstack-cli
kinds:
  backend: {max: 1}
  frontend: {max: 1}
  mobile: {max: 2}
  fullstack: {max: 1}
stacks:
  - id: go/api
    kind: backend
    name: Go API
    template: templates/api
    variables: [module_path]
  - id: next
    kind: frontend
    name: Next.js
    template: templates/web
    variables: [package_manager]
  - id: android/compose
    kind: mobile
    name: Android
    template: templates/android
    variables: [application_id, package_name, api_base_url]
  - id: ios/swiftui
    kind: mobile
    name: iOS
    template: templates/ios
    variables: [bundle_id, ios_api_base_url]
  - id: workspace/mobile
    kind: fullstack
    name: Mobile Workspace
    template: templates/root-mobile
    variables: [package_manager]
  - id: workspace/fullstack
    kind: fullstack
    name: Workspace
    template: templates/root
    variables: [package_manager]
presets:
  - id: fullstack-api
    name: Fullstack API
    kinds: [fullstack]
    default: true
    features: [auth:jwt]
  - id: mobile-api
    name: Mobile API
    kinds: [mobile, fullstack]
    features: [auth:jwt]
features:
  - id: auth
    kinds: [backend, frontend]
    variants:
      - id: jwt
        stacks: [go/api]
        overlays:
          go/api: overlays/api/auth
      - id: bff
        stacks: [next]
        overlays:
          next: overlays/web/bff
`

func writeFullstackCLIRegistry(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "manifest.yaml"), cliFullstackManifestYAML)
	writeTestFile(t, filepath.Join(dir, "templates", "api", "template.yaml"),
		"id: go/api\nvariables:\n  - name: module_path\n    default: example.com/api\n")
	writeTestFile(t, filepath.Join(dir, "templates", "api", "files", "go.mod.tmpl"),
		"module {{.module_path}}\n\ngo 1.26\n")
	writeTestFile(t, filepath.Join(dir, "templates", "web", "template.yaml"),
		"id: next\nvariables:\n  - name: package_manager\n    default: pnpm\n")
	writeTestFile(t, filepath.Join(dir, "templates", "web", "files", "package.json.tmpl"),
		"{\"name\":\"{{.project_slug}}\"}\n")
	writeTestFile(t, filepath.Join(dir, "templates", "android", "template.yaml"),
		"id: android/compose\nvariables:\n  - name: application_id\n    default: com.example.app\n  - name: package_name\n    default: com.example.app\n  - name: api_base_url\n    default: http://10.0.2.2:8080\n")
	writeTestFile(t, filepath.Join(dir, "templates", "android", "files", "README.md.tmpl"),
		"# {{.project_name}}\n")
	writeTestFile(t, filepath.Join(dir, "templates", "ios", "template.yaml"),
		"id: ios/swiftui\nvariables:\n  - name: bundle_id\n    default: com.example.app\n  - name: ios_api_base_url\n    default: http://127.0.0.1:8080\n")
	writeTestFile(t, filepath.Join(dir, "templates", "ios", "files", "README.md.tmpl"),
		"# {{.project_name}}\n")
	writeTestFile(t, filepath.Join(dir, "templates", "root", "template.yaml"),
		"id: workspace/fullstack\nvariables:\n  - name: package_manager\n    default: pnpm\n")
	writeTestFile(t, filepath.Join(dir, "templates", "root-mobile", "template.yaml"),
		"id: workspace/mobile\nvariables:\n  - name: package_manager\n    default: pnpm\n")
	writeTestFile(t, filepath.Join(dir, "templates", "root-mobile", "files", "CONTEXT-MAP.md"),
		"# Context map\n")
	writeTestFile(t, filepath.Join(dir, "overlays", "workspace", "fullstack", "mobile", "overlay.yaml"),
		"id: mobile\nops:\n  - addFiles:\n      - {path: README.md, dest: README.md}\n")
	writeTestFile(t, filepath.Join(dir, "overlays", "workspace", "fullstack", "mobile", "files", "README.md"),
		"# Mobile workspace\n")
	writeTestFile(t, filepath.Join(dir, "overlays", "api", "auth", "overlay.yaml"),
		"id: auth/jwt\nops:\n  - env:\n      key: AUTH_ENABLED\n      example: \"1\"\n")
	writeTestFile(t, filepath.Join(dir, "overlays", "web", "bff", "overlay.yaml"),
		"id: auth/bff\nops:\n  - env:\n      key: BFF_ENABLED\n      example: \"1\"\n")
	writeTestFile(t, filepath.Join(dir, "templates", "root", "files", "CONTEXT-MAP.md"),
		"# Context map\n")

	return dir
}

func TestNewFullstack(t *testing.T) {
	t.Parallel()

	registryDir := writeFullstackCLIRegistry(t)
	target := filepath.Join(t.TempDir(), "acme-fullstack")

	out, _, err := executeCommand(
		t,
		"new", target,
		"--registry", registryDir,
		"--kind", "fullstack",
		"--backend", "go/api",
		"--frontend", "next",
		"--name", "Acme Fullstack",
		"--preset", "fullstack-api",
		"--var", "module_path=example.com/acme",
		"--no-input",
		"--no-git",
	)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	for _, want := range []string{"Acme Fullstack", "apps/api (go/api)", "apps/web (next)"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output %q missing %q", out, want)
		}
	}

	for _, want := range []string{"CONTEXT-MAP.md", "apps/api/go.mod", "apps/web/package.json"} {
		if _, err := os.Stat(filepath.Join(target, filepath.FromSlash(want))); err != nil {
			t.Fatalf("missing %s: %v", want, err)
		}
	}
}

func TestNewFullstackMobile(t *testing.T) {
	t.Parallel()

	registryDir := writeFullstackCLIRegistry(t)
	target := filepath.Join(t.TempDir(), "acme-fullstack-mobile")

	out, _, err := executeCommand(
		t,
		"new", target,
		"--registry", registryDir,
		"--kind", "fullstack",
		"--backend", "go/api",
		"--mobile", "android/compose",
		"--name", "Acme Fullstack Mobile",
		"--preset", "mobile-api",
		"--var", "module_path=example.com/acme",
		"--var", "application_id=com.example.acme",
		"--var", "package_name=com.example.acme",
		"--var", "api_base_url=http://10.0.2.2:8080",
		"--no-input",
		"--no-git",
	)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	for _, want := range []string{"apps/android (android/compose)", "make mobile-build"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output %q missing %q", out, want)
		}
	}

	if _, err := os.Stat(filepath.Join(target, "apps", "android", "README.md")); err != nil {
		t.Fatalf("mobile target missing: %v", err)
	}
}

func TestMobileRequiresFullstack(t *testing.T) {
	t.Parallel()

	registryDir := writeFullstackCLIRegistry(t)

	_, _, err := executeCommand(
		t,
		"new", filepath.Join(t.TempDir(), "invalid-mobile"),
		"--registry", registryDir,
		"--kind", "backend",
		"--stack", "go/api",
		"--mobile", "android/compose",
		"--name", "Invalid Mobile",
		"--no-input",
		"--no-git",
	)
	if err == nil || !strings.Contains(err.Error(), "--mobile is only valid") {
		t.Fatalf("error = %v, want mobile composition restriction", err)
	}
}

func TestNewMobileWithBackend(t *testing.T) {
	t.Parallel()

	registryDir := writeFullstackCLIRegistry(t)
	target := filepath.Join(t.TempDir(), "acme-mobile-api")

	out, _, err := executeCommand(
		t,
		"new", target,
		"--registry", registryDir,
		"--kind", "mobile",
		"--stack", "android/compose",
		"--backend", "go/api",
		"--name", "Acme Mobile API",
		"--var", "module_path=example.com/acme",
		"--var", "application_id=com.example.acme",
		"--var", "package_name=com.example.acme",
		"--var", "api_base_url=http://10.0.2.2:8080",
		"--no-input",
		"--no-git",
	)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	for _, want := range []string{"apps/api (go/api)", "apps/android (android/compose)"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output %q missing %q", out, want)
		}
	}

	if _, err := os.Stat(filepath.Join(target, "apps", "api", "go.mod")); err != nil {
		t.Fatalf("API target missing: %v", err)
	}

	if _, err := os.Stat(filepath.Join(target, "apps", "web")); !os.IsNotExist(err) {
		t.Fatalf("web target should not exist: %v", err)
	}
}

func TestNewFullstackBothMobiles(t *testing.T) {
	t.Parallel()

	registryDir := writeFullstackCLIRegistry(t)
	target := filepath.Join(t.TempDir(), "acme-both-mobile")

	out, _, err := executeCommand(
		t,
		"new", target,
		"--registry", registryDir,
		"--kind", "fullstack",
		"--backend", "go/api",
		"--mobile", "android/compose",
		"--mobile", "ios/swiftui",
		"--name", "Acme Both Mobile",
		"--preset", "mobile-api",
		"--var", "module_path=example.com/acme",
		"--var", "package_manager=pnpm",
		"--var", "application_id=com.example.acme",
		"--var", "package_name=com.example.acme",
		"--var", "api_base_url=http://10.0.2.2:8080",
		"--var", "bundle_id=com.example.acme",
		"--var", "ios_api_base_url=http://127.0.0.1:8080",
		"--no-input",
		"--no-git",
	)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	for _, want := range []string{"apps/android (android/compose)", "apps/ios (ios/swiftui)", "make mobile-build"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output %q missing %q", out, want)
		}
	}

	for _, path := range []string{"apps/android/README.md", "apps/ios/README.md", "apps/api/go.mod"} {
		if _, err := os.Stat(filepath.Join(target, filepath.FromSlash(path))); err != nil {
			t.Fatalf("missing %s: %v", path, err)
		}
	}

	if _, err := os.Stat(filepath.Join(target, "apps", "web")); !os.IsNotExist(err) {
		t.Fatalf("web target should not exist: %v", err)
	}
}

func TestDefaultPresetTracksComposition(t *testing.T) {
	t.Parallel()

	m := &manifest.Manifest{
		Presets: []manifest.Preset{
			{ID: "mobile", Kinds: []string{"mobile"}, Default: true},
			{ID: "mobile-api", Kinds: []string{"mobile", "fullstack"}},
			{ID: "fullstack-api", Kinds: []string{"fullstack"}, Default: true},
		},
	}

	mobileAPI := withDefaultPreset(resolve.Selection{
		Kind:    "mobile",
		Backend: "go/api",
		Stack:   "ios/swiftui",
	}, m)
	if mobileAPI.Preset != "mobile-api" {
		t.Fatalf("mobile+backend preset = %q", mobileAPI.Preset)
	}

	fullstackWeb := withDefaultPreset(resolve.Selection{
		Kind:     "fullstack",
		Backend:  "go/api",
		Frontend: "next",
	}, m)
	if fullstackWeb.Preset != "fullstack-api" {
		t.Fatalf("web fullstack preset = %q", fullstackWeb.Preset)
	}

	fullstackMobile := withDefaultPreset(resolve.Selection{
		Kind:    "fullstack",
		Backend: "go/api",
		Mobiles: []string{"ios/swiftui"},
	}, m)
	if fullstackMobile.Preset != "mobile-api" {
		t.Fatalf("mobile fullstack preset = %q", fullstackMobile.Preset)
	}
}

func TestNextStepLinesDesktop(t *testing.T) {
	t.Parallel()

	plan := &resolve.Plan{
		Target: resolve.TargetPlan{
			Stack: manifest.Stack{ID: "desktop/electron", Name: "Electron"},
		},
	}

	lines := strings.Join(nextStepLines(plan, "my-desktop", true), "\n")
	for _, want := range []string{"make install", "make dev", "make test", "make build"} {
		if !strings.Contains(lines, want) {
			t.Fatalf("desktop next steps missing %q:\n%s", want, lines)
		}
	}
}

const desktopManifestYAML = `schemaVersion: 1
name: desktop-cli
kinds:
  desktop: {max: 1}
stacks:
  - id: desktop/electron
    kind: desktop
    name: Electron
    template: templates/electron
    variables: [bundle_id, package_manager, api_url]
presets:
  - id: desktop
    name: Desktop
    kinds: [desktop]
    default: true
    features: []
features: []
`

func writeDesktopCLIRegistry(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "manifest.yaml"), desktopManifestYAML)
	writeTestFile(t, filepath.Join(dir, "templates", "electron", "template.yaml"),
		"id: desktop/electron\nvariables:\n  - name: bundle_id\n    default: com.example.app\n  - name: package_manager\n    default: pnpm\n  - name: api_url\n    default: http://localhost:8080\n")
	writeTestFile(t, filepath.Join(dir, "templates", "electron", "files", "README.md.tmpl"),
		"# {{.project_name}}\n")
	writeTestFile(t, filepath.Join(dir, "templates", "electron", "files", "package.json.tmpl"),
		"{\"name\":\"{{.project_slug}}\",\"packageManager\":\"{{.package_manager}}\"}\n")
	writeTestFile(t, filepath.Join(dir, "templates", "electron", "files", "src", "main", "index.ts"),
		"export const app = true;\n")

	return dir
}

func TestNewDesktop(t *testing.T) {
	t.Parallel()

	registryDir := writeDesktopCLIRegistry(t)
	target := filepath.Join(t.TempDir(), "desktop")

	out, _, err := executeCommand(
		t,
		"new", target,
		"--registry", registryDir,
		"--kind", "desktop",
		"--stack", "desktop/electron",
		"--name", "Desktop App",
		"--var", "bundle_id=com.example.desktop",
		"--var", "package_manager=npm",
		"--var", "api_url=http://localhost:8080",
		"--no-input",
		"--no-git",
	)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	for _, want := range []string{"Desktop App", "make dev", "make build"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output %q missing %q", out, want)
		}
	}

	if _, err := os.Stat(filepath.Join(target, "src", "main", "index.ts")); err != nil {
		t.Fatalf("desktop entry missing: %v", err)
	}
}

func TestNextStepLinesRustDesktopUseWatchMode(t *testing.T) {
	t.Parallel()

	for _, stack := range []string{"desktop/egui", "desktop/gpui"} {
		plan := &resolve.Plan{
			Target: resolve.TargetPlan{
				Stack: manifest.Stack{ID: stack, Name: stack},
			},
		}

		lines := strings.Join(nextStepLines(plan, "my-desktop", true), "\n")
		for _, want := range []string{
			"make install",
			"make dev-tools",
			"make dev ",
			"make test",
			"make build",
		} {
			if !strings.Contains(lines, want) {
				t.Fatalf("%s next steps missing %q:\n%s", stack, want, lines)
			}
		}
	}
}

func TestNextStepLinesElectronDesktopHotReload(t *testing.T) {
	t.Parallel()

	plan := &resolve.Plan{
		Target: resolve.TargetPlan{
			Stack: manifest.Stack{ID: "desktop/electron", Name: "Electron"},
		},
	}

	lines := strings.Join(nextStepLines(plan, "my-electron", true), "\n")
	if !strings.Contains(lines, "hot reload") {
		t.Fatalf("electron next steps should explain make dev hot reload:\n%s", lines)
	}
}

func TestNewRejectsUnsafeProjectName(t *testing.T) {
	t.Parallel()

	registryDir := writeDesktopCLIRegistry(t)
	target := filepath.Join(t.TempDir(), "unsafe")

	_, _, err := executeCommand(
		t,
		"new", target,
		"--registry", registryDir,
		"--kind", "desktop",
		"--stack", "desktop/electron",
		"--name", "Demo $(touch pwned)",
		"--var", "bundle_id=com.example.desktop",
		"--var", "package_manager=npm",
		"--var", "api_url=http://localhost:8080",
		"--no-input",
		"--no-git",
	)
	if err == nil || !strings.Contains(err.Error(), "project name") {
		t.Fatalf("error = %v, want project name error", err)
	}

	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("target should not be created, stat err = %v", statErr)
	}
}
