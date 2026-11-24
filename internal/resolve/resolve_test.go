package resolve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BoatengPeter/smokeforge/internal/manifest"
)

const (
	nextStackID       = "next"
	databaseFeatureID = "database"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func testManifest() *manifest.Manifest {
	return &manifest.Manifest{
		SchemaVersion: 1,
		Name:          "test",
		Kinds: map[string]manifest.Kind{
			"backend":  {Max: 1},
			"frontend": {Max: 1},
		},
		Stacks: []manifest.Stack{
			{
				ID:        "go/chi",
				Kind:      "backend",
				Name:      "Go",
				Template:  "templates/backend/go",
				Variables: []string{"module_path", "greeting"},
			},
			{
				ID:        nextStackID,
				Kind:      "frontend",
				Name:      "Next",
				Template:  "templates/frontend/next",
				Variables: []string{"module_path", "greeting"},
			},
		},
		Features: []manifest.Feature{
			{
				ID:    databaseFeatureID,
				Name:  "Database",
				Kinds: []string{"backend"},
				Variants: []manifest.Variant{
					{
						ID:      "postgres",
						Name:    "PostgreSQL",
						Default: true,
						Overlays: map[string]string{
							"go/chi": "overlays/go/database/postgres",
						},
					},
				},
			},
			{
				ID:    "auth",
				Name:  "Auth",
				Kinds: []string{"backend"},
				Variants: []manifest.Variant{
					{
						ID:       "jwt",
						Name:     "JWT",
						Default:  true,
						Overlays: map[string]string{"go/chi": "overlays/go/auth/jwt"},
					},
				},
			},
			{
				ID:       "totp",
				Name:     "TOTP",
				Kinds:    []string{"backend"},
				Requires: []string{"auth:jwt"},
				Overlays: map[string]string{"go/chi": "overlays/go/totp"},
			},
			{
				ID:       "logging",
				Name:     "Logging",
				Kinds:    []string{"backend"},
				Overlays: map[string]string{"go/chi": "overlays/go/logging"},
			},
		},
	}
}

func testRegistry(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	template := `id: %s
variables:
  - name: module_path
    prompt: Module path
    default: example.com/app
  - name: greeting
    prompt: Greeting
    default: hello
`
	writeFile(t, filepath.Join(dir, "templates", "backend", "go", "template.yaml"),
		strings.ReplaceAll(template, "%s", "go/chi"))
	writeFile(t, filepath.Join(dir, "templates", "frontend", nextStackID, "template.yaml"),
		strings.ReplaceAll(template, "%s", nextStackID))

	for _, overlayPath := range []string{
		"overlays/go/database/postgres",
		"overlays/go/auth/jwt",
		"overlays/go/totp",
		"overlays/go/logging",
	} {
		writeFile(t, filepath.Join(dir, overlayPath, "overlay.yaml"),
			"id: "+filepath.Base(overlayPath)+"\nops:\n  - env:\n      key: TEST_FLAG\n      example: \"1\"\n")
	}

	return dir
}

func baseSelection() Selection {
	return Selection{
		Name:      "Acme App",
		Kind:      "backend",
		Stack:     "go/chi",
		Features:  []string{databaseFeatureID},
		Variables: map[string]string{},
	}
}

func TestResolve(t *testing.T) {
	t.Parallel()

	registryDir := testRegistry(t)

	selection := baseSelection()
	selection.Features = []string{"totp", "auth", databaseFeatureID}

	plan, err := Resolve(testManifest(), registryDir, selection)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if plan.Name != "Acme App" {
		t.Fatalf("name = %q", plan.Name)
	}

	if plan.Target.Stack.ID != "go/chi" {
		t.Fatalf("stack = %q", plan.Target.Stack.ID)
	}

	if plan.Target.Variables["module_path"] != "example.com/app" {
		t.Fatalf("variables = %v", plan.Target.Variables)
	}

	ids := make([]string, 0, len(plan.Target.Overlays))
	for _, overlay := range plan.Target.Overlays {
		ids = append(ids, overlay.FeatureID)
	}

	if strings.Join(ids, ",") != "auth,totp,database" {
		t.Fatalf("overlay order = %v, want auth,totp,database", ids)
	}
}

func TestResolveRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*manifest.Manifest, *Selection)
		want   string
	}{
		{
			name:   "unknown kind",
			mutate: func(_ *manifest.Manifest, sel *Selection) { sel.Kind = "quantum" },
			want:   "kind",
		},
		{
			name:   "unknown stack",
			mutate: func(_ *manifest.Manifest, sel *Selection) { sel.Stack = "nope" },
			want:   "stack",
		},
		{
			name:   "stack kind mismatch",
			mutate: func(_ *manifest.Manifest, sel *Selection) { sel.Stack = nextStackID },
			want:   "kind",
		},
		{
			name: "unsupported feature",
			mutate: func(m *manifest.Manifest, sel *Selection) {
				sel.Kind = "frontend"
				sel.Stack = nextStackID
				sel.Features = []string{databaseFeatureID}
				m.Features[0].Overlays = map[string]string{nextStackID: "overlays/next/database"}
				m.Features[0].Variants[0].Overlays = nil
				m.Features[0].Overlays = map[string]string{nextStackID: "overlays/next/database"}
				m.Features[0].Variants = nil
			},
			want: "feature",
		},
		{
			name: "unknown variant",
			mutate: func(_ *manifest.Manifest, sel *Selection) {
				sel.Features = []string{"database:mysql"}
			},
			want: "variant",
		},
		{
			name: "conflict",
			mutate: func(m *manifest.Manifest, sel *Selection) {
				m.Features[3].Conflicts = []string{databaseFeatureID}
				sel.Features = []string{databaseFeatureID, "logging"}
			},
			want: "conflicts",
		},
		{
			name: "duplicate feature",
			mutate: func(_ *manifest.Manifest, sel *Selection) {
				sel.Features = []string{databaseFeatureID, databaseFeatureID}
			},
			want: "duplicate",
		},
		{
			name:   "missing name",
			mutate: func(_ *manifest.Manifest, sel *Selection) { sel.Name = "" },
			want:   "name",
		},
		{
			name:   "unknown variable",
			mutate: func(_ *manifest.Manifest, sel *Selection) { sel.Variables = map[string]string{"typo": "x"} },
			want:   "variable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := testManifest()
			registryDir := testRegistry(t)
			sel := baseSelection()
			tt.mutate(m, &sel)

			_, err := Resolve(m, registryDir, sel)
			if err == nil {
				t.Fatal("expected error")
			}

			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error %q does not contain %q", err, tt.want)
			}
		})
	}
}

func variablesFixture() (manifest.Stack, manifest.Template) {
	stack := testManifest().Stacks[0]
	tmpl := manifest.Template{
		ID: "go/chi",
		Variables: []manifest.TemplateVariable{
			{Name: "module_path", Default: "example.com/default"},
			{Name: "greeting", Default: "hello"},
		},
	}

	return stack, tmpl
}

func TestVariables(t *testing.T) {
	t.Parallel()

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()

		stack, tmpl := variablesFixture()

		values, err := Variables(stack, tmpl, map[string]string{})
		if err != nil {
			t.Fatalf("resolve variables: %v", err)
		}

		if values["module_path"] != "example.com/default" {
			t.Fatalf("variables = %v", values)
		}
	})

	t.Run("overrides", func(t *testing.T) {
		t.Parallel()

		stack, tmpl := variablesFixture()

		values, err := Variables(
			stack,
			tmpl,
			map[string]string{"module_path": "example.com/custom"},
		)
		if err != nil {
			t.Fatalf("resolve variables: %v", err)
		}

		if values["module_path"] != "example.com/custom" {
			t.Fatalf("variables = %v", values)
		}
	})

	t.Run("missing without default", func(t *testing.T) {
		t.Parallel()

		stack, tmpl := variablesFixture()
		tmpl.Variables[1].Default = ""

		_, err := Variables(stack, tmpl, map[string]string{})
		if err == nil || !strings.Contains(err.Error(), "greeting") {
			t.Fatalf("error = %v, want greeting", err)
		}
	})

	t.Run("unknown", func(t *testing.T) {
		t.Parallel()

		stack, tmpl := variablesFixture()

		_, err := Variables(stack, tmpl, map[string]string{"typo": "x"})
		if err == nil || !strings.Contains(err.Error(), "typo") {
			t.Fatalf("error = %v, want typo", err)
		}
	})

	t.Run("stack and template mismatch", func(t *testing.T) {
		t.Parallel()

		stack, tmpl := variablesFixture()
		tmpl.Variables = tmpl.Variables[:1]

		_, err := Variables(stack, tmpl, map[string]string{})
		if err == nil || !strings.Contains(err.Error(), "mismatch") {
			t.Fatalf("error = %v, want mismatch", err)
		}
	})
}

func nextVariantManifest() *manifest.Manifest {
	return &manifest.Manifest{
		SchemaVersion: 1,
		Name:          "variant-test",
		Kinds: map[string]manifest.Kind{
			"backend":  {Max: 1},
			"frontend": {Max: 1},
		},
		Stacks: []manifest.Stack{
			{ID: "go/chi", Kind: "backend", Name: "Go", Template: "templates/go", Variables: []string{"module_path"}},
			{ID: nextStackID, Kind: "frontend", Name: "Next", Template: "templates/next", Variables: []string{"package_manager", "site_description"}},
		},
		Features: []manifest.Feature{
			{
				ID:    databaseFeatureID,
				Name:  "Database",
				Kinds: []string{"backend", "frontend"},
				Variants: []manifest.Variant{
					{ID: "postgres", Default: true},
				},
			},
			{
				ID:    "auth",
				Name:  "Auth",
				Kinds: []string{"backend", "frontend"},
				Variants: []manifest.Variant{
					{ID: "jwt", Default: true, Stacks: []string{"go/chi"}},
					{ID: "session", Stacks: []string{nextStackID}, Requires: []string{"database:postgres"}},
				},
			},
			{
				ID:       "totp",
				Name:     "TOTP",
				Kinds:    []string{"backend", "frontend"},
				Requires: []string{"auth"},
			},
		},
	}
}

func variantRegistry(t *testing.T) (string, *manifest.Manifest) {
	t.Helper()

	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "templates", "go", "template.yaml"), `id: go/chi
variables:
  - name: module_path
    prompt: Module path
    default: example.com/app
`)
	writeFile(t, filepath.Join(dir, "templates", nextStackID, "template.yaml"), `id: next
variables:
  - name: package_manager
    prompt: Package manager
    default: pnpm
    choices: [pnpm, npm, bun]
  - name: site_description
    prompt: Site description
    default: A modern web application.
`)

	for _, overlay := range []string{
		"overlays/go/database/postgres",
		"overlays/next/database/postgres",
		"overlays/go/auth/jwt",
		"overlays/next/auth/session",
		"overlays/go/totp",
		"overlays/next/totp",
	} {
		writeFile(t, filepath.Join(dir, overlay, "overlay.yaml"),
			"id: "+filepath.Base(overlay)+"\nops:\n  - env:\n      key: TEST_FLAG\n      example: \"1\"\n")
	}

	m := nextVariantManifest()
	m.Features[0].Variants[0].Overlays = map[string]string{
		"go/chi":    "overlays/go/database/postgres",
		nextStackID: "overlays/next/database/postgres",
	}
	m.Features[1].Variants[0].Overlays = map[string]string{
		"go/chi": "overlays/go/auth/jwt",
	}
	m.Features[1].Variants[1].Overlays = map[string]string{
		nextStackID: "overlays/next/auth/session",
	}
	m.Features[2].Overlays = map[string]string{
		"go/chi":    "overlays/go/totp",
		nextStackID: "overlays/next/totp",
	}

	return dir, m
}

func TestResolveVariantSelection(t *testing.T) {
	t.Parallel()

	registryDir, m := variantRegistry(t)

	plan, err := Resolve(m, registryDir, Selection{
		Name:      "Next App",
		Kind:      "frontend",
		Stack:     nextStackID,
		Features:  []string{"auth", databaseFeatureID},
		Variables: map[string]string{"package_manager": "pnpm"},
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if len(plan.Target.Overlays) != 2 {
		t.Fatalf("overlays = %+v", plan.Target.Overlays)
	}

	if plan.Target.Overlays[0].FeatureID != databaseFeatureID {
		t.Fatalf("order = %+v, want database first", plan.Target.Overlays)
	}

	if plan.Target.Overlays[1].FeatureID != "auth" || plan.Target.Overlays[1].VariantID != "session" {
		t.Fatalf("auth variant = %+v, want session", plan.Target.Overlays[1])
	}
}

func TestResolveAutoAddsRequirements(t *testing.T) {
	t.Parallel()

	registryDir, m := variantRegistry(t)

	plan, err := Resolve(m, registryDir, Selection{
		Name:      "Next App",
		Kind:      "frontend",
		Stack:     nextStackID,
		Features:  []string{"auth"},
		Variables: map[string]string{"package_manager": "pnpm"},
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if len(plan.Target.Overlays) != 2 {
		t.Fatalf("overlays = %+v, want database and auth", plan.Target.Overlays)
	}

	if plan.Target.Overlays[0].FeatureID != databaseFeatureID {
		t.Fatalf("first overlay = %+v, want database", plan.Target.Overlays[0])
	}

	if plan.Target.Overlays[1].FeatureID != "auth" {
		t.Fatalf("second overlay = %+v, want auth", plan.Target.Overlays[1])
	}
}

func TestResolvePresetFeatures(t *testing.T) {
	t.Parallel()

	registryDir, m := variantRegistry(t)
	m.Presets = []manifest.Preset{
		{ID: "fullstack", Name: "Fullstack", Default: true, Features: []string{"auth"}},
	}

	plan, err := Resolve(m, registryDir, Selection{
		Name:      "Next App",
		Kind:      "frontend",
		Stack:     nextStackID,
		Preset:    "fullstack",
		Variables: map[string]string{"package_manager": "pnpm"},
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if len(plan.Target.Overlays) != 2 {
		t.Fatalf("overlays = %+v, want database and auth", plan.Target.Overlays)
	}
}

func TestResolvePresetKindMismatch(t *testing.T) {
	t.Parallel()

	registryDir, m := variantRegistry(t)
	m.Presets = []manifest.Preset{
		{ID: "web", Name: "Web", Kinds: []string{"frontend"}, Features: []string{"auth"}},
	}

	_, err := Resolve(m, registryDir, Selection{
		Name:      "Go App",
		Kind:      "backend",
		Stack:     "go/chi",
		Preset:    "web",
		Variables: map[string]string{"module_path": "example.com/app"},
	})
	if err == nil || !strings.Contains(err.Error(), "does not support kind") {
		t.Fatalf("error = %v, want kind mismatch", err)
	}
}

func TestResolveUnknownPreset(t *testing.T) {
	t.Parallel()

	registryDir, m := variantRegistry(t)

	_, err := Resolve(m, registryDir, Selection{
		Name:      "Next App",
		Kind:      "frontend",
		Stack:     nextStackID,
		Preset:    "missing",
		Variables: map[string]string{"package_manager": "pnpm"},
	})
	if err == nil || !strings.Contains(err.Error(), "preset") {
		t.Fatalf("error = %v, want preset error", err)
	}
}

func TestResolveVariantUnsupportedStack(t *testing.T) {
	t.Parallel()

	registryDir, m := variantRegistry(t)

	_, err := Resolve(m, registryDir, Selection{
		Name:      "Next App",
		Kind:      "frontend",
		Stack:     nextStackID,
		Features:  []string{databaseFeatureID, "auth:jwt"},
		Variables: map[string]string{"package_manager": "pnpm"},
	})
	if err == nil || !strings.Contains(err.Error(), "does not support stack") {
		t.Fatalf("error = %v, want unsupported variant error", err)
	}
}

func TestResolveVariableChoices(t *testing.T) {
	t.Parallel()

	registryDir, m := variantRegistry(t)

	_, err := Resolve(m, registryDir, Selection{
		Name:      "Next App",
		Kind:      "frontend",
		Stack:     nextStackID,
		Variables: map[string]string{"package_manager": "yarn"},
	})
	if err == nil || !strings.Contains(err.Error(), "must be one of") {
		t.Fatalf("error = %v, want choices error", err)
	}
}

func TestResolveBackendDefaultVariant(t *testing.T) {
	t.Parallel()

	registryDir, m := variantRegistry(t)

	plan, err := Resolve(m, registryDir, Selection{
		Name:      "Go App",
		Kind:      "backend",
		Stack:     "go/chi",
		Features:  []string{"auth"},
		Variables: map[string]string{"module_path": "example.com/app"},
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if len(plan.Target.Overlays) != 1 || plan.Target.Overlays[0].VariantID != "jwt" {
		t.Fatalf("overlays = %+v, want jwt", plan.Target.Overlays)
	}
}

//nolint:funlen // Test fixture keeps the fullstack shape visible in one place.
func fullstackManifest() *manifest.Manifest {
	return &manifest.Manifest{
		SchemaVersion: 1,
		Name:          "fullstack",
		Kinds: map[string]manifest.Kind{
			"backend":   {Max: 1},
			"frontend":  {Max: 1},
			"fullstack": {Max: 1},
			"mobile":    {Max: 2},
			"desktop":   {Max: 1},
		},
		Stacks: []manifest.Stack{
			{
				ID:        "go/api",
				Kind:      "backend",
				Name:      "Go API",
				Template:  "templates/api",
				Variables: []string{"module_path"},
			},
			{
				ID:        nextStackID,
				Kind:      "frontend",
				Name:      "Next.js",
				Template:  "templates/web",
				Variables: []string{"package_manager", "module_path"},
			},
			{
				ID:        "android/compose",
				Kind:      "mobile",
				Name:      "Android",
				Template:  "templates/android",
				Variables: []string{"application_id", "package_name", "api_base_url"},
			},
			{
				ID:        "ios/swiftui",
				Kind:      "mobile",
				Name:      "iOS",
				Template:  "templates/ios",
				Variables: []string{"bundle_id", "ios_api_base_url"},
			},
			{
				ID:        "desktop/electron",
				Kind:      "desktop",
				Name:      "Electron",
				Template:  "templates/desktop",
				Variables: []string{"bundle_id", "package_manager", "api_url"},
			},
			{
				ID:        "desktop/egui",
				Kind:      "desktop",
				Name:      "egui",
				Template:  "templates/desktop-egui",
				Variables: []string{"crate_name", "bundle_id", "api_url"},
			},
			{
				ID:        "workspace/mobile",
				Kind:      "fullstack",
				Name:      "Mobile workspace",
				Template:  "templates/root-mobile",
				Variables: []string{"package_manager"},
			},
			{
				ID:        "workspace/fullstack",
				Kind:      "fullstack",
				Name:      "Fullstack workspace",
				Template:  "templates/root",
				Variables: []string{"package_manager"},
			},
		},
		Presets: []manifest.Preset{
			{
				ID:       "fullstack-api",
				Name:     "Fullstack API",
				Kinds:    []string{"fullstack"},
				Default:  true,
				Features: []string{"database:postgres", "auth:jwt", "data:tanstack-query", "docker", "ci:github"},
			},
			{
				ID:       "mobile-api",
				Name:     "Mobile API",
				Kinds:    []string{"fullstack", "mobile"},
				Features: []string{"database:postgres", "auth:jwt", "ci:github"},
			},
		},
		Features: []manifest.Feature{
			{
				ID:       databaseFeatureID,
				Kinds:    []string{"backend"},
				Variants: []manifest.Variant{{ID: "postgres", Default: true, Overlays: map[string]string{"go/api": "overlays/api/db"}}},
			},
			{
				ID:    "auth",
				Kinds: []string{"backend", "frontend", "desktop"},
				Variants: []manifest.Variant{
					{ID: "jwt", Requires: []string{"database:postgres"}, Overlays: map[string]string{"go/api": "overlays/api/auth"}},
					{ID: "bff", Requires: []string{"data:tanstack-query"}, Overlays: map[string]string{nextStackID: "overlays/web/bff"}},
					{ID: "electron", Requires: []string{"data:tanstack-query"}, Overlays: map[string]string{"desktop/electron": "overlays/desktop/auth"}},
				},
			},
			{
				ID:    "state",
				Kinds: []string{"frontend", "desktop"},
				Variants: []manifest.Variant{{ID: "zustand", Default: true, Overlays: map[string]string{
					nextStackID: "overlays/web/state", "desktop/electron": "overlays/desktop/state",
				}}},
			},
			{
				ID:    "sync",
				Kinds: []string{"backend", "desktop"},
				Variants: []manifest.Variant{
					{ID: "library", Requires: []string{"database:postgres", "auth:jwt"}, Overlays: map[string]string{"go/api": "overlays/api/sync"}},
					{ID: "library-client", Overlays: map[string]string{"desktop/electron": "overlays/desktop/sync"}},
				},
			},
			{
				ID:    "data",
				Kinds: []string{"frontend", "desktop"},
				Variants: []manifest.Variant{{ID: "tanstack-query", Default: true, Overlays: map[string]string{
					nextStackID: "overlays/web/query", "desktop/electron": "overlays/desktop/query",
				}}},
			},
			{
				ID:    "storage",
				Kinds: []string{"desktop"},
				Variants: []manifest.Variant{{ID: "sqlite", Default: true, Overlays: map[string]string{
					"desktop/electron": "overlays/desktop/storage",
				}}},
			},
			{
				ID:       "docker",
				Kinds:    []string{"backend", "frontend", "fullstack"},
				Overlays: map[string]string{"go/api": "overlays/api/docker", nextStackID: "overlays/web/docker", "workspace/fullstack": "overlays/root/docker"},
			},
			{
				ID:    "ci",
				Kinds: []string{"backend", "frontend", "fullstack"},
				Variants: []manifest.Variant{{ID: "github", Default: true, Overlays: map[string]string{
					"go/api": "overlays/api/ci", nextStackID: "overlays/web/ci", "workspace/fullstack": "overlays/root/ci", "workspace/mobile": "overlays/workspace/mobile/ci",
				}}},
			},
		},
	}
}

func writeFullstackRegistry(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "templates", "api", "template.yaml"),
		"id: go/api\nvariables:\n  - name: module_path\n    default: example.com/api\n")
	writeFile(t, filepath.Join(dir, "templates", "web", "template.yaml"),
		"id: next\nvariables:\n  - name: package_manager\n    default: pnpm\n  - name: module_path\n    default: web\n")
	writeFile(t, filepath.Join(dir, "templates", "root", "template.yaml"),
		"id: workspace/fullstack\nvariables:\n  - name: package_manager\n    default: pnpm\n")
	writeFile(t, filepath.Join(dir, "templates", "root-mobile", "template.yaml"),
		"id: workspace/mobile\nvariables:\n  - name: package_manager\n    default: pnpm\n")
	writeFile(t, filepath.Join(dir, "templates", "android", "template.yaml"),
		"id: android/compose\nvariables:\n  - name: application_id\n    default: com.example.app\n  - name: package_name\n    default: com.example.app\n  - name: api_base_url\n    default: http://10.0.2.2:8080\n")
	writeFile(t, filepath.Join(dir, "templates", "ios", "template.yaml"),
		"id: ios/swiftui\nvariables:\n  - name: bundle_id\n    default: com.example.app\n  - name: ios_api_base_url\n    default: http://127.0.0.1:8080\n")
	writeFile(t, filepath.Join(dir, "templates", "desktop", "template.yaml"),
		"id: desktop/electron\nvariables:\n  - name: bundle_id\n    default: com.example.app\n  - name: package_manager\n    default: pnpm\n  - name: api_url\n    default: http://localhost:8080\n")
	writeFile(t, filepath.Join(dir, "templates", "desktop-egui", "template.yaml"),
		"id: desktop/egui\nvariables:\n  - name: crate_name\n    default: example_app\n  - name: bundle_id\n    default: com.example.app\n  - name: api_url\n    default: http://localhost:8080\n")

	for _, path := range []string{
		"overlays/api/db", "overlays/api/auth", "overlays/web/bff",
		"overlays/web/query", "overlays/desktop/query", "overlays/desktop/state", "overlays/desktop/storage",
		"overlays/desktop/auth", "overlays/desktop/sync", "overlays/api/sync", "overlays/api/docker", "overlays/web/docker",
		"overlays/root/docker", "overlays/workspace/fullstack/mobile", "overlays/workspace/mobile/ci", "overlays/workspace/fullstack/desktop", "overlays/workspace/fullstack/desktop-egui", "overlays/api/ci", "overlays/web/ci", "overlays/root/ci",
	} {
		writeFile(t, filepath.Join(dir, path, "overlay.yaml"),
			"id: "+strings.ReplaceAll(path, "overlays/", "")+"\nops:\n  - env:\n      key: TEST_FLAG\n      example: \"1\"\n")
	}

	return dir
}

func TestResolveFullstack(t *testing.T) {
	t.Parallel()

	plan, err := Resolve(fullstackManifest(), writeFullstackRegistry(t), Selection{
		Name:      "Acme App",
		Kind:      "fullstack",
		Backend:   "go/api",
		Frontend:  "next",
		Preset:    "fullstack-api",
		Variables: map[string]string{"module_path": "example.com/acme", "package_manager": "pnpm"},
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	assertFullstackTargets(t, plan)
	assertFullstackOverlays(t, plan)
	assertFullstackVariables(t, plan)
}

func TestResolveFullstackDesktop(t *testing.T) {
	t.Parallel()

	plan, err := Resolve(fullstackManifest(), writeFullstackRegistry(t), Selection{
		Name:     "Acme Desktop",
		Kind:     "fullstack",
		Backend:  "go/api",
		Frontend: "next",
		Desktop:  "electron",
		Preset:   "fullstack-api",
		Variables: map[string]string{
			"module_path":     "example.com/acme",
			"package_manager": "pnpm",
			"bundle_id":       "com.example.acme",
			"api_url":         "http://localhost:8080",
		},
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if len(plan.Targets) != 3 {
		t.Fatalf("targets = %+v", plan.Targets)
	}

	if plan.Targets[2].Path != "apps/desktop" || plan.Targets[2].Stack.ID != "desktop/electron" {
		t.Fatalf("desktop target = %+v", plan.Targets[2])
	}

	if got := overlayIDs(plan.Targets[2].Overlays); strings.Join(got, ",") != "data:tanstack-query,auth:electron,state:zustand,storage:sqlite,sync:library-client" {
		t.Fatalf("desktop overlays = %v", got)
	}

	if got := overlayIDs(plan.Root.Overlays); strings.Join(got, ",") != "docker,ci:github,desktop" {
		t.Fatalf("root overlays = %v", got)
	}

	if plan.Targets[2].Variables["bundle_id"] != "com.example.acme" {
		t.Fatalf("desktop variables = %v", plan.Targets[2].Variables)
	}
}

const appsAndroidPath = "apps/android"

func TestResolveFullstackMobile(t *testing.T) {
	t.Parallel()

	plan, err := Resolve(fullstackManifest(), writeFullstackRegistry(t), Selection{
		Name:    "Acme Mobile",
		Kind:    "fullstack",
		Backend: "go/api",
		Mobiles: []string{"android/compose"},
		Preset:  "mobile-api",
		Variables: map[string]string{
			"module_path":     "example.com/acme",
			"package_manager": "pnpm",
			"application_id":  "com.example.acme",
			"package_name":    "com.example.acme",
			"api_base_url":    "http://10.0.2.2:8080",
		},
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if len(plan.Targets) != 2 {
		t.Fatalf("targets = %+v", plan.Targets)
	}

	if plan.Targets[1].Path != appsAndroidPath || plan.Targets[1].Stack.ID != "android/compose" {
		t.Fatalf("mobile target = %+v", plan.Targets[1])
	}

	if plan.Root == nil || plan.Root.Stack.ID != "workspace/mobile" {
		t.Fatalf("root = %+v", plan.Root)
	}

	if got := overlayIDs(plan.Root.Overlays); strings.Join(got, ",") != "ci:github" {
		t.Fatalf("root overlays = %v", got)
	}

	if plan.Targets[1].Variables["package_name"] != "com.example.acme" {
		t.Fatalf("mobile variables = %v", plan.Targets[1].Variables)
	}
}

func TestResolveFullstackMobileRequiresAPIJWT(t *testing.T) {
	t.Parallel()

	m := fullstackManifest()
	m.Presets[1].Features = []string{"database:postgres"}

	_, err := Resolve(m, writeFullstackRegistry(t), Selection{
		Name:    "Acme Mobile",
		Kind:    "fullstack",
		Backend: "go/api",
		Mobiles: []string{"android/compose"},
		Preset:  "mobile-api",
		Variables: map[string]string{
			"module_path":     "example.com/acme",
			"package_manager": "pnpm",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "requires auth:jwt") {
		t.Fatalf("error = %v, want auth requirement", err)
	}
}

func TestResolveMobileBackend(t *testing.T) {
	t.Parallel()

	plan, err := Resolve(fullstackManifest(), writeFullstackRegistry(t), Selection{
		Name:    "Acme Mobile API",
		Kind:    "mobile",
		Stack:   "android/compose",
		Backend: "go/api",
		Preset:  "mobile-api",
		Variables: map[string]string{
			"module_path":     "example.com/acme",
			"package_manager": "pnpm",
			"application_id":  "com.example.acme",
			"package_name":    "com.example.acme",
			"api_base_url":    "http://10.0.2.2:8080",
		},
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if plan.Kind != "mobile" {
		t.Fatalf("kind = %q, want mobile", plan.Kind)
	}

	if len(plan.Targets) != 2 || plan.Targets[1].Path != appsAndroidPath {
		t.Fatalf("targets = %+v", plan.Targets)
	}

	if plan.Root == nil || plan.Root.Stack.ID != "workspace/mobile" {
		t.Fatalf("root = %+v", plan.Root)
	}
}

func TestResolveFullstackBothMobiles(t *testing.T) {
	t.Parallel()

	plan, err := Resolve(fullstackManifest(), writeFullstackRegistry(t), Selection{
		Name:    "Acme Both Mobile",
		Kind:    "fullstack",
		Backend: "go/api",
		Mobiles: []string{"android/compose", "ios/swiftui"},
		Preset:  "mobile-api",
		Variables: map[string]string{
			"module_path":      "example.com/acme",
			"package_manager":  "pnpm",
			"application_id":   "com.example.acme",
			"package_name":     "com.example.acme",
			"api_base_url":     "http://10.0.2.2:8080",
			"bundle_id":        "com.example.acme",
			"ios_api_base_url": "http://127.0.0.1:8080",
		},
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if len(plan.Targets) != 3 {
		t.Fatalf("targets = %+v", plan.Targets)
	}

	if plan.Targets[1].Path != appsAndroidPath || plan.Targets[2].Path != "apps/ios" {
		t.Fatalf("mobile targets = %+v", plan.Targets)
	}

	if plan.Targets[2].Stack.ID != "ios/swiftui" {
		t.Fatalf("iOS target = %+v", plan.Targets[2])
	}
}

func TestResolveFullstackAPIPresetRequiresWeb(t *testing.T) {
	t.Parallel()

	_, err := Resolve(fullstackManifest(), writeFullstackRegistry(t), Selection{
		Name:    "Acme Mobile",
		Kind:    "fullstack",
		Backend: "go/api",
		Mobiles: []string{"ios/swiftui"},
		Preset:  "fullstack-api",
		Variables: map[string]string{
			"module_path":      "example.com/acme",
			"package_manager":  "pnpm",
			"bundle_id":        "com.example.acme",
			"ios_api_base_url": "http://127.0.0.1:8080",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "requires --frontend next") {
		t.Fatalf("error = %v, want explicit frontend guidance", err)
	}
}

func assertFullstackTargets(t *testing.T, plan *Plan) {
	t.Helper()

	if len(plan.Targets) != 2 {
		t.Fatalf("targets = %+v", plan.Targets)
	}

	if plan.Targets[0].Path != "apps/api" || plan.Targets[0].Stack.ID != "go/api" {
		t.Fatalf("api target = %+v", plan.Targets[0])
	}

	if plan.Targets[1].Path != "apps/web" || plan.Targets[1].Stack.ID != nextStackID {
		t.Fatalf("web target = %+v", plan.Targets[1])
	}

	if plan.Root == nil || plan.Root.Path != "." || plan.Root.Stack.ID != "workspace/fullstack" {
		t.Fatalf("root = %+v", plan.Root)
	}
}

func assertFullstackOverlays(t *testing.T, plan *Plan) {
	t.Helper()

	if got := overlayIDs(plan.Targets[0].Overlays); strings.Join(got, ",") != "database:postgres,auth:jwt,docker" {
		t.Fatalf("api overlays = %v", got)
	}

	if got := overlayIDs(plan.Targets[1].Overlays); strings.Join(got, ",") != "data:tanstack-query,auth:bff,docker" {
		t.Fatalf("web overlays = %v", got)
	}

	if got := overlayIDs(plan.Root.Overlays); strings.Join(got, ",") != "docker,ci:github" {
		t.Fatalf("root overlays = %v", got)
	}
}

func assertFullstackVariables(t *testing.T, plan *Plan) {
	t.Helper()

	if plan.Targets[0].Variables["module_path"] != "example.com/acme" {
		t.Fatalf("api variables = %v", plan.Targets[0].Variables)
	}

	if plan.Targets[1].Variables["package_manager"] != "pnpm" {
		t.Fatalf("web variables = %v", plan.Targets[1].Variables)
	}
}

func overlayIDs(overlays []OverlayPlan) []string {
	ids := make([]string, 0, len(overlays))
	for _, overlay := range overlays {
		id := overlay.FeatureID
		if overlay.VariantID != "" {
			id += ":" + overlay.VariantID
		}

		ids = append(ids, id)
	}

	return ids
}

func TestSelectFeaturesRejectsTwoVariantsOfOneFeature(t *testing.T) {
	t.Parallel()

	m := &manifest.Manifest{
		Features: []manifest.Feature{{
			ID:    "auth",
			Kinds: []string{"desktop"},
			Variants: []manifest.Variant{
				{ID: "electron", Overlays: map[string]string{"desktop/electron": "auth/electron"}},
				{ID: "oauth-pkce", Overlays: map[string]string{"desktop/electron": "auth/oauth-pkce"}},
			},
		}},
		Stacks: []manifest.Stack{{ID: "desktop/electron", Kind: "desktop"}},
	}

	_, err := selectFeatures(m, manifest.Stack{ID: "desktop/electron", Kind: "desktop"}, []string{"auth:electron", "auth:oauth-pkce"})
	if err == nil || !strings.Contains(err.Error(), "choose one variant") {
		t.Fatalf("error = %v, want one-variant guidance", err)
	}
}

func TestResolveFullstackDesktopEgui(t *testing.T) {
	t.Parallel()

	plan, err := Resolve(fullstackManifest(), writeFullstackRegistry(t), Selection{
		Name:     "Acme Egui",
		Kind:     "fullstack",
		Backend:  "go/api",
		Frontend: "next",
		Desktop:  "egui",
		Preset:   "fullstack-api",
		Variables: map[string]string{
			"module_path":     "example.com/acme",
			"package_manager": "pnpm",
			"crate_name":      "acme_desktop",
			"bundle_id":       "com.example.acme",
			"api_url":         "http://localhost:8080",
		},
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if len(plan.Targets) != 3 {
		t.Fatalf("targets = %+v", plan.Targets)
	}

	if plan.Targets[2].Path != "apps/desktop" || plan.Targets[2].Stack.ID != "desktop/egui" {
		t.Fatalf("desktop target = %+v", plan.Targets[2])
	}

	if len(plan.Targets[2].Overlays) != 0 {
		t.Fatalf("egui desktop overlays = %+v, want none", plan.Targets[2].Overlays)
	}

	if got := overlayIDs(plan.Root.Overlays); strings.Join(got, ",") != "docker,ci:github,desktop-egui" {
		t.Fatalf("root overlays = %v", got)
	}
}
