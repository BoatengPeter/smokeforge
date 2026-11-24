package cli

import (
	"strings"
	"testing"

	"github.com/BoatengPeter/smokeforge/internal/manifest"
	"github.com/BoatengPeter/smokeforge/internal/resolve"
)

func TestValidateWorkspaceSelection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		selection resolve.Selection
		want      string
	}{
		{
			name: "api and desktop",
			selection: resolve.Selection{
				Name:    "Workspace",
				Kind:    "workspace",
				Backend: "go/api",
				Desktop: "electron",
			},
		},
		{
			name: "client only",
			selection: resolve.Selection{
				Name:     "Workspace",
				Kind:     "workspace",
				Frontend: "next",
				Mobiles:  []string{"ios/swiftui"},
			},
		},
		{
			name: "stack flag rejected",
			selection: resolve.Selection{
				Name:  "Workspace",
				Kind:  "workspace",
				Stack: "go/api",
			},
			want: "--stack",
		},
		{
			name: "desktop rejected outside workspace",
			selection: resolve.Selection{
				Name:    "Backend",
				Kind:    "backend",
				Stack:   "go/api",
				Desktop: "electron",
			},
			want: "--desktop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateSelection(&tt.selection)
			if tt.want == "" && err != nil {
				t.Fatalf("error = %v", err)
			}

			if tt.want != "" && (err == nil || !strings.Contains(err.Error(), tt.want)) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestWorkspaceDefaultPreset(t *testing.T) {
	t.Parallel()

	m := &manifest.Manifest{
		Presets: []manifest.Preset{
			{ID: "workspace-api", Kinds: []string{"workspace"}},
			{ID: "fullstack-api", Kinds: []string{"fullstack"}, Default: true},
		},
	}

	withAPI := withDefaultPreset(resolve.Selection{
		Kind:    "workspace",
		Backend: "go/api",
		Desktop: "electron",
	}, m)
	if withAPI.Preset != "workspace-api" {
		t.Fatalf("preset = %q, want workspace-api", withAPI.Preset)
	}

	withoutAPI := withDefaultPreset(resolve.Selection{
		Kind:     "workspace",
		Frontend: "next",
	}, m)
	if withoutAPI.Preset != "" {
		t.Fatalf("preset = %q, want empty", withoutAPI.Preset)
	}
}

func TestWorkspaceNextStepsWithoutAPI(t *testing.T) {
	t.Parallel()

	plan := &resolve.Plan{
		Name: "Workspace",
		Kind: "workspace",
		Targets: []resolve.TargetPlan{
			{Path: "apps/web", Stack: manifest.Stack{ID: "next", Kind: "frontend", Name: "Next.js"}},
			{Path: "apps/android", Stack: manifest.Stack{ID: "android/compose", Kind: "mobile", Name: "Android"}},
		},
		Root: &resolve.TargetPlan{
			Path:  ".",
			Stack: manifest.Stack{ID: "workspace/monorepo", Kind: "workspace"},
		},
	}

	lines := strings.Join(nextStepLines(plan, "/tmp/acme", false), "\n")
	if strings.Contains(lines, "api-dev") {
		t.Fatalf("next steps mention API without an API target:\n%s", lines)
	}

	for _, want := range []string{"make install", "make web-dev", "make mobile-build", "make test"} {
		if !strings.Contains(lines, want) {
			t.Fatalf("next steps missing %q:\n%s", want, lines)
		}
	}
}

func TestWorkspaceSummaryWithoutAPI(t *testing.T) {
	t.Parallel()

	plan := &resolve.Plan{
		Name: "Workspace",
		Kind: "workspace",
		Targets: []resolve.TargetPlan{
			{Path: "apps/web", Stack: manifest.Stack{ID: "next", Name: "Next.js"}},
		},
		Root: &resolve.TargetPlan{Path: ".", Stack: manifest.Stack{ID: "workspace/monorepo"}},
	}

	lines := strings.Join(fullstackSummaryLines(plan, "/tmp/acme", "local"), "\n")
	if strings.Contains(lines, "API") {
		t.Fatalf("summary mentions API without an API target:\n%s", lines)
	}

	if !strings.Contains(lines, "apps/web") {
		t.Fatalf("summary missing web target:\n%s", lines)
	}
}
