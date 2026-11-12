package manifest

import (
	"strings"
	"testing"
)

func validManifest() *Manifest {
	return &Manifest{
		SchemaVersion: 1,
		Name:          "test",
		Kinds: map[string]Kind{
			"backend": {Max: 1},
		},
		Stacks: []Stack{
			{
				ID:        "go/chi",
				Kind:      "backend",
				Name:      "Go",
				Template:  "templates/go",
				Variables: []string{"module_path"},
			},
		},
		Presets: []Preset{
			{
				ID:       "fullstack",
				Name:     "Fullstack",
				Default:  true,
				Features: []string{"database:postgres", "auth:jwt"},
			},
			{
				ID:   "minimal",
				Name: "Minimal",
			},
		},
		Features: []Feature{
			{
				ID:    "database",
				Name:  "Database",
				Kinds: []string{"backend"},
				Variants: []Variant{
					{
						ID:       "postgres",
						Name:     "PostgreSQL",
						Default:  true,
						Overlays: map[string]string{"go/chi": "overlays/go/database/postgres"},
					},
				},
			},
			{
				ID:    "auth",
				Name:  "Authentication",
				Kinds: []string{"backend"},
				Variants: []Variant{
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
		},
	}
}

type validationCase struct {
	name   string
	mutate func(*Manifest)
	want   string
}

func runValidationCases(t *testing.T, cases []validationCase) {
	t.Helper()

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := validManifest()
			tt.mutate(m)

			err := m.Validate()
			if err == nil {
				t.Fatal("expected error")
			}

			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error %q does not contain %q", err, tt.want)
			}
		})
	}
}

func TestValidateAcceptsValidManifest(t *testing.T) {
	t.Parallel()

	if err := validManifest().Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestValidateRejectsStructure(t *testing.T) {
	t.Parallel()

	runValidationCases(t, []validationCase{
		{
			name:   "schema version",
			mutate: func(m *Manifest) { m.SchemaVersion = 2 },
			want:   "schema version",
		},
		{
			name:   "empty name",
			mutate: func(m *Manifest) { m.Name = "" },
			want:   "name",
		},
		{
			name:   "empty kinds",
			mutate: func(m *Manifest) { m.Kinds = map[string]Kind{} },
			want:   "kinds",
		},
		{
			name:   "negative kind max",
			mutate: func(m *Manifest) { m.Kinds["backend"] = Kind{Max: -1} },
			want:   "max",
		},
		{
			name:   "empty stacks",
			mutate: func(m *Manifest) { m.Stacks = []Stack{} },
			want:   "stacks",
		},
		{
			name:   "unknown kind",
			mutate: func(m *Manifest) { m.Stacks[0].Kind = "quantum" },
			want:   "kind",
		},
		{
			name: "duplicate stack",
			mutate: func(m *Manifest) {
				m.Stacks = append(m.Stacks, m.Stacks[0])
			},
			want: "duplicate stack",
		},
		{
			name:   "empty template",
			mutate: func(m *Manifest) { m.Stacks[0].Template = "" },
			want:   "template",
		},
		{
			name:   "unsafe template path",
			mutate: func(m *Manifest) { m.Stacks[0].Template = "../templates/go" },
			want:   "template",
		},
		{
			name:   "empty variables",
			mutate: func(m *Manifest) { m.Stacks[0].Variables = []string{} },
			want:   "variables",
		},
		{
			name:   "duplicate variables",
			mutate: func(m *Manifest) { m.Stacks[0].Variables = []string{"port", "port"} },
			want:   "variable",
		},
		{
			name: "duplicate feature",
			mutate: func(m *Manifest) {
				m.Features = append(m.Features, m.Features[0])
			},
			want: "duplicate feature",
		},
		{
			name:   "feature without target",
			mutate: func(m *Manifest) { m.Features[0].Kinds = nil; m.Features[0].Stacks = nil },
			want:   "kinds or stacks",
		},
		{
			name:   "feature unknown kind",
			mutate: func(m *Manifest) { m.Features[0].Kinds = []string{"frontend"} },
			want:   "unknown kind",
		},
		{
			name:   "feature unknown stack",
			mutate: func(m *Manifest) { m.Features[0].Stacks = []string{"nope"} },
			want:   "unknown stack",
		},
	})
}

func TestValidateRejectsVariantsAndOverlays(t *testing.T) {
	t.Parallel()

	runValidationCases(t, []validationCase{
		{
			name: "duplicate variant",
			mutate: func(m *Manifest) {
				m.Features[0].Variants = append(m.Features[0].Variants, m.Features[0].Variants[0])
			},
			want: "duplicate variant",
		},
		{
			name: "two default variants",
			mutate: func(m *Manifest) {
				m.Features[0].Variants = append(
					m.Features[0].Variants,
					Variant{
						ID:       "sqlite",
						Name:     "SQLite",
						Default:  true,
						Overlays: map[string]string{"go/chi": "overlays/go/database/sqlite"},
					},
				)
			},
			want: "default",
		},
		{
			name: "missing feature overlays",
			mutate: func(m *Manifest) {
				m.Features[2].Overlays = nil
			},
			want: "overlay",
		},
		{
			name: "missing variant overlays",
			mutate: func(m *Manifest) {
				m.Features[0].Variants[0].Overlays = nil
			},
			want: "overlay",
		},
		{
			name: "unsafe overlay path",
			mutate: func(m *Manifest) {
				m.Features[2].Overlays = map[string]string{"go/chi": "../escape"}
			},
			want: "overlay",
		},
		{
			name: "overlay unknown stack",
			mutate: func(m *Manifest) {
				m.Features[2].Overlays = map[string]string{"nope": "overlays/go/totp"}
			},
			want: "unknown stack",
		},
		{
			name: "feature overlays with variants",
			mutate: func(m *Manifest) {
				m.Features[0].Overlays = map[string]string{"go/chi": "overlays/go/database"}
			},
			want: "variants",
		},
	})
}

func TestValidateRejectsReferences(t *testing.T) {
	t.Parallel()

	runValidationCases(t, []validationCase{
		{
			name:   "unknown requirement",
			mutate: func(m *Manifest) { m.Features[0].Requires = []string{"nope"} },
			want:   "unknown feature",
		},
		{
			name:   "unknown required variant",
			mutate: func(m *Manifest) { m.Features[2].Requires = []string{"auth:oauth"} },
			want:   "unknown variant",
		},
		{
			name:   "self requirement",
			mutate: func(m *Manifest) { m.Features[0].Requires = []string{"database:postgres"} },
			want:   "itself",
		},
		{
			name:   "unknown conflict",
			mutate: func(m *Manifest) { m.Features[0].Conflicts = []string{"nope"} },
			want:   "unknown feature",
		},
	})
}

func variantManifest() *Manifest {
	return &Manifest{
		SchemaVersion: 1,
		Name:          "variant-test",
		Kinds: map[string]Kind{
			"backend":  {Max: 1},
			"frontend": {Max: 1},
		},
		Stacks: []Stack{
			{ID: "go/chi", Kind: "backend", Name: "Go", Template: "templates/go", Variables: []string{"module_path"}},
			{ID: "next", Kind: "frontend", Name: "Next", Template: "templates/next", Variables: []string{"package_manager"}},
		},
		Presets: []Preset{
			{
				ID:       "fullstack",
				Name:     "Fullstack",
				Default:  true,
				Features: []string{"database:postgres", "auth:jwt"},
			},
			{
				ID:   "minimal",
				Name: "Minimal",
			},
		},
		Features: []Feature{
			{
				ID:    "database",
				Name:  "Database",
				Kinds: []string{"backend", "frontend"},
				Variants: []Variant{
					{
						ID:      "postgres",
						Default: true,
						Overlays: map[string]string{
							"go/chi": "overlays/go/database/postgres",
							"next":   "overlays/next/database/postgres",
						},
					},
				},
			},
			{
				ID:    "auth",
				Name:  "Auth",
				Kinds: []string{"backend", "frontend"},
				Variants: []Variant{
					{
						ID:       "jwt",
						Default:  true,
						Stacks:   []string{"go/chi"},
						Overlays: map[string]string{"go/chi": "overlays/go/auth/jwt"},
					},
					{
						ID:       "session",
						Stacks:   []string{"next"},
						Requires: []string{"database:postgres"},
						Overlays: map[string]string{"next": "overlays/next/auth/session"},
					},
				},
			},
			{
				ID:       "totp",
				Name:     "TOTP",
				Kinds:    []string{"backend", "frontend"},
				Requires: []string{"auth"},
				Overlays: map[string]string{
					"go/chi": "overlays/go/totp",
					"next":   "overlays/next/totp",
				},
			},
		},
	}
}

func TestValidateVariantScopes(t *testing.T) {
	t.Parallel()

	if err := variantManifest().Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*Manifest)
		want   string
	}{
		{
			name: "unknown variant stack",
			mutate: func(m *Manifest) {
				m.Features[1].Variants[1].Stacks = []string{"nope"}
			},
			want: "unknown stack",
		},
		{
			name: "variant stack outside feature scope",
			mutate: func(m *Manifest) {
				m.Features[1].Stacks = []string{"go/chi"}
			},
			want: "not declared",
		},
		{
			name: "variant overlay outside scope",
			mutate: func(m *Manifest) {
				m.Features[1].Variants[1].Overlays["go/chi"] = "overlays/go/auth/session"
			},
			want: "unsupported stack",
		},
		{
			name: "variant missing overlay",
			mutate: func(m *Manifest) {
				m.Features[1].Variants[1].Overlays = map[string]string{}
			},
			want: "overlay required",
		},
		{
			name: "variant unknown requirement",
			mutate: func(m *Manifest) {
				m.Features[1].Variants[1].Requires = []string{"nope"}
			},
			want: "unknown feature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := variantManifest()
			tt.mutate(m)

			err := m.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}
