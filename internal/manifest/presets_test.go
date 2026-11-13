package manifest

import (
	"strings"
	"testing"
)

func TestDefaultPreset(t *testing.T) {
	t.Parallel()

	m := validManifest()

	p := m.DefaultPresetForKind("backend")
	if p == nil || p.ID != "fullstack" {
		t.Fatalf("default preset = %+v, want fullstack", p)
	}
}

func TestDefaultPresetForKind(t *testing.T) {
	t.Parallel()

	m := validManifest()
	m.Kinds["frontend"] = Kind{Max: 1}
	m.Stacks = append(m.Stacks, Stack{
		ID:        "next",
		Kind:      "frontend",
		Name:      "Next",
		Template:  "templates/next",
		Variables: []string{"package_manager"},
	})
	m.Presets = []Preset{
		{ID: "api", Name: "API", Default: true, Kinds: []string{"backend"}},
		{ID: "web", Name: "Web", Default: true, Kinds: []string{"frontend"}},
	}

	if p := m.DefaultPresetForKind("backend"); p == nil || p.ID != "api" {
		t.Fatalf("backend default preset = %+v, want api", p)
	}

	if p := m.DefaultPresetForKind("frontend"); p == nil || p.ID != "web" {
		t.Fatalf("frontend default preset = %+v, want web", p)
	}

	if p := m.DefaultPresetForKind("mobile"); p != nil {
		t.Fatalf("mobile default preset = %+v, want nil", p)
	}
}

func TestPresetAppliesToKind(t *testing.T) {
	t.Parallel()

	all := Preset{ID: "all"}
	backend := Preset{ID: "backend", Kinds: []string{"backend"}}

	if !all.AppliesToKind("backend") {
		t.Fatal("preset with no kinds should apply to every kind")
	}

	if !backend.AppliesToKind("backend") {
		t.Fatal("backend preset should apply to backend")
	}

	if backend.AppliesToKind("frontend") {
		t.Fatal("backend preset should not apply to frontend")
	}
}

func TestFindPreset(t *testing.T) {
	t.Parallel()

	m := validManifest()

	if p := m.FindPreset("minimal"); p == nil || p.ID != "minimal" {
		t.Fatalf("minimal preset = %+v", p)
	}

	if p := m.FindPreset("missing"); p != nil {
		t.Fatalf("missing preset = %+v, want nil", p)
	}
}

func TestValidatePresets(t *testing.T) {
	t.Parallel()

	runValidationCases(t, []validationCase{
		{
			name: "duplicate preset id",
			mutate: func(m *Manifest) {
				m.Presets = append(m.Presets, Preset{ID: "fullstack", Name: "Duplicate"})
			},
			want: "duplicate preset",
		},
		{
			name: "multiple defaults for one kind",
			mutate: func(m *Manifest) {
				m.Presets[1].Default = true
			},
			want: "default preset",
		},
		{
			name: "unknown preset kind",
			mutate: func(m *Manifest) {
				m.Presets[0].Kinds = []string{"mobile"}
			},
			want: "unknown kind",
		},
		{
			name: "unknown feature",
			mutate: func(m *Manifest) {
				m.Presets[0].Features = []string{"missing"}
			},
			want: "unknown feature",
		},
		{
			name: "unknown variant",
			mutate: func(m *Manifest) {
				m.Presets[0].Features = []string{"auth:missing"}
			},
			want: "unknown variant",
		},
		{
			name: "duplicate feature",
			mutate: func(m *Manifest) {
				m.Presets[0].Features = []string{"auth:jwt", "auth:jwt"}
			},
			want: "duplicate",
		},
		{
			name: "empty preset id",
			mutate: func(m *Manifest) {
				m.Presets[0].ID = ""
			},
			want: "preset id",
		},
	})
}

func TestValidatePresetsAllowsEmptyFeatures(t *testing.T) {
	t.Parallel()

	m := validManifest()
	m.Presets = []Preset{{ID: "frontend", Name: "Frontend", Default: true}}

	if err := m.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestValidatePresetUnknownFeatureMessage(t *testing.T) {
	t.Parallel()

	m := validManifest()
	m.Presets[0].Features = []string{"missing"}

	err := m.Validate()
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("error = %v, want missing feature", err)
	}
}
