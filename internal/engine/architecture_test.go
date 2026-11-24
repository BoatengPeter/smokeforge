package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BoatengPeter/smokeforge/internal/manifest"
	"github.com/BoatengPeter/smokeforge/internal/patch"
	"github.com/BoatengPeter/smokeforge/internal/resolve"
)

func TestWriteArchitectureUsesCapabilitiesNotInternalFeatureIDs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	for _, dir := range []string{"apps/api", "apps/web", "apps/desktop/crates/app", "apps/desktop/crates/storage", "apps/desktop/crates/desktop", "apps/desktop/docs"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o750); err != nil {
			t.Fatal(err)
		}
	}

	for _, file := range []string{"apps/api/README.md", "apps/web/README.md", "apps/desktop/README.md", "apps/desktop/crates/app", "apps/desktop/crates/storage", "apps/desktop/crates/desktop", "apps/desktop/docs"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, filepath.FromSlash(file))), 0o750); err != nil {
			t.Fatal(err)
		}
	}

	dependency := patch.Dependency{File: "Cargo.toml", Module: "redis", Version: "0.32"}
	plan := &resolve.Plan{
		Kind: "fullstack",
		Targets: []resolve.TargetPlan{
			{Path: "apps/api", Stack: manifest.Stack{Kind: "backend"}},
			{Path: "apps/web", Stack: manifest.Stack{Kind: "frontend"}},
			{Path: "apps/desktop", Stack: manifest.Stack{Kind: "desktop"}},
		},
		Root: &resolve.TargetPlan{Path: "."},
	}
	plan.Targets[0].Overlays = []resolve.OverlayPlan{{
		FeatureID: "cache",
		VariantID: "redis",
		Overlay:   &patch.Overlay{ID: "cache:redis", Ops: []patch.Op{{AddDependency: &dependency}}},
	}}
	m := &manifest.Manifest{Features: []manifest.Feature{{
		ID: "cache", Name: "Cache",
		Variants: []manifest.Variant{{ID: "redis", Name: "Redis"}},
	}}}

	if err := writeArchitecture(root, plan, m); err != nil {
		t.Fatal(err)
	}

	//nolint:gosec // path is constrained to the test temporary directory.
	data, err := os.ReadFile(filepath.Join(root, architectureFile))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	for _, want := range []string{"Cache — Redis", "redis", "apps/desktop", "Browser"} {
		if !strings.Contains(content, want) {
			t.Fatalf("architecture missing %q:\n%s", want, content)
		}
	}

	for _, forbidden := range []string{"cache:redis", "desktop/egui", "sync:library-client"} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("architecture leaked %q:\n%s", forbidden, content)
		}
	}
}

func TestWriteArchitectureIncludesMobileTarget(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "apps", "android", "app"), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(root, "apps", "android", "README.md"), []byte("# Android\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	plan := &resolve.Plan{
		Kind: "fullstack",
		Targets: []resolve.TargetPlan{
			{Path: "apps/api", Stack: manifest.Stack{Kind: "backend"}},
			{Path: "apps/web", Stack: manifest.Stack{Kind: "frontend"}},
			{Path: "apps/android", Stack: manifest.Stack{ID: "android/compose", Kind: "mobile"}},
		},
		Root: &resolve.TargetPlan{Path: "."},
	}

	if err := writeArchitecture(root, plan, &manifest.Manifest{}); err != nil {
		t.Fatal(err)
	}

	//nolint:gosec // path is constrained to the test temporary directory.
	data, err := os.ReadFile(filepath.Join(root, architectureFile))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	for _, want := range []string{"apps/android", "Android app", "Android app] -->|authenticated API|"} {
		if !strings.Contains(content, want) {
			t.Fatalf("architecture missing %q:\n%s", want, content)
		}
	}
}

func TestWriteArchitectureUsesIOSLayout(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	for _, dir := range []string{"App", "Packages/Core", "Config", "docs"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o750); err != nil {
			t.Fatal(err)
		}
	}

	for _, file := range []string{"project.yml", "Makefile", "README.md", "CONTEXT.md", "DESIGN.md", "skills-lock.json"} {
		if err := os.WriteFile(filepath.Join(root, file), []byte("test"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	plan := &resolve.Plan{
		Kind: "mobile",
		Targets: []resolve.TargetPlan{
			{Path: ".", Stack: manifest.Stack{ID: iosStackID, Kind: "mobile"}},
		},
	}

	if err := writeArchitecture(root, plan, &manifest.Manifest{}); err != nil {
		t.Fatal(err)
	}

	//nolint:gosec // path is constrained to the test temporary directory.
	data, err := os.ReadFile(filepath.Join(root, architectureFile))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	for _, want := range []string{"project.yml", "App/", "Packages/", "Screens and navigation", "Xcode project and configs"} {
		if !strings.Contains(content, want) {
			t.Fatalf("architecture missing %q:\n%s", want, content)
		}
	}

	for _, forbidden := range []string{"App/Sources/", "UITests/", "Secure local settings"} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("architecture contains stale iOS entry %q:\n%s", forbidden, content)
		}
	}
}
