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
		Features: []Feature{
			{
				ID:    "database",
				Name:  "Database",
				Kinds: []string{"backend"},
				Variants: []Variant{
					{ID: "postgres", Name: "PostgreSQL", Default: true},
				},
			},
			{
				ID:       "auth",
				Name:     "Authentication",
				Kinds:    []string{"backend"},
				Variants: []Variant{{ID: "jwt", Name: "JWT", Default: true}},
			},
			{
				ID:       "totp",
				Name:     "TOTP",
				Kinds:    []string{"backend"},
				Requires: []string{"auth:jwt"},
			},
		},
	}
}

func TestValidateAcceptsValidManifest(t *testing.T) {
	if err := validManifest().Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestValidateRejects(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Manifest)
		want   string
	}{
		{
			name:   "schema version",
			mutate: func(m *Manifest) { m.SchemaVersion = 2 },
			want:   "schemaVersion",
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
		{
			name: "duplicate variant",
			mutate: func(m *Manifest) {
				m.Features[0].Variants = append(m.Features[0].Variants, m.Features[0].Variants[0])
			},
			want: "duplicate variant",
		},
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
		{
			name: "two default variants",
			mutate: func(m *Manifest) {
				m.Features[0].Variants = append(
					m.Features[0].Variants,
					Variant{ID: "sqlite", Name: "SQLite", Default: true},
				)
			},
			want: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
