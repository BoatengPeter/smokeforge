package engine

import (
	"fmt"
	"maps"
	"path/filepath"
	"time"

	"github.com/BoatengPeter/smokeforge/internal/project"
)

func buildState(
	req Request,
	applications []overlayApplication,
	staging string,
) (*project.State, *project.Lock, error) {
	features := make([]string, 0, len(req.Plan.Target.Overlays))
	for _, overlay := range req.Plan.Target.Overlays {
		features = append(features, overlay.FeatureID+":"+overlay.VariantID)
	}

	applied := make([]project.AppliedOverlay, 0, len(applications))
	for _, application := range applications {
		files := make(map[string]string, len(application.Files))
		for _, file := range application.Files {
			hash, err := project.HashFile(filepath.Join(staging, filepath.FromSlash(file)))
			if err != nil {
				return nil, nil, fmt.Errorf("hash generated file %q: %w", file, err)
			}

			files[file] = hash
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
			Target:      req.Plan.Target.Path,
			RegistryTag: req.Registry.Tag,
			Files:       files,
			Markers:     markers,
		})
	}

	templateHash, err := project.HashDir(req.Plan.Target.TemplateDir)
	if err != nil {
		return nil, nil, fmt.Errorf("hash template: %w", err)
	}

	overlayHashes := make(map[string]string, len(applications))
	for _, application := range applications {
		hash, hashErr := project.HashDir(application.Dir)
		if hashErr != nil {
			return nil, nil, fmt.Errorf("hash overlay %q: %w", application.Path, hashErr)
		}

		overlayHashes[application.Path] = hash
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
		Variables:    maps.Clone(req.Plan.Target.Variables),
		Targets:      []project.Target{{Path: req.Plan.Target.Path, Stack: req.Plan.Target.Stack.ID, Features: features}},
		RootFeatures: make([]string, 0),
		Applied:      applied,
	}

	lock := &project.Lock{
		LockVersion: 1,
		Registry: project.RegistryLock{
			Source: req.Registry.Source,
			Tag:    req.Registry.Tag,
		},
		Templates: map[string]string{req.Plan.Target.Stack.Template: templateHash},
		Overlays:  overlayHashes,
	}

	return state, lock, nil
}
