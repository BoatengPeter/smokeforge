package engine

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BoatengPeter/smokeforge/internal/project"
	"github.com/BoatengPeter/smokeforge/internal/registry"
	"github.com/BoatengPeter/smokeforge/internal/resolve"
)

const metricsFeature = "metrics"

func projectWithFeatures(t *testing.T, features []string) (string, registry.Result) {
	t.Helper()

	reg, _ := fixture(t)

	plan, err := resolve.Resolve(reg.Manifest, reg.Dir, resolve.Selection{
		Name:      "Acme App",
		Kind:      "backend",
		Stack:     "test/plain",
		Features:  features,
		Variables: map[string]string{"module_path": "example.com/acme"},
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	target := filepath.Join(t.TempDir(), "app")
	if _, err := Generate(t.Context(), Request{
		Registry: &reg,
		Plan:     plan,
		Target:   target,
		NoGit:    true,
	}); err != nil {
		t.Fatalf("generate: %v", err)
	}

	return target, reg
}

func addMetrics(t *testing.T) (string, registry.Result, *MutationResult) {
	t.Helper()

	target, reg := projectWithFeatures(t, nil)

	result, err := Add(t.Context(), AddRequest{
		ProjectDir: target,
		Registry:   &reg,
		Features:   []string{metricsFeature},
	})
	if err != nil {
		t.Fatalf("add: %v", err)
	}

	return target, reg, result
}

func TestAddFeature(t *testing.T) {
	t.Parallel()

	target, _, result := addMetrics(t)

	if len(result.Added) != 1 || result.Added[0] != metricsFeature {
		t.Fatalf("added = %v", result.Added)
	}

	metrics := mustReadFile(t, filepath.Join(target, "internal", metricsFeature, "metrics.go"))
	if !strings.Contains(string(metrics), "acme") {
		t.Fatalf("metrics file not rendered:\n%s", metrics)
	}

	envFile := mustReadFile(t, filepath.Join(target, ".env.example"))
	if !strings.Contains(string(envFile), "METRICS_ENABLED=1") {
		t.Fatalf(".env.example missing metrics key:\n%s", envFile)
	}
}

func TestAddFeatureState(t *testing.T) {
	t.Parallel()

	target, _, _ := addMetrics(t)

	state, err := project.LoadState(target)
	if err != nil {
		t.Fatal(err)
	}

	if len(state.Applied) != 1 || state.Applied[0].Overlay != metricsFeature {
		t.Fatalf("applied = %+v", state.Applied)
	}

	if len(state.Applied[0].Env) != 1 || state.Applied[0].Env[0] != "METRICS_ENABLED" {
		t.Fatalf("env keys = %+v", state.Applied[0].Env)
	}

	if len(state.Targets[0].Features) != 1 || state.Targets[0].Features[0] != metricsFeature {
		t.Fatalf("features = %+v", state.Targets[0].Features)
	}

	lock, err := project.LoadLock(target)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := lock.Overlays["overlays/metrics/basic"]; !ok {
		t.Fatalf("lock overlays = %+v", lock.Overlays)
	}

	drift, err := project.CheckDrift(target, state)
	if err != nil {
		t.Fatal(err)
	}

	if len(drift) != 0 {
		t.Fatalf("drift after add = %+v", drift)
	}
}

func TestAddRejectsAlreadyAppliedFeature(t *testing.T) {
	t.Parallel()

	target, reg := projectWithFeatures(t, []string{metricsFeature})

	_, err := Add(t.Context(), AddRequest{
		ProjectDir: target,
		Registry:   &reg,
		Features:   []string{metricsFeature},
	})
	if err == nil || !strings.Contains(err.Error(), "already") {
		t.Fatalf("error = %v, want already-applied error", err)
	}
}

func TestRemoveFeature(t *testing.T) {
	t.Parallel()

	target, reg := projectWithFeatures(t, []string{metricsFeature})

	result, err := Remove(t.Context(), RemoveRequest{
		ProjectDir: target,
		Registry:   &reg,
		Features:   []string{metricsFeature},
	})
	if err != nil {
		t.Fatalf("remove: %v", err)
	}

	if len(result.Removed) != 1 || result.Removed[0] != metricsFeature {
		t.Fatalf("removed = %v", result.Removed)
	}

	if _, err := os.Stat(filepath.Join(target, "internal", metricsFeature, "metrics.go")); !os.IsNotExist(err) {
		t.Fatalf("metrics file still exists: %v", err)
	}

	envFile := mustReadFile(t, filepath.Join(target, ".env.example"))
	if strings.Contains(string(envFile), "METRICS_ENABLED") {
		t.Fatalf(".env.example still has metrics key:\n%s", envFile)
	}

	state, err := project.LoadState(target)
	if err != nil {
		t.Fatal(err)
	}

	if len(state.Applied) != 0 || len(state.Targets[0].Features) != 0 {
		t.Fatalf("state not cleaned: %+v", state)
	}

	lock, err := project.LoadLock(target)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := lock.Overlays["overlays/metrics/basic"]; ok {
		t.Fatalf("lock still has metrics overlay: %+v", lock.Overlays)
	}
}

func TestRemoveMarkerFeature(t *testing.T) {
	t.Parallel()

	target, reg := projectWithFeatures(t, []string{"logging"})

	if _, err := Remove(t.Context(), RemoveRequest{
		ProjectDir: target,
		Registry:   &reg,
		Features:   []string{"logging"},
	}); err != nil {
		t.Fatalf("remove: %v", err)
	}

	mainContent := mustReadFile(t, filepath.Join(target, "main.go"))
	if strings.Contains(string(mainContent), "smokeforge:begin logging") {
		t.Fatalf("markers remain:\n%s", mainContent)
	}

	if _, err := os.Stat(filepath.Join(target, "internal", "log", "logger.go")); !os.IsNotExist(err) {
		t.Fatalf("logger file still exists: %v", err)
	}
}

func TestRemoveRejectsNonAppliedFeature(t *testing.T) {
	t.Parallel()

	target, reg := projectWithFeatures(t, nil)

	_, err := Remove(t.Context(), RemoveRequest{
		ProjectDir: target,
		Registry:   &reg,
		Features:   []string{metricsFeature},
	})
	if err == nil || !strings.Contains(err.Error(), "not applied") {
		t.Fatalf("error = %v, want not-applied error", err)
	}
}

func TestAddRejectsDrift(t *testing.T) {
	t.Parallel()

	target, reg := projectWithFeatures(t, []string{"logging"})

	mainPath := filepath.Join(target, "main.go")

	content := mustReadFile(t, mainPath)
	if err := os.WriteFile(mainPath, append(content, []byte("// user edit\n")...), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Add(t.Context(), AddRequest{
		ProjectDir: target,
		Registry:   &reg,
		Features:   []string{metricsFeature},
	})

	var driftErr *DriftError
	if !errors.As(err, &driftErr) {
		t.Fatalf("error = %v, want DriftError", err)
	}

	if len(driftErr.Drifts) == 0 || driftErr.Drifts[0].Path != "main.go" {
		t.Fatalf("drifts = %+v", driftErr.Drifts)
	}
}

func TestAddForceOverridesDrift(t *testing.T) {
	t.Parallel()

	target, reg := projectWithFeatures(t, []string{"logging"})

	mainPath := filepath.Join(target, "main.go")

	content := mustReadFile(t, mainPath)
	if err := os.WriteFile(mainPath, append(content, []byte("// user edit\n")...), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Add(t.Context(), AddRequest{
		ProjectDir: target,
		Registry:   &reg,
		Features:   []string{metricsFeature},
		Force:      true,
	}); err != nil {
		t.Fatalf("force add: %v", err)
	}

	updated := mustReadFile(t, mainPath)
	if !strings.Contains(string(updated), "// user edit") {
		t.Fatal("user edit was lost")
	}
}

func TestRemoveRejectsDrift(t *testing.T) {
	t.Parallel()

	target, reg := projectWithFeatures(t, []string{"logging"})

	mainPath := filepath.Join(target, "main.go")

	content := mustReadFile(t, mainPath)
	if err := os.WriteFile(mainPath, append(content, []byte("// user edit\n")...), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Remove(t.Context(), RemoveRequest{
		ProjectDir: target,
		Registry:   &reg,
		Features:   []string{"logging"},
	})

	var driftErr *DriftError
	if !errors.As(err, &driftErr) {
		t.Fatalf("error = %v, want DriftError", err)
	}
}

func TestAddRollsBackOnFailure(t *testing.T) {
	t.Parallel()

	_, registryDir := fixture(t)
	writeFile(
		t,
		filepath.Join(registryDir, "overlays", metricsFeature, "basic", "overlay.yaml"),
		metricsOverlayYAML+"  - insertAfter:\n      file: main.go\n      anchor: missing\n      content: 'x'\n",
	)

	loaded, err := registry.Load(t.Context(), registry.Options{Source: registryDir})
	if err != nil {
		t.Fatal(err)
	}

	plan, err := resolve.Resolve(loaded.Manifest, loaded.Dir, resolve.Selection{
		Name:      "Acme App",
		Kind:      "backend",
		Stack:     "test/plain",
		Variables: map[string]string{"module_path": "example.com/acme"},
	})
	if err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(t.TempDir(), "app")
	if _, err := Generate(t.Context(), Request{
		Registry: loaded,
		Plan:     plan,
		Target:   target,
		NoGit:    true,
	}); err != nil {
		t.Fatal(err)
	}

	beforeEnv := mustReadFile(t, filepath.Join(target, ".env.example"))

	_, err = Add(t.Context(), AddRequest{
		ProjectDir: target,
		Registry:   loaded,
		Features:   []string{metricsFeature},
	})
	if err == nil || !strings.Contains(err.Error(), "anchor") {
		t.Fatalf("error = %v, want anchor error", err)
	}

	if _, statErr := os.Stat(filepath.Join(target, "internal", metricsFeature, "metrics.go")); !os.IsNotExist(statErr) {
		t.Fatalf("metrics file survived rollback: %v", statErr)
	}

	afterEnv := mustReadFile(t, filepath.Join(target, ".env.example"))
	if string(beforeEnv) != string(afterEnv) {
		t.Fatalf("env file changed after rollback:\nbefore:\n%s\nafter:\n%s", beforeEnv, afterEnv)
	}

	state, err := project.LoadState(target)
	if err != nil {
		t.Fatal(err)
	}

	if len(state.Applied) != 0 {
		t.Fatalf("state advanced after rollback: %+v", state.Applied)
	}
}

func TestRemoveRejectsNonReversibleOverlay(t *testing.T) {
	t.Parallel()

	target, reg := projectWithFeatures(t, []string{"config"})

	_, err := Remove(t.Context(), RemoveRequest{
		ProjectDir: target,
		Registry:   &reg,
		Features:   []string{"config"},
	})
	if err == nil || !strings.Contains(err.Error(), "non-reversible") {
		t.Fatalf("error = %v, want non-reversible error", err)
	}

	if _, statErr := os.Stat(filepath.Join(target, "config.yaml")); statErr != nil {
		t.Fatalf("config.yaml was removed despite refusal: %v", statErr)
	}
}
