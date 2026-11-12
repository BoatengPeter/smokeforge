//go:build integration

package engine

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BoatengPeter/smokeforge/internal/project"
	"github.com/BoatengPeter/smokeforge/internal/registry"
	"github.com/BoatengPeter/smokeforge/internal/resolve"
)

func TestGenerateIntegration(t *testing.T) {
	t.Parallel()

	registryDir, err := filepath.Abs(filepath.Join("..", "..", "testdata", "registry"))
	if err != nil {
		t.Fatal(err)
	}

	reg, err := registry.Load(t.Context(), registry.Options{Source: registryDir})
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}

	plan, err := resolve.Resolve(reg.Manifest, reg.Dir, resolve.Selection{
		Name:      "Integration App",
		Kind:      "backend",
		Stack:     "test/plain",
		Features:  []string{"logging"},
		Variables: map[string]string{"module_path": "example.com/integration"},
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	target := filepath.Join(t.TempDir(), "service")
	if _, err := Generate(t.Context(), Request{
		Registry: reg,
		Plan:     plan,
		Target:   target,
		NoGit:    true,
	}); err != nil {
		t.Fatalf("generate: %v", err)
	}

	mainContent, err := os.ReadFile(filepath.Join(target, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"smokeforge:begin logging:basic",
		"smokeforge:end logging:basic-use",
	} {
		if !strings.Contains(string(mainContent), want) {
			t.Fatalf("main.go missing %q:\n%s", want, mainContent)
		}
	}

	state, err := project.LoadState(target)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state.Applied[0].Files["main.go"] == "" {
		t.Fatalf("state missing main.go hash: %+v", state.Applied)
	}

	buildGeneratedProject(t, target)

	if _, err := Add(t.Context(), AddRequest{
		ProjectDir: target,
		Registry:   reg,
		Features:   []string{"metrics"},
	}); err != nil {
		t.Fatalf("add metrics: %v", err)
	}

	buildGeneratedProject(t, target)

	if _, err := Remove(t.Context(), RemoveRequest{
		ProjectDir: target,
		Registry:   reg,
		Features:   []string{"metrics"},
	}); err != nil {
		t.Fatalf("remove metrics: %v", err)
	}

	if _, err := os.Stat(filepath.Join(target, "internal", "metrics", "metrics.go")); !os.IsNotExist(err) {
		t.Fatalf("metrics file survived removal: %v", err)
	}

	buildGeneratedProject(t, target)
}

func buildGeneratedProject(t *testing.T, target string) {
	t.Helper()

	command := exec.CommandContext(t.Context(), "go", "build", "./...")
	command.Dir = target

	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go build: %v\n%s", err, output)
	}
}
