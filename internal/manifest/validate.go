package manifest

import (
	"errors"
	"fmt"
	"maps"
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

	if strings.TrimSpace(stack.Template) == "" {
		return fmt.Errorf("stack %q: template must not be empty", stack.ID)
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
	stackIDs := make(map[string]bool, len(stacks))
	for _, stack := range stacks {
		stackIDs[stack.ID] = true
	}

	seen := make(map[string]bool, len(features))
	for _, feature := range features {
		if err := validateFeature(feature, kinds, stackIDs, seen); err != nil {
			return err
		}

		seen[feature.ID] = true
	}

	return nil
}

func validateFeature(
	feature Feature,
	kinds map[string]Kind,
	stackIDs map[string]bool,
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
		if !stackIDs[stack] {
			return fmt.Errorf("feature %q: unknown stack %q", feature.ID, stack)
		}
	}

	return validateVariants(feature)
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
