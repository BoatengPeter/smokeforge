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

func TestValidateFeatureVariantsRejectsTwoVariants(t *testing.T) {
	t.Parallel()

	err := validateFeatureVariants([]string{"auth:electron", "auth:oauth-pkce"})
	if err == nil || !strings.Contains(err.Error(), "choose one variant") {
		t.Fatalf("error = %v, want one-variant guidance", err)
	}

	if err := validateFeatureVariants([]string{"auth:electron", "database:postgres"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPresetsForStackFiltersUnsupportedFeatures(t *testing.T) {
	t.Parallel()

	m := &manifest.Manifest{
		Stacks: []manifest.Stack{
			{ID: "desktop/electron", Kind: "desktop"},
			{ID: "desktop/egui", Kind: "desktop"},
		},
		Features: []manifest.Feature{{
			ID:    "storage",
			Kinds: []string{"desktop"},
			Variants: []manifest.Variant{{
				ID:       "sqlite",
				Default:  true,
				Overlays: map[string]string{"desktop/electron": "overlays/storage"},
			}},
		}},
		Presets: []manifest.Preset{
			{ID: "desktop", Kinds: []string{"desktop"}, Default: true, Features: []string{"storage:sqlite"}},
			{ID: "desktop-egui", Kinds: []string{"desktop"}},
		},
	}

	got := presetsForStack(m, "desktop", manifest.Stack{ID: "desktop/egui", Kind: "desktop"})
	if len(got) != 1 || got[0].ID != "desktop-egui" {
		t.Fatalf("egui presets = %+v, want desktop-egui only", got)
	}
}

func workspaceWizardManifest() *manifest.Manifest {
	return &manifest.Manifest{
		Kinds: map[string]manifest.Kind{
			"backend":   {Max: 1},
			"frontend":  {Max: 1},
			"desktop":   {Max: 1},
			"mobile":    {Max: 2},
			"workspace": {Max: 1},
		},
		Stacks: []manifest.Stack{
			{ID: "go/api", Kind: "backend", Name: "Go API"},
			{ID: "next", Kind: "frontend", Name: "Next.js"},
			{ID: "android/compose", Kind: "mobile", Name: "Android"},
			{ID: "ios/swiftui", Kind: "mobile", Name: "iOS"},
			{ID: "desktop/electron", Kind: "desktop", Name: "Electron"},
			{ID: "workspace/monorepo", Kind: "workspace", Name: "Composable workspace"},
		},
		Presets: []manifest.Preset{{ID: "workspace-api", Kinds: []string{"workspace"}}},
	}
}

func TestWorkspaceWizardResultUsesTargetsWithoutStack(t *testing.T) {
	t.Parallel()

	wizard := &Wizard{Manifest: workspaceWizardManifest()}

	result, err := wizard.workspaceResult(
		[]string{"api", "web", "android", "ios"},
		"go/api",
		"",
		"Acme Workspace",
		"",
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("workspace result: %v", err)
	}

	if result.Kind != workspaceKind || result.Stack != "" {
		t.Fatalf("result = %+v, want workspace kind without stack", result)
	}

	if result.Backend != "go/api" || result.Frontend != "next" {
		t.Fatalf("result = %+v", result)
	}

	if strings.Join(result.Mobiles, ",") != "android/compose,ios/swiftui" {
		t.Fatalf("mobiles = %v", result.Mobiles)
	}

	if result.Preset != "workspace-api" {
		t.Fatalf("preset = %q, want workspace-api", result.Preset)
	}
}

func TestWorkspaceWizardRejectsNoTargets(t *testing.T) {
	t.Parallel()

	wizard := &Wizard{Manifest: workspaceWizardManifest()}
	if _, err := wizard.workspaceResult(nil, "", "", "Acme", "", nil, nil); err == nil {
		t.Fatal("expected an empty workspace target error")
	}
}
