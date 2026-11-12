package engine

import (
	"fmt"
	"strings"

	"github.com/BoatengPeter/smokeforge/internal/manifest"
)

func expandRemovals(
	m *manifest.Manifest,
	active []string,
	requested []string,
	cascade bool,
) ([]string, error) {
	removing := initialRemoving(active, requested)
	expanded := append([]string{}, requested...)

	for {
		added, err := addDependentRemovals(m, active, removing, cascade)
		if err != nil {
			return nil, err
		}

		if len(added) == 0 {
			return expanded, nil
		}

		expanded = append(expanded, added...)
	}
}

func initialRemoving(active, requested []string) map[string]string {
	removing := make(map[string]string, len(requested))

	for _, raw := range requested {
		id, variantID := splitFeature(raw)
		if variantID == "" {
			variantID = activeVariant(active, id)
		}

		removing[id] = variantID
	}

	return removing
}

func addDependentRemovals(
	m *manifest.Manifest,
	active []string,
	removing map[string]string,
	cascade bool,
) ([]string, error) {
	added := make([]string, 0)

	for _, activeRaw := range active {
		id, variantID := splitFeature(activeRaw)
		if _, alreadyRemoving := removing[id]; alreadyRemoving {
			continue
		}

		feature, ok := findManifestFeature(m, id)
		if !ok {
			continue
		}

		requiredID, blocked := blockingRequirement(feature, variantID, removing)
		if !blocked {
			continue
		}

		if !cascade {
			return nil, fmt.Errorf(
				"feature %q is required by %q; use --cascade to remove both",
				requiredID,
				id,
			)
		}

		removing[id] = variantID

		added = append(added, activeRaw)
	}

	return added, nil
}

func blockingRequirement(
	feature manifest.Feature,
	variantID string,
	removing map[string]string,
) (string, bool) {
	for _, ref := range featureRequirements(feature, variantID) {
		requiredID, requiredVariant := splitFeature(ref)

		removedVariant, removingRequired := removing[requiredID]
		if !removingRequired {
			continue
		}

		if requiredVariant != "" && removedVariant != "" && requiredVariant != removedVariant {
			continue
		}

		return requiredID, true
	}

	return "", false
}

func activeVariant(active []string, featureID string) string {
	for _, raw := range active {
		id, variantID := splitFeature(raw)
		if id == featureID {
			return variantID
		}
	}

	return ""
}

func findManifestFeature(m *manifest.Manifest, id string) (manifest.Feature, bool) {
	for _, feature := range m.Features {
		if feature.ID == id {
			return feature, true
		}
	}

	return manifest.Feature{}, false
}

func featureRequirements(feature manifest.Feature, variantID string) []string {
	refs := append([]string{}, feature.Requires...)

	for _, variant := range feature.Variants {
		if strings.TrimSpace(variantID) != "" && variant.ID == variantID {
			refs = append(refs, variant.Requires...)

			break
		}
	}

	return refs
}
