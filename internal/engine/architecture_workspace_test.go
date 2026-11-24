package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BoatengPeter/smokeforge/internal/manifest"
	"github.com/BoatengPeter/smokeforge/internal/resolve"
)

func TestWriteArchitectureForWorkspace(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	for _, dir := range []string{".agents/skills/workspace-orchestration", ".claude/skills/workspace-orchestration", "docs", "apps/api", "apps/web", "apps/android"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o750); err != nil {
			t.Fatal(err)
		}
	}

	for _, file := range []string{"AGENTS.md", "CLAUDE.md", "CONTEXT.md", "CONTEXT-MAP.md", "Makefile", "skills-lock.json", "docs/CLIENTS.md"} {
		if err := os.WriteFile(filepath.Join(root, file), []byte("# test\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	api := resolve.TargetPlan{Path: "apps/api", Stack: manifest.Stack{ID: "go/api", Kind: "backend"}}
	plan := &resolve.Plan{
		Kind: "workspace",
		API:  &api,
		Targets: []resolve.TargetPlan{
			api,
			{Path: "apps/web", Stack: manifest.Stack{ID: "next", Kind: "frontend"}},
			{Path: "apps/android", Stack: manifest.Stack{ID: "android/compose", Kind: "mobile"}},
		},
		Root: &resolve.TargetPlan{Path: ".", Stack: manifest.Stack{ID: "workspace/monorepo", Kind: "workspace"}},
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
	for _, want := range []string{"AGENTS.md", "CLAUDE.md", "skills-lock.json", ".agents/", "apps/api", "apps/web", "apps/android", "Workspace boundaries", "Shared skills"} {
		if !strings.Contains(content, want) {
			t.Fatalf("architecture missing %q:\n%s", want, content)
		}
	}
}

func TestWriteArchitectureForAPIlessWorkspace(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	for _, dir := range []string{"docs", "apps/web", "apps/ios"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o750); err != nil {
			t.Fatal(err)
		}
	}

	plan := &resolve.Plan{
		Kind: "workspace",
		Targets: []resolve.TargetPlan{
			{Path: "apps/web", Stack: manifest.Stack{ID: "next", Kind: "frontend"}},
			{Path: "apps/ios", Stack: manifest.Stack{ID: "ios/swiftui", Kind: "mobile"}},
		},
		Root: &resolve.TargetPlan{Path: ".", Stack: manifest.Stack{ID: "workspace/monorepo", Kind: "workspace"}},
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
	if strings.Contains(content, "API service") {
		t.Fatalf("API-less architecture mentions API service:\n%s", content)
	}

	for _, want := range []string{"apps/web", "apps/ios", "Next.js app", "iOS app"} {
		if !strings.Contains(content, want) {
			t.Fatalf("architecture missing %q:\n%s", want, content)
		}
	}
}
