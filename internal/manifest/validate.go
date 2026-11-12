package manifest

import (
	"errors"
	"fmt"
	"maps"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// Validate checks the schema version, identifiers, references, variants, and
// selection limits. It returns the first violation with enough context to fix
// the manifest.
func (m *Manifest) Validate() error {
	if m.SchemaVersion != 1 {
		return fmt.Errorf("unsupported schema version %d: expected 1", m.SchemaVersion)
	}

	if strings.TrimSpace(m.Name) == "" {
		return errors.New("name must not be empty")
	}

	if len(m.Kinds) == 0 {
		return errors.New("kinds must not be empty")
	}

	if len(m.Stacks) == 0 {
		return errors.New("stacks must not be empty")
	}

	if err := validateKinds(m.Kinds); err != nil {
		return err
	}

	if err := validateStacks(m.Stacks, m.Kinds); err != nil {
		return err
	}

	if err := validateFeatures(m.Features, m.Kinds, m.Stacks); err != nil {
		return err
	}

	return validateFeatureReferences(m.Features)
}

func validateKinds(kinds map[string]Kind) error {
	for _, name := range slices.Sorted(maps.Keys(kinds)) {
		if strings.TrimSpace(name) == "" {
			return errors.New("kind name must not be empty")
		}

		if kinds[name].Max < 0 {
			return fmt.Errorf("kind %q: max must be >= 0", name)
		}
	}

	return nil
}

func validateStacks(stacks []Stack, kinds map[string]Kind) error {
	seen := make(map[string]bool, len(stacks))
	for _, stack := range stacks {
		if err := validateStack(stack, kinds, seen); err != nil {
			return err
		}
	}

	return nil
}

func validateStack(stack Stack, kinds map[string]Kind, seen map[string]bool) error {
	if strings.TrimSpace(stack.ID) == "" {
		return errors.New("stack id must not be empty")
	}

	if seen[stack.ID] {
		return fmt.Errorf("duplicate stack %q", stack.ID)
	}

	if _, ok := kinds[stack.Kind]; !ok {
		return fmt.Errorf("stack %q: unknown kind %q", stack.ID, stack.Kind)
	}

	if err := validateRegistryPath(stack.Template); err != nil {
		return fmt.Errorf("stack %q: template path: %w", stack.ID, err)
	}

	if len(stack.Variables) == 0 {
		return fmt.Errorf("stack %q: variables must not be empty", stack.ID)
	}

	if err := validateVariables(stack); err != nil {
		return err
	}

	seen[stack.ID] = true

	return nil
}

func validateVariables(stack Stack) error {
	seen := make(map[string]bool, len(stack.Variables))
	for _, variable := range stack.Variables {
		if strings.TrimSpace(variable) == "" {
			return fmt.Errorf("stack %q: variable must not be empty", stack.ID)
		}

		if seen[variable] {
			return fmt.Errorf("stack %q: duplicate variable %q", stack.ID, variable)
		}

		seen[variable] = true
	}

	return nil
}

func validateFeatures(features []Feature, kinds map[string]Kind, stacks []Stack) error {
	stackByID := make(map[string]Stack, len(stacks))
	for _, stack := range stacks {
		stackByID[stack.ID] = stack
	}

	seen := make(map[string]bool, len(features))
	for _, feature := range features {
		if err := validateFeature(feature, kinds, stackByID, seen); err != nil {
			return err
		}

		seen[feature.ID] = true
	}

	return nil
}

func validateFeature(
	feature Feature,
	kinds map[string]Kind,
	stackByID map[string]Stack,
	seen map[string]bool,
) error {
	if strings.TrimSpace(feature.ID) == "" {
		return errors.New("feature id must not be empty")
	}

	if seen[feature.ID] {
		return fmt.Errorf("duplicate feature %q", feature.ID)
	}

	if len(feature.Kinds) == 0 && len(feature.Stacks) == 0 {
		return fmt.Errorf("feature %q: kinds or stacks required", feature.ID)
	}

	for _, kind := range feature.Kinds {
		if _, ok := kinds[kind]; !ok {
			return fmt.Errorf("feature %q: unknown kind %q", feature.ID, kind)
		}
	}

	for _, stack := range feature.Stacks {
		if _, ok := stackByID[stack]; !ok {
			return fmt.Errorf("feature %q: unknown stack %q", feature.ID, stack)
		}
	}

	if err := validateVariants(feature); err != nil {
		return err
	}

	return validateFeatureOverlays(feature, stackByID)
}

func validateVariants(feature Feature) error {
	seen := make(map[string]bool, len(feature.Variants))
	defaultCount := 0

	for _, variant := range feature.Variants {
		if strings.TrimSpace(variant.ID) == "" {
			return fmt.Errorf("feature %q: variant id must not be empty", feature.ID)
		}

		if seen[variant.ID] {
			return fmt.Errorf("feature %q: duplicate variant %q", feature.ID, variant.ID)
		}

		seen[variant.ID] = true
		if variant.Default {
			defaultCount++
		}
	}

	if defaultCount > 1 {
		return fmt.Errorf("feature %q: at most one default variant is allowed", feature.ID)
	}

	return nil
}

func validateFeatureReferences(features []Feature) error {
	byID := make(map[string]Feature, len(features))
	for _, feature := range features {
		byID[feature.ID] = feature
	}

	for _, feature := range features {
		refs := make([]string, 0, len(feature.Requires)+len(feature.Conflicts))
		refs = append(refs, feature.Requires...)

		refs = append(refs, feature.Conflicts...)
		for _, ref := range refs {
			if err := validateReference(feature, ref, byID); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateReference(source Feature, ref string, byID map[string]Feature) error {
	id, variant, _ := strings.Cut(ref, ":")
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("feature %q: invalid reference %q", source.ID, ref)
	}

	if id == source.ID {
		return fmt.Errorf("feature %q: cannot reference itself", source.ID)
	}

	target, ok := byID[id]
	if !ok {
		return fmt.Errorf("feature %q: unknown feature %q", source.ID, id)
	}

	if variant == "" {
		return nil
	}

	for _, candidate := range target.Variants {
		if candidate.ID == variant {
			return nil
		}
	}

	return fmt.Errorf("feature %q: unknown variant %q on feature %q", source.ID, variant, id)
}

func validateFeatureOverlays(feature Feature, stackByID map[string]Stack) error {
	supported := supportedStacks(feature, stackByID)
	if len(supported) == 0 {
		return fmt.Errorf("feature %q: no stacks match its kinds", feature.ID)
	}

	if len(feature.Variants) > 0 && len(feature.Overlays) > 0 {
		return fmt.Errorf(
			"feature %q: features with variants must not define overlays",
			feature.ID,
		)
	}

	if len(feature.Variants) == 0 {
		if err := validateOverlaySet(feature.ID, "", feature.Overlays, supported, stackByID); err != nil {
			return err
		}
	}

	for _, variant := range feature.Variants {
		if err := validateOverlaySet(
			feature.ID,
			variant.ID,
			variant.Overlays,
			supported,
			stackByID,
		); err != nil {
			return err
		}
	}

	return nil
}

func supportedStacks(feature Feature, stackByID map[string]Stack) map[string]bool {
	supported := make(map[string]bool, len(feature.Stacks))
	if len(feature.Stacks) > 0 {
		for _, stackID := range feature.Stacks {
			supported[stackID] = true
		}

		return supported
	}

	for stackID, stack := range stackByID {
		if slices.Contains(feature.Kinds, stack.Kind) {
			supported[stackID] = true
		}
	}

	return supported
}

func validateOverlaySet(
	featureID string,
	variantID string,
	overlays map[string]string,
	supported map[string]bool,
	stackByID map[string]Stack,
) error {
	label := featureID
	if variantID != "" {
		label = featureID + ":" + variantID
	}

	for _, stackID := range slices.Sorted(maps.Keys(overlays)) {
		if _, known := stackByID[stackID]; !known {
			return fmt.Errorf("feature %q: unknown stack %q", featureID, stackID)
		}

		if !supported[stackID] {
			return fmt.Errorf("feature %q: overlay for unsupported stack %q", label, stackID)
		}

		if err := validateRegistryPath(overlays[stackID]); err != nil {
			return fmt.Errorf("feature %q: overlay path: %w", label, err)
		}
	}

	for _, stackID := range slices.Sorted(maps.Keys(supported)) {
		if strings.TrimSpace(overlays[stackID]) == "" {
			return fmt.Errorf("feature %q: overlay required for stack %q", label, stackID)
		}
	}

	return nil
}

func validateRegistryPath(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("path must not be empty")
	}

	if strings.Contains(value, "\\") {
		return fmt.Errorf("path %q must use forward slashes", value)
	}

	cleaned := path.Clean(value)
	if cleaned != value || path.IsAbs(value) || !filepath.IsLocal(filepath.FromSlash(value)) {
		return fmt.Errorf("path %q must be a safe relative path", value)
	}

	return nil
}
