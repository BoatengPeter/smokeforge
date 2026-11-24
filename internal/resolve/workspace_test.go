package resolve

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/BoatengPeter/smokeforge/internal/manifest"
)

func workspaceManifest() *manifest.Manifest {
	m := fullstackManifest()
	m.Kinds["workspace"] = manifest.Kind{Max: 1}
	m.Stacks = append(m.Stacks, manifest.Stack{
		ID:        "workspace/monorepo",
		Kind:      "workspace",
		Name:      "Composable workspace",
		Template:  "templates/root-workspace",
		Variables: []string{"package_manager"},
	})
	m.Presets = append(m.Presets, manifest.Preset{
		ID:       "workspace-api",
		Name:     "Workspace API",
		Kinds:    []string{"workspace"},
		Features: []string{"database:postgres", "auth:jwt", "ci:github"},
	})

	for index := range m.Features {
		if m.Features[index].ID != "ci" {
			continue
		}

		m.Features[index].Kinds = append(m.Features[index].Kinds, "workspace")
		m.Features[index].Variants[0].Overlays["workspace/monorepo"] = "overlays/workspace/monorepo/ci"
	}

	return m
}

func writeWorkspaceRegistry(t *testing.T) string {
	t.Helper()

	dir := writeFullstackRegistry(t)
	writeFile(t, filepath.Join(dir, "templates", "root-workspace", "template.yaml"),
		"id: workspace/monorepo\nvariables:\n  - name: package_manager\n    default: pnpm\n")

	for _, path := range []string{
		"overlays/workspace/monorepo/api",
		"overlays/workspace/monorepo/web",
		"overlays/workspace/monorepo/mobile",
		"overlays/workspace/monorepo/desktop",
		"overlays/workspace/monorepo/desktop-egui",
		"overlays/workspace/monorepo/desktop-gpui",
		"overlays/workspace/monorepo/ci",
	} {
		writeFile(t, filepath.Join(dir, path, "overlay.yaml"),
			"id: "+strings.ReplaceAll(path, "overlays/", "")+"\nops:\n  - env:\n      key: TEST_FLAG\n      example: \"1\"\n")
	}

	return dir
}

func TestResolveWorkspaceAllClients(t *testing.T) {
	t.Parallel()

	plan, err := Resolve(workspaceManifest(), writeWorkspaceRegistry(t), Selection{
		Name:     "Acme Workspace",
		Kind:     "workspace",
		Backend:  "go/api",
		Frontend: "next",
		Desktop:  "electron",
		Mobiles:  []string{"android/compose", "ios/swiftui"},
		Preset:   "workspace-api",
		Variables: map[string]string{
			"module_path":      "example.com/acme",
			"package_manager":  "pnpm",
			"application_id":   "com.example.acme",
			"package_name":     "com.example.acme",
			"api_base_url":     "http://10.0.2.2:8080",
			"bundle_id":        "com.example.acme",
			"ios_api_base_url": "http://127.0.0.1:8080",
			"api_url":          "http://localhost:8080",
		},
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if plan.Kind != "workspace" || plan.API == nil || plan.Root == nil {
		t.Fatalf("plan = %+v", plan)
	}

	paths := make([]string, 0, len(plan.Targets))
	for _, target := range plan.Targets {
		paths = append(paths, target.Path)
	}

	if got, want := strings.Join(paths, ","), "apps/api,apps/web,apps/android,apps/ios,apps/desktop"; got != want {
		t.Fatalf("targets = %q, want %q", got, want)
	}

	if plan.Root.Stack.ID != "workspace/monorepo" {
		t.Fatalf("root = %+v", plan.Root)
	}

	if got := overlayIDs(plan.Root.Overlays); strings.Join(got, ",") != "ci:github,api,web,mobile,desktop" {
		t.Fatalf("root overlays = %v", got)
	}
}

func TestResolveWorkspaceDesktopWithoutWeb(t *testing.T) {
	t.Parallel()

	plan, err := Resolve(workspaceManifest(), writeWorkspaceRegistry(t), Selection{
		Name:    "Acme Desktop",
		Kind:    "workspace",
		Backend: "go/api",
		Desktop: "egui",
		Preset:  "workspace-api",
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

	if len(plan.Targets) != 2 || plan.Targets[1].Path != "apps/desktop" {
		t.Fatalf("targets = %+v", plan.Targets)
	}

	if workspaceTargetByPath(plan, "apps/web") != nil {
		t.Fatal("web target should not exist")
	}

	if got := overlayIDs(plan.Root.Overlays); strings.Join(got, ",") != "ci:github,api,desktop-egui" {
		t.Fatalf("root overlays = %v", got)
	}
}

func TestResolveWorkspaceClientOnly(t *testing.T) {
	t.Parallel()

	plan, err := Resolve(workspaceManifest(), writeWorkspaceRegistry(t), Selection{
		Name:     "Acme Clients",
		Kind:     "workspace",
		Frontend: "next",
		Desktop:  "electron",
		Mobiles:  []string{"android/compose"},
		Variables: map[string]string{
			"package_manager": "pnpm",
			"bundle_id":       "com.example.acme",
			"api_url":         "http://localhost:8080",
		},
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if plan.API != nil || plan.Target.Stack.ID != "" {
		t.Fatalf("api = %+v, target = %+v", plan.API, plan.Target)
	}

	if got, want := len(plan.Targets), 3; got != want {
		t.Fatalf("targets = %d, want %d", got, want)
	}

	if got := overlayIDs(plan.Root.Overlays); strings.Join(got, ",") != "web,mobile,desktop" {
		t.Fatalf("root overlays = %v", got)
	}
}

func TestResolveWorkspaceRejectsInvalidShapes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		selection Selection
		want      string
	}{
		{
			name:      "no targets",
			selection: Selection{Name: "Empty", Kind: "workspace"},
			want:      "at least one target",
		},
		{
			name:      "stack flag",
			selection: Selection{Name: "Stack", Kind: "workspace", Stack: "go/api"},
			want:      "--stack",
		},
		{
			name:      "duplicate mobile",
			selection: Selection{Name: "Duplicate", Kind: "workspace", Mobiles: []string{"ios/swiftui", "ios/swiftui"}},
			want:      "duplicate mobile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := Resolve(workspaceManifest(), writeWorkspaceRegistry(t), tt.selection)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestResolveWorkspaceMobileRequiresAPIJWT(t *testing.T) {
	t.Parallel()

	m := workspaceManifest()
	for index := range m.Presets {
		if m.Presets[index].ID == "workspace-api" {
			m.Presets[index].Features = []string{"database:postgres"}
		}
	}

	_, err := Resolve(m, writeWorkspaceRegistry(t), Selection{
		Name:    "Acme Mobile",
		Kind:    "workspace",
		Backend: "go/api",
		Mobiles: []string{"android/compose"},
		Preset:  "workspace-api",
		Variables: map[string]string{
			"module_path":     "example.com/acme",
			"package_manager": "pnpm",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "requires auth:jwt") {
		t.Fatalf("error = %v, want auth requirement", err)
	}
}

func workspaceTargetByPath(plan *Plan, targetPath string) *TargetPlan {
	for index := range plan.Targets {
		if plan.Targets[index].Path == targetPath {
			return &plan.Targets[index]
		}
	}

	return nil
}
