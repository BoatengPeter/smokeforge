package tui

import (
	"strings"
	"testing"

	"github.com/BoatengPeter/smokeforge/internal/manifest"
	"github.com/charmbracelet/huh"
)

func TestFilterFeatureValues(t *testing.T) {
	t.Parallel()

	options := []huh.Option[string]{
		huh.NewOption("Auth", "auth"),
		huh.NewOption("Database", "database:postgres"),
	}

	got := filterFeatureValues(
		[]string{"auth", "missing", "database:postgres"},
		options,
	)

	if strings.Join(got, ",") != "auth,database:postgres" {
		t.Fatalf("filtered = %v", got)
	}
}

func TestWizardFullstackFeatureOptions(t *testing.T) {
	t.Parallel()

	m := &manifest.Manifest{
		Features: []manifest.Feature{
			{
				ID:    "auth",
				Kinds: []string{"backend", "frontend"},
				Variants: []manifest.Variant{
					{ID: "jwt", Overlays: map[string]string{"go/api": "auth/jwt"}},
					{ID: "session", Overlays: map[string]string{"next": "auth/session"}},
				},
			},
			{
				ID:    "data",
				Kinds: []string{"frontend"},
				Variants: []manifest.Variant{
					{ID: "tanstack-query", Overlays: map[string]string{"next": "data/query"}},
				},
			},
		},
		Stacks: []manifest.Stack{
			{ID: "go/api", Kind: "backend"},
			{ID: "next", Kind: "frontend"},
		},
	}

	options := wizardFeatureOptions(m, fullstackKind, manifest.Stack{ID: "go/api", Kind: "backend"})

	values := make(map[string]bool, len(options))
	for _, option := range options {
		values[option.Value] = true
	}

	for _, want := range []string{"auth:jwt", "data:tanstack-query"} {
		if !values[want] {
			t.Fatalf("fullstack options %v missing %q", values, want)
		}
	}

	if values["auth:session"] {
		t.Fatal("fullstack options must not include frontend-owned session auth")
	}
}
