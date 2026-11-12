// Package engine renders a resolved plan into a staged project tree and
// commits it atomically.
package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BoatengPeter/smokeforge/internal/project"
	"github.com/BoatengPeter/smokeforge/internal/registry"
	"github.com/BoatengPeter/smokeforge/internal/render"
	"github.com/BoatengPeter/smokeforge/internal/resolve"
)

// Request configures one project generation.
type Request struct {
	Registry    *registry.Result
	Plan        *resolve.Plan
	Target      string
	NoGit       bool
	DryRun      bool
	KeepStaging bool
	Progress    func(stage string)
}

// Result reports the committed project and its recorded state.
type Result struct {
	Target  string
	DryRun  bool
	Files   []string
	State   *project.State
	Lock    *project.Lock
	Staging string
}

// Generate renders the plan into a staging directory and moves it into place
// only after every operation succeeds.
func Generate(ctx context.Context, req Request) (result *Result, err error) {
	if err := validateRequest(ctx, req); err != nil {
		return nil, err
	}

	target, err := filepath.Abs(req.Target)
	if err != nil {
		return nil, fmt.Errorf("resolve target path: %w", err)
	}

	if err := ensureTargetAvailable(target); err != nil {
		return nil, err
	}

	staging, err := prepareStaging(target)
	if err != nil {
		return nil, err
	}

	committed := false

	defer func() {
		if committed {
			return
		}

		if req.KeepStaging {
			if err != nil {
				err = fmt.Errorf("%w (staging retained at %s)", err, staging)
			}

			return
		}

		_ = os.RemoveAll(staging)
	}()

	result, err = buildProject(ctx, req, staging)
	if err != nil {
		return nil, err
	}

	result.Target = target

	if req.DryRun {
		committed = true

		if !req.KeepStaging {
			_ = os.RemoveAll(staging)
		}

		return result, nil
	}

	if err := commitProject(staging, target); err != nil {
		req.KeepStaging = true

		return nil, err
	}

	committed = true

	report(req, "project created")

	return result, nil
}

func prepareStaging(target string) (string, error) {
	parent := filepath.Dir(target)

	//nolint:gosec // generated project parents must be traversable by the user.
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", fmt.Errorf("create target parent: %w", err)
	}

	staging, err := os.MkdirTemp(parent, ".smokeforge-staging-*")
	if err != nil {
		return "", fmt.Errorf("create staging directory: %w", err)
	}

	return staging, nil
}

func buildProject(ctx context.Context, req Request, staging string) (*Result, error) {
	root, err := os.OpenRoot(staging)
	if err != nil {
		return nil, fmt.Errorf("open staging root: %w", err)
	}

	defer func() {
		_ = root.Close()
	}()

	renderer := render.New(req.Plan.Name, req.Plan.Target.Variables)

	report(req, "rendering base templates")

	if _, err := renderBase(ctx, root, req.Plan, renderer); err != nil {
		return nil, err
	}

	report(req, "applying feature overlays")

	applications, envEntries, err := applyOverlays(ctx, staging, req.Plan, renderer)
	if err != nil {
		return nil, err
	}

	report(req, "writing environment files")

	if err := writeEnvFiles(root, envEntries); err != nil {
		return nil, err
	}

	report(req, "writing project state")

	state, lock, err := buildState(req, applications, staging)
	if err != nil {
		return nil, err
	}

	if err := project.WriteState(staging, state, lock); err != nil {
		return nil, err
	}

	if !req.NoGit {
		report(req, "initializing git repository")

		if err := initGit(ctx, staging); err != nil {
			return nil, err
		}
	}

	files, err := listFiles(staging)
	if err != nil {
		return nil, err
	}

	return &Result{
		DryRun:  req.DryRun,
		Files:   files,
		State:   state,
		Lock:    lock,
		Staging: staging,
	}, nil
}

func commitProject(staging, target string) error {
	//nolint:gosec // generated projects must be readable by their owner and tools.
	if err := os.Chmod(staging, 0o755); err != nil {
		return fmt.Errorf("set project permissions: %w", err)
	}

	if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("replace empty target: %w", err)
	}

	if err := os.Rename(staging, target); err != nil {
		return fmt.Errorf("commit project: %w", err)
	}

	return nil
}

func validateRequest(ctx context.Context, req Request) error {
	if ctx == nil {
		return errors.New("generate: context is required")
	}

	if req.Registry == nil || req.Registry.Manifest == nil {
		return errors.New("generate: registry is required")
	}

	if req.Plan == nil {
		return errors.New("generate: plan is required")
	}

	if req.Target == "" {
		return errors.New("generate: target is required")
	}

	if req.Plan.Target.Path != "." {
		return fmt.Errorf("generate: unsupported target path %q", req.Plan.Target.Path)
	}

	if req.Plan.Target.TemplateDir == "" {
		return errors.New("generate: template directory is required")
	}

	return nil
}

func ensureTargetAvailable(target string) error {
	info, err := os.Stat(target)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("stat target %q: %w", target, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("target %q is not a directory", target)
	}

	entries, err := os.ReadDir(target)
	if err != nil {
		return fmt.Errorf("read target %q: %w", target, err)
	}

	if len(entries) > 0 {
		return fmt.Errorf("target %q is not empty", target)
	}

	return nil
}

func report(req Request, stage string) {
	if req.Progress != nil {
		req.Progress(stage)
	}
}
