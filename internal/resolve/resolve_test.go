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
			name: "missing requirement",
			mutate: func(_ *manifest.Manifest, sel *Selection) {
				sel.Features = []string{"totp"}
			},
			want: "requires",
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
