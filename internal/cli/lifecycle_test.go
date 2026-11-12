package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BoatengPeter/smokeforge/internal/project"
)

func createProject(t *testing.T, registryDir string, features ...string) string {
	t.Helper()

	target := filepath.Join(t.TempDir(), "app")

	args := []string{
		"new", target,
		"--registry", registryDir,
		"--kind", "backend",
		"--stack", "test/plain",
		"--name", "Acme App",
		"--preset", "frontend",
		"--no-input",
		"--no-git",
	}
	for _, feature := range features {
		args = append(args, "--feature", feature)
	}

	if _, _, err := executeCommand(t, args...); err != nil {
		t.Fatalf("new: %v", err)
	}

	return target
}

func TestAddRemoveCommands(t *testing.T) {
	t.Parallel()

	registryDir := writeGenerationRegistry(t)
	target := createProject(t, registryDir)

	out, _, err := executeCommand(
		t,
		"add", "metrics",
		"--project-dir", target,
		"--registry", registryDir,
	)
	if err != nil {
		t.Fatalf("add: %v", err)
	}

	if !strings.Contains(out, "Features added") {
		t.Fatalf("output %q missing add summary", out)
	}

	if _, statErr := os.Stat(filepath.Join(target, "internal", "metrics", "metrics.go")); statErr != nil {
		t.Fatalf("metrics file missing: %v", statErr)
	}

	out, _, err = executeCommand(
		t,
		"remove", "metrics",
		"--project-dir", target,
		"--registry", registryDir,
	)
	if err != nil {
		t.Fatalf("remove: %v", err)
	}

	if !strings.Contains(out, "Features removed") {
		t.Fatalf("output %q missing remove summary", out)
	}

	if _, statErr := os.Stat(filepath.Join(target, "internal", "metrics", "metrics.go")); !os.IsNotExist(statErr) {
		t.Fatalf("metrics file still exists: %v", statErr)
	}

	state, err := project.LoadState(target)
	if err != nil {
		t.Fatal(err)
	}

	if len(state.Applied) != 0 {
		t.Fatalf("applied = %+v", state.Applied)
	}
}

func TestAddForceCommand(t *testing.T) {
	t.Parallel()

	registryDir := writeGenerationRegistry(t)
	target := createProject(t, registryDir, "logging")

	mainPath := filepath.Join(target, "main.go")

	content := mustReadFile(t, mainPath)
	if err := os.WriteFile(mainPath, append(content, []byte("// user edit\n")...), 0o600); err != nil {
		t.Fatal(err)
	}

	_, _, err := executeCommand(
		t,
		"add", "metrics",
		"--project-dir", target,
		"--registry", registryDir,
	)
	if err == nil || !strings.Contains(err.Error(), "drifted") {
		t.Fatalf("error = %v, want drift error", err)
	}

	_, _, err = executeCommand(
		t,
		"add", "metrics",
		"--project-dir", target,
		"--registry", registryDir,
		"--force",
	)
	if err != nil {
		t.Fatalf("force add: %v", err)
	}
}

//nolint:paralleltest // uses t.Chdir to exercise project detection.
func TestDoctorProjectState(t *testing.T) {
	registryDir := writeGenerationRegistry(t)
	target := createProject(t, registryDir, "logging")

	t.Chdir(target)

	out, _, err := executeCommand(
		t,
		"doctor",
		"--registry", registryDir,
		"--cache-dir", t.TempDir(),
	)
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}

	if !strings.Contains(out, "project: ok") {
		t.Fatalf("output %q missing project ok", out)
	}

	mainPath := filepath.Join(target, "main.go")

	content := mustReadFile(t, mainPath)
	if err := os.WriteFile(mainPath, append(content, []byte("// edit\n")...), 0o600); err != nil {
		t.Fatal(err)
	}

	out, _, err = executeCommand(
		t,
		"doctor",
		"--registry", registryDir,
		"--cache-dir", t.TempDir(),
	)
	if err == nil {
		t.Fatal("expected doctor failure after drift")
	}

	if !strings.Contains(out, "project: fail") {
		t.Fatalf("output %q missing project failure", out)
	}
}
