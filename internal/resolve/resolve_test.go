package resolve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BoatengPeter/smokeforge/internal/manifest"
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
				ID:        "next",
				Kind:      "frontend",
				Name:      "Next",
				Template:  "templates/frontend/next",
				Variables: []string{"module_path", "greeting"},
			},
		},
		Features: []manifest.Feature{
			{
				ID:    "database",
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
	writeFile(t, filepath.Join(dir, "templates", "frontend", "next", "template.yaml"),
		strings.ReplaceAll(template, "%s", "next"))

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
		Features:  []string{"database"},
		Variables: map[string]string{},
	}
}

func TestResolve(t *testing.T) {
	t.Parallel()

	registryDir := testRegistry(t)

	selection := baseSelection()
	selection.Features = []string{"totp", "auth", "database"}

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
			mutate: func(_ *manifest.Manifest, sel *Selection) { sel.Stack = "next" },
			want:   "kind",
		},
		{
			name: "unsupported feature",
			mutate: func(m *manifest.Manifest, sel *Selection) {
				sel.Kind = "frontend"
				sel.Stack = "next"
				sel.Features = []string{"database"}
				m.Features[0].Overlays = map[string]string{"next": "overlays/next/database"}
				m.Features[0].Variants[0].Overlays = nil
				m.Features[0].Overlays = map[string]string{"next": "overlays/next/database"}
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
				m.Features[3].Conflicts = []string{"database"}
				sel.Features = []string{"database", "logging"}
			},
			want: "conflicts",
		},
		{
			name: "duplicate feature",
			mutate: func(_ *manifest.Manifest, sel *Selection) {
				sel.Features = []string{"database", "database"}
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
			{ID: "next", Kind: "frontend", Name: "Next", Template: "templates/next", Variables: []string{"package_manager", "site_description"}},
		},
		Features: []manifest.Feature{
			{
				ID:    "database",
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
					{ID: "session", Stacks: []string{"next"}, Requires: []string{"database:postgres"}},
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
	writeFile(t, filepath.Join(dir, "templates", "next", "template.yaml"), `id: next
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
		"go/chi": "overlays/go/database/postgres",
		"next":   "overlays/next/database/postgres",
	}
	m.Features[1].Variants[0].Overlays = map[string]string{
		"go/chi": "overlays/go/auth/jwt",
	}
	m.Features[1].Variants[1].Overlays = map[string]string{
		"next": "overlays/next/auth/session",
	}
	m.Features[2].Overlays = map[string]string{
		"go/chi": "overlays/go/totp",
		"next":   "overlays/next/totp",
	}

	return dir, m
}

func TestResolveVariantSelection(t *testing.T) {
	t.Parallel()

	registryDir, m := variantRegistry(t)

	plan, err := Resolve(m, registryDir, Selection{
		Name:      "Next App",
		Kind:      "frontend",
		Stack:     "next",
		Features:  []string{"auth", "database"},
		Variables: map[string]string{"package_manager": "pnpm"},
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if len(plan.Target.Overlays) != 2 {
		t.Fatalf("overlays = %+v", plan.Target.Overlays)
	}

	if plan.Target.Overlays[0].FeatureID != "database" {
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
		Stack:     "next",
		Features:  []string{"auth"},
		Variables: map[string]string{"package_manager": "pnpm"},
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if len(plan.Target.Overlays) != 2 {
		t.Fatalf("overlays = %+v, want database and auth", plan.Target.Overlays)
	}

	if plan.Target.Overlays[0].FeatureID != "database" {
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
		Stack:     "next",
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
		Stack:     "next",
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
		Stack:     "next",
		Features:  []string{"database", "auth:jwt"},
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
		Stack:     "next",
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
