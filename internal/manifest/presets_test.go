package manifest

import (
	"strings"
	"testing"
)

func TestDefaultPreset(t *testing.T) {
	t.Parallel()

	m := validManifest()

	p := m.DefaultPreset()
	if p == nil || p.ID != "fullstack" {
		t.Fatalf("default preset = %+v, want fullstack", p)
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
			name: "multiple defaults",
			mutate: func(m *Manifest) {
				m.Presets[1].Default = true
			},
			want: "default preset",
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
