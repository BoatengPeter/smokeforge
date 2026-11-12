package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BoatengPeter/smokeforge/internal/patch"
	"github.com/BoatengPeter/smokeforge/internal/project"
	"github.com/BoatengPeter/smokeforge/internal/registry"
	"github.com/BoatengPeter/smokeforge/internal/render"
	"github.com/BoatengPeter/smokeforge/internal/resolve"
)

// AddRequest configures adding features to a generated project.
type AddRequest struct {
	ProjectDir string
	Registry   *registry.Result
	Features   []string
	Force      bool
}

// RemoveRequest configures removing features from a generated project.
type RemoveRequest struct {
	ProjectDir string
	Registry   *registry.Result
	Features   []string
	Cascade    bool
}

// MutationResult reports the updated project and affected overlays.
type MutationResult struct {
	ProjectDir string
	State      *project.State
	Lock       *project.Lock
	Added      []string
	Removed    []string
}

// DriftError reports local edits that block a mutation.
type DriftError struct {
	Drifts []project.Drift
}

// Error lists the drifted files.
func (e *DriftError) Error() string {
	paths := make([]string, 0, len(e.Drifts))
	for _, drift := range e.Drifts {
		paths = append(paths, drift.Path)
	}

	return "project files have drifted from generated state: " + strings.Join(paths, ", ")
}

type projectContext struct {
	root     string
	state    *project.State
	lock     *project.Lock
	target   project.Target
	renderer *render.Renderer
}

type preparedAdd struct {
	pc       *projectContext
	plan     *resolve.Plan
	overlays []resolve.OverlayPlan
	touched  []string
}

// Add resolves requested features against the locked registry, applies their
// overlays, and updates project state inside a backup-protected transaction.
func Add(ctx context.Context, req AddRequest) (*MutationResult, error) {
	prepared, err := prepareAdd(req)
	if err != nil {
		return nil, err
	}

	result := &MutationResult{ProjectDir: prepared.pc.root}

	err = withBackup(prepared.pc.root, prepared.touched, func() error {
		return applyAddedFeatures(ctx, prepared, result)
	})
	if err != nil {
		return nil, err
	}

	result.State = prepared.pc.state
	result.Lock = prepared.pc.lock

	return result, nil
}

func prepareAdd(req AddRequest) (*preparedAdd, error) {
	if len(req.Features) == 0 {
		return nil, errors.New("add: at least one feature is required")
	}

	pc, err := loadProjectContext(req.ProjectDir)
	if err != nil {
		return nil, err
	}

	if err := validateRegistry(req.Registry, pc.lock); err != nil {
		return nil, err
	}

	if err := ensureFeaturesNew(pc.state, req.Features); err != nil {
		return nil, err
	}

	plan, err := resolveFeatures(pc, req.Registry, append(slices.Clone(pc.target.Features), req.Features...))
	if err != nil {
		return nil, err
	}

	overlays := notApplied(plan, pc.state)
	if len(overlays) == 0 {
		return nil, errors.New("nothing to add: requested features are already applied")
	}

	if !req.Force {
		if err := ensureNoDrift(pc.root, pc.state); err != nil {
			return nil, err
		}
	}

	touched, err := impactedFiles(overlays, pc.renderer)
	if err != nil {
		return nil, err
	}

	touched = append(touched, project.StateFile, project.LockFile)

	return &preparedAdd{pc: pc, plan: plan, overlays: overlays, touched: touched}, nil
}

func applyAddedFeatures(
	ctx context.Context,
	prepared *preparedAdd,
	result *MutationResult,
) error {
	pc := prepared.pc

	applications, envEntries, err := applyOverlayPlans(
		ctx,
		pc.root,
		prepared.overlays,
		pc.renderer,
	)
	if err != nil {
		return err
	}

	if err := applyEnvEntries(pc.root, envEntries); err != nil {
		return err
	}

	for _, application := range applications {
		appliedState, err := buildAppliedOverlay(pc.root, application)
		if err != nil {
			return err
		}

		appliedState.RegistryTag = pc.lock.Registry.Tag

		hash, err := project.HashDir(application.Dir)
		if err != nil {
			return fmt.Errorf("hash overlay %q: %w", application.Path, err)
		}

		pc.state.Applied = append(pc.state.Applied, appliedState)
		pc.lock.Overlays[application.Path] = hash

		result.Added = append(result.Added, appliedState.Overlay)
	}

	pc.state.Targets[0].Features = featuresFromPlan(prepared.plan)

	if err := project.Rehash(pc.root, pc.state); err != nil {
		return err
	}

	return project.WriteState(pc.root, pc.state, pc.lock)
}

type preparedRemove struct {
	pc            *projectContext
	remainingPlan *resolve.Plan
	fullPlan      *resolve.Plan
	plans         []resolve.OverlayPlan
	indexes       []int
	touched       []string
	envKeys       []string
}

// Remove reverts reversible overlays in reverse application order and updates
// project state inside a backup-protected transaction.
func Remove(ctx context.Context, req RemoveRequest) (*MutationResult, error) {
	prepared, err := prepareRemove(req)
	if err != nil {
		return nil, err
	}

	result := &MutationResult{ProjectDir: prepared.pc.root}

	err = withBackup(prepared.pc.root, prepared.touched, func() error {
		return applyRemovedFeatures(ctx, prepared, result)
	})
	if err != nil {
		return nil, err
	}

	result.State = prepared.pc.state
	result.Lock = prepared.pc.lock

	return result, nil
}

func prepareRemove(req RemoveRequest) (*preparedRemove, error) {
	if len(req.Features) == 0 {
		return nil, errors.New("remove: at least one feature is required")
	}

	pc, err := loadProjectContext(req.ProjectDir)
	if err != nil {
		return nil, err
	}

	if err := validateRegistry(req.Registry, pc.lock); err != nil {
		return nil, err
	}

	if _, err := appliedIndexes(pc.state, req.Features); err != nil {
		return nil, err
	}

	removal, err := expandRemovals(
		req.Registry.Manifest,
		pc.target.Features,
		req.Features,
		req.Cascade,
	)
	if err != nil {
		return nil, err
	}

	indexes, err := appliedIndexes(pc.state, removal)
	if err != nil {
		return nil, err
	}

	remaining, err := remainingFeatures(pc.target.Features, removal)
	if err != nil {
		return nil, err
	}

	remainingPlan, err := resolveFeatures(pc, req.Registry, remaining)
	if err != nil {
		return nil, fmt.Errorf("remove would break dependencies: %w", err)
	}

	fullPlan, err := resolveFeatures(pc, req.Registry, pc.target.Features)
	if err != nil {
		return nil, err
	}

	plans, touched, envKeys, err := prepareRemovalArtifacts(pc, fullPlan, indexes)
	if err != nil {
		return nil, err
	}

	return &preparedRemove{
		pc:            pc,
		remainingPlan: remainingPlan,
		fullPlan:      fullPlan,
		plans:         plans,
		indexes:       indexes,
		touched:       touched,
		envKeys:       envKeys,
	}, nil
}

func prepareRemovalArtifacts(
	pc *projectContext,
	fullPlan *resolve.Plan,
	indexes []int,
) ([]resolve.OverlayPlan, []string, []string, error) {
	plans, err := overlayPlansForIndexes(fullPlan, pc.state, indexes)
	if err != nil {
		return nil, nil, nil, err
	}

	for _, overlay := range plans {
		if err := patch.CheckRemovable(overlay.Overlay); err != nil {
			return nil, nil, nil, err
		}
	}

	if err := ensureNoDrift(pc.root, pc.state); err != nil {
		return nil, nil, nil, err
	}

	touched, err := impactedFiles(plans, pc.renderer)
	if err != nil {
		return nil, nil, nil, err
	}

	touched = append(touched, project.StateFile, project.LockFile)

	envKeys := make([]string, 0, len(indexes))
	for _, index := range indexes {
		envKeys = append(envKeys, pc.state.Applied[index].Env...)
	}

	return plans, touched, envKeys, nil
}

func applyRemovedFeatures(
	ctx context.Context,
	prepared *preparedRemove,
	result *MutationResult,
) error {
	pc := prepared.pc

	sorted := slices.Clone(prepared.indexes)
	slices.Sort(sorted)
	slices.Reverse(sorted)

	for _, index := range sorted {
		overlay, err := overlayPlanByID(prepared.fullPlan, pc.state.Applied[index].Overlay)
		if err != nil {
			return err
		}

		if err := patch.Remove(ctx, overlay.Overlay, patch.RemoveOptions{
			Root:       pc.root,
			OverlayDir: overlay.Dir,
			Renderer:   pc.renderer,
		}); err != nil {
			return err
		}

		result.Removed = append(result.Removed, pc.state.Applied[index].Overlay)
	}

	if err := removeEnvKeys(pc.root, prepared.envKeys); err != nil {
		return err
	}

	pc.state.Applied = removeApplied(pc.state.Applied, prepared.indexes)
	pc.state.Targets[0].Features = featuresFromPlan(prepared.remainingPlan)

	for _, plan := range prepared.plans {
		delete(pc.lock.Overlays, plan.Path)
	}

	if err := project.Rehash(pc.root, pc.state); err != nil {
		return err
	}

	return project.WriteState(pc.root, pc.state, pc.lock)
}

func loadProjectContext(projectDir string) (*projectContext, error) {
	root, err := project.FindRoot(projectDir)
	if err != nil {
		return nil, err
	}

	state, err := project.LoadState(root)
	if err != nil {
		return nil, err
	}

	lock, err := project.LoadLock(root)
	if err != nil {
		return nil, err
	}

	if len(state.Targets) != 1 || state.Targets[0].Path != "." {
		return nil, errors.New("multi-target projects are not supported yet")
	}

	return &projectContext{
		root:     root,
		state:    state,
		lock:     lock,
		target:   state.Targets[0],
		renderer: render.New(state.Name, state.Variables),
	}, nil
}

func validateRegistry(reg *registry.Result, lock *project.Lock) error {
	if reg == nil || reg.Manifest == nil {
		return errors.New("registry is required")
	}

	if lock.Registry.Tag != "" && reg.Tag != lock.Registry.Tag {
		return fmt.Errorf(
			"registry tag mismatch: project is locked to %q, loaded %q",
			lock.Registry.Tag,
			reg.Tag,
		)
	}

	return nil
}

func resolveFeatures(
	pc *projectContext,
	reg *registry.Result,
	features []string,
) (*resolve.Plan, error) {
	return resolve.Resolve(reg.Manifest, reg.Dir, resolve.Selection{
		Name:      pc.state.Name,
		Kind:      pc.state.Kind,
		Stack:     pc.target.Stack,
		Features:  features,
		Variables: pc.state.Variables,
	})
}

func ensureFeaturesNew(state *project.State, requested []string) error {
	applied := make(map[string]bool, len(state.Applied))
	for _, overlay := range state.Applied {
		featureID, _ := splitFeature(overlay.Overlay)
		applied[featureID] = true
	}

	for _, raw := range requested {
		featureID, _ := splitFeature(raw)
		if applied[featureID] {
			return fmt.Errorf("feature %q is already applied", featureID)
		}
	}

	return nil
}

func notApplied(plan *resolve.Plan, state *project.State) []resolve.OverlayPlan {
	applied := make(map[string]bool, len(state.Applied))
	for _, overlay := range state.Applied {
		applied[overlay.Overlay] = true
	}

	overlays := make([]resolve.OverlayPlan, 0, len(plan.Target.Overlays))
	for _, overlay := range plan.Target.Overlays {
		id := overlay.FeatureID
		if overlay.VariantID != "" {
			id += ":" + overlay.VariantID
		}

		if applied[id] {
			continue
		}

		overlays = append(overlays, overlay)
	}

	return overlays
}

func appliedIndexes(state *project.State, requested []string) ([]int, error) {
	indexes := make([]int, 0, len(requested))
	matched := make([]bool, len(requested))

	for index := range state.Applied {
		appliedFeature, appliedVariant := splitFeature(state.Applied[index].Overlay)

		for position, raw := range requested {
			featureID, variantID := splitFeature(raw)
			if featureID != appliedFeature {
				continue
			}

			if variantID != "" && variantID != appliedVariant {
				continue
			}

			matched[position] = true

			if !slices.Contains(indexes, index) {
				indexes = append(indexes, index)
			}

			break
		}
	}

	for position, raw := range requested {
		if !matched[position] {
			return nil, fmt.Errorf("feature %q is not applied", raw)
		}
	}

	return indexes, nil
}

func remainingFeatures(existing, removed []string) ([]string, error) {
	remaining := make([]string, 0, len(existing))

	for _, feature := range existing {
		featureID, variantID := splitFeature(feature)
		keep := true

		for _, raw := range removed {
			removedID, removedVariant := splitFeature(raw)
			if featureID != removedID {
				continue
			}

			if removedVariant == "" || removedVariant == variantID {
				keep = false

				break
			}
		}

		if keep {
			remaining = append(remaining, feature)
		}
	}

	return remaining, nil
}

func overlayPlansForIndexes(
	plan *resolve.Plan,
	state *project.State,
	indexes []int,
) ([]resolve.OverlayPlan, error) {
	plans := make([]resolve.OverlayPlan, 0, len(indexes))

	for _, index := range indexes {
		overlay, err := overlayPlanByID(plan, state.Applied[index].Overlay)
		if err != nil {
			return nil, err
		}

		plans = append(plans, *overlay)
	}

	return plans, nil
}

func overlayPlanByID(
	plan *resolve.Plan,
	id string,
) (*resolve.OverlayPlan, error) {
	for index := range plan.Target.Overlays {
		overlay := &plan.Target.Overlays[index]

		overlayID := overlay.FeatureID
		if overlay.VariantID != "" {
			overlayID += ":" + overlay.VariantID
		}

		if overlayID == id {
			return overlay, nil
		}
	}

	return nil, fmt.Errorf("overlay %q is not present in the registry", id)
}

func removeApplied(applied []project.AppliedOverlay, indexes []int) []project.AppliedOverlay {
	kept := make([]project.AppliedOverlay, 0, len(applied)-len(indexes))

	for index, overlay := range applied {
		if slices.Contains(indexes, index) {
			continue
		}

		kept = append(kept, overlay)
	}

	return kept
}

func splitFeature(raw string) (string, string) {
	featureID, variantID, _ := strings.Cut(raw, ":")

	return featureID, variantID
}

func ensureNoDrift(root string, state *project.State) error {
	drifts, err := project.CheckDrift(root, state)
	if err != nil {
		return err
	}

	if len(drifts) > 0 {
		return &DriftError{Drifts: drifts}
	}

	return nil
}

func withBackup(root string, files []string, mutate func() error) error {
	backup, err := project.CreateBackup(root, files)
	if err != nil {
		return err
	}

	if err := mutate(); err != nil {
		if restoreErr := backup.Restore(); restoreErr != nil {
			return errors.Join(
				err,
				fmt.Errorf("restore backup at %s: %w", backup.Dir(), restoreErr),
			)
		}

		return err
	}

	_ = backup.Discard()

	return nil
}

func applyOverlayPlans(
	ctx context.Context,
	root string,
	plans []resolve.OverlayPlan,
	renderer *render.Renderer,
) ([]overlayApplication, []patch.Env, error) {
	applications := make([]overlayApplication, 0, len(plans))
	envEntries := make([]patch.Env, 0, len(plans))

	for _, plan := range plans {
		if err := ctx.Err(); err != nil {
			return nil, nil, fmt.Errorf("apply overlay: %w", err)
		}

		applied, err := patch.Apply(ctx, plan.Overlay, patch.ApplyOptions{
			Root:       root,
			OverlayDir: plan.Dir,
			Renderer:   renderer,
		})
		if err != nil {
			return nil, nil, err
		}

		applications = append(applications, overlayApplication{
			FeatureID: plan.FeatureID,
			VariantID: plan.VariantID,
			Path:      plan.Path,
			Dir:       plan.Dir,
			Files:     applied.Files,
			Markers:   applied.Markers,
			Env:       applied.Env,
		})
		envEntries = append(envEntries, applied.Env...)
	}

	return applications, envEntries, nil
}

func buildAppliedOverlay(
	root string,
	application overlayApplication,
) (project.AppliedOverlay, error) {
	files, envKeys, err := hashApplication(root, application)
	if err != nil {
		return project.AppliedOverlay{}, err
	}

	overlayID := application.FeatureID
	if application.VariantID != "" {
		overlayID += ":" + application.VariantID
	}

	markers := application.Markers
	if markers == nil {
		markers = make([]string, 0)
	}

	return project.AppliedOverlay{
		Overlay:     overlayID,
		Target:      ".",
		RegistryTag: "",
		Files:       files,
		Markers:     markers,
		Env:         envKeys,
	}, nil
}

func impactedFiles(
	plans []resolve.OverlayPlan,
	renderer *render.Renderer,
) ([]string, error) {
	files := make([]string, 0, len(plans)*4)

	for _, plan := range plans {
		for _, op := range plan.Overlay.Ops {
			paths, err := opPaths(op, renderer)
			if err != nil {
				return nil, err
			}

			for _, path := range paths {
				if !slices.Contains(files, path) {
					files = append(files, path)
				}
			}
		}
	}

	return files, nil
}

func opPaths(op patch.Op, renderer *render.Renderer) ([]string, error) {
	switch {
	case op.AddFiles != nil:
		paths := make([]string, 0, len(*op.AddFiles))
		for _, file := range *op.AddFiles {
			dest, err := renderer.RenderPath(file.Dest)
			if err != nil {
				return nil, fmt.Errorf("render overlay destination %q: %w", file.Dest, err)
			}

			paths = append(paths, dest)
		}

		return paths, nil
	case op.InsertBefore != nil:
		return renderPath(renderer, op.InsertBefore.File)
	case op.InsertAfter != nil:
		return renderPath(renderer, op.InsertAfter.File)
	case op.PatchJSON != nil:
		return renderPath(renderer, op.PatchJSON.File)
	case op.PatchYAML != nil:
		return renderPath(renderer, op.PatchYAML.File)
	case op.PatchTOML != nil:
		return renderPath(renderer, op.PatchTOML.File)
	case op.AddDependency != nil:
		return renderPath(renderer, op.AddDependency.File)
	case op.Env != nil:
		return []string{".env", ".env.example"}, nil
	default:
		return nil, nil
	}
}

func renderPath(renderer *render.Renderer, value string) ([]string, error) {
	path, err := renderer.RenderPath(value)
	if err != nil {
		return nil, fmt.Errorf("render overlay path %q: %w", value, err)
	}

	return []string{path}, nil
}

func applyEnvEntries(root string, entries []patch.Env) error {
	if len(entries) == 0 {
		return nil
	}

	handle, err := os.OpenRoot(root)
	if err != nil {
		return fmt.Errorf("open project root: %w", err)
	}

	defer func() {
		_ = handle.Close()
	}()

	return writeEnvFiles(handle, entries)
}

func removeEnvKeys(root string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	targets := make(map[string]bool, len(keys))
	for _, key := range keys {
		targets[key] = true
	}

	for _, name := range []string{".env", ".env.example"} {
		full := filepath.Join(root, name)

		info, err := os.Stat(full)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}

		if err != nil {
			return fmt.Errorf("stat %q: %w", name, err)
		}

		//nolint:gosec // full is a known env file under the project root.
		data, err := os.ReadFile(full)
		if err != nil {
			return fmt.Errorf("read %q: %w", name, err)
		}

		filtered := filterEnvLines(string(data), targets)

		if err := os.WriteFile(full, []byte(filtered), info.Mode().Perm()); err != nil {
			return fmt.Errorf("write %q: %w", name, err)
		}
	}

	return nil
}

func filterEnvLines(content string, remove map[string]bool) string {
	lines := splitConfigLines(content)
	kept := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			key, _, found := strings.Cut(trimmed, "=")
			if found && remove[strings.TrimSpace(key)] {
				continue
			}
		}

		kept = append(kept, line)
	}

	return renderLines(kept)
}
