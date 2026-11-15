package engine

import (
	"fmt"
	"maps"
	"path/filepath"
	"time"

	"github.com/BoatengPeter/smokeforge/internal/project"
	"github.com/BoatengPeter/smokeforge/internal/resolve"
)

type targetBuild struct {
	plan         resolve.TargetPlan
	dir          string
	applications []overlayApplication
}

type buildMetadata struct {
	applied        []project.AppliedOverlay
	targets        []project.Target
	rootFeatures   []string
	templateHashes map[string]string
	overlayHashes  map[string]string
	variables      map[string]string
}

func buildState(
	req Request,
	builds []targetBuild,
) (*project.State, *project.Lock, error) {
	metadata, err := collectBuildMetadata(req, builds)
	if err != nil {
		return nil, nil, err
	}

	state := &project.State{
		SchemaVersion: 1,
		Name:          req.Plan.Name,
		Kind:          req.Plan.Kind,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		Registry: project.RegistryRef{
			Source: req.Registry.Source,
			Tag:    req.Registry.Tag,
		},
		Variables:    metadata.variables,
		Targets:      metadata.targets,
		RootFeatures: metadata.rootFeatures,
		Applied:      metadata.applied,
	}

	lock := &project.Lock{
		LockVersion: 1,
		Registry: project.RegistryLock{
			Source: req.Registry.Source,
			Tag:    req.Registry.Tag,
		},
		Templates: metadata.templateHashes,
		Overlays:  metadata.overlayHashes,
	}

	return state, lock, nil
}

func collectBuildMetadata(req Request, builds []targetBuild) (*buildMetadata, error) {
	metadata := &buildMetadata{
		applied:        make([]project.AppliedOverlay, 0, len(builds)),
		targets:        make([]project.Target, 0, len(builds)),
		rootFeatures:   make([]string, 0),
		templateHashes: make(map[string]string, len(builds)),
		overlayHashes:  make(map[string]string, len(builds)),
		variables:      make(map[string]string),
	}

	for _, build := range builds {
		features := featuresFromTarget(build.plan)
		metadata.recordTarget(req, build, features)

		applications, err := recordApplications(req, build)
		if err != nil {
			return nil, err
		}

		metadata.applied = append(metadata.applied, applications...)

		templateHash, err := project.HashDir(build.plan.TemplateDir)
		if err != nil {
			return nil, fmt.Errorf("hash template: %w", err)
		}

		metadata.templateHashes[build.plan.Stack.Template] = templateHash

		if err := recordOverlayHashes(build, metadata.overlayHashes); err != nil {
			return nil, err
		}
	}

	return metadata, nil
}

func (m *buildMetadata) recordTarget(req Request, build targetBuild, features []string) {
	if build.plan.Path == "." && req.Plan.Root != nil {
		m.rootFeatures = features

		return
	}

	if len(m.targets) == 0 {
		m.variables = maps.Clone(build.plan.Variables)
	}

	m.targets = append(m.targets, project.Target{
		Path:     build.plan.Path,
		Stack:    build.plan.Stack.ID,
		Features: features,
	})
}

func recordApplications(req Request, build targetBuild) ([]project.AppliedOverlay, error) {
	applied := make([]project.AppliedOverlay, 0, len(build.applications))

	for _, application := range build.applications {
		files, envKeys, err := hashApplication(build.dir, application)
		if err != nil {
			return nil, err
		}

		overlayID := application.FeatureID
		if application.VariantID != "" {
			overlayID += ":" + application.VariantID
		}

		markers := application.Markers
		if markers == nil {
			markers = make([]string, 0)
		}

		applied = append(applied, project.AppliedOverlay{
			Overlay:     overlayID,
			Target:      build.plan.Path,
			RegistryTag: req.Registry.Tag,
			Files:       files,
			Markers:     markers,
			Env:         envKeys,
		})
	}

	return applied, nil
}

func recordOverlayHashes(build targetBuild, hashes map[string]string) error {
	for _, application := range build.applications {
		hash, err := project.HashDir(application.Dir)
		if err != nil {
			return fmt.Errorf("hash overlay %q: %w", application.Path, err)
		}

		hashes[application.Path] = hash
	}

	return nil
}

func hashApplication(
	staging string,
	application overlayApplication,
) (map[string]string, []string, error) {
	tracked := make([]string, 0, len(application.Files)+2)
	tracked = append(tracked, application.Files...)

	envKeys := make([]string, 0, len(application.Env))
	for _, entry := range application.Env {
		envKeys = append(envKeys, entry.Key)
	}

	if len(envKeys) > 0 {
		tracked = append(tracked, ".env.example")
	}

	files := make(map[string]string, len(tracked))
	for _, file := range tracked {
		hash, err := project.HashFile(filepath.Join(staging, filepath.FromSlash(file)))
		if err != nil {
			return nil, nil, fmt.Errorf("hash generated file %q: %w", file, err)
		}

		files[file] = hash
	}

	return files, envKeys, nil
}

func featuresFromTarget(target resolve.TargetPlan) []string {
	features := make([]string, 0, len(target.Overlays))

	for _, overlay := range target.Overlays {
		feature := overlay.FeatureID
		if overlay.VariantID != "" {
			feature += ":" + overlay.VariantID
		}

		features = append(features, feature)
	}

	return features
}

func featuresFromPlan(plan *resolve.Plan) []string {
	return featuresFromTarget(plan.Target)
}
