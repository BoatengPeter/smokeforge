package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BoatengPeter/smokeforge/internal/manifest"
	"github.com/charmbracelet/huh"
)

// WizardResult holds the choices collected by the interactive wizard.
type WizardResult struct {
	Kind      string
	Stack     string
	Name      string
	Preset    string
	Features  []string
	Variables map[string]string
}

// Wizard collects a project selection from the user.
type Wizard struct {
	Manifest    *manifest.Manifest
	RegistryDir string
	In          io.Reader
	Out         io.Writer
	Result      WizardResult
}

// Run executes the interactive wizard. It aborts cleanly when the user cancels.
func (w *Wizard) Run(ctx context.Context) error {
	kinds := availableKinds(w.Manifest)
	if len(kinds) == 0 {
		return errors.New("wizard: the registry has no selectable kinds")
	}

	kind := w.Result.Kind
	if !slices.Contains(kinds, kind) {
		kind = kinds[0]
	}

	kindForm := newForm(w, huh.NewGroup(
		huh.NewSelect[string]().
			Title("What kind of project is this?").
			Options(kindOptions(kinds)...).
			Value(&kind),
	))
	if err := kindForm.RunWithContext(ctx); err != nil {
		return fmt.Errorf("wizard: %w", err)
	}

	stack, err := w.selectStack(ctx, kind)
	if err != nil {
		return err
	}

	name, err := w.askName(ctx)
	if err != nil {
		return err
	}

	preset, err := w.selectPreset(ctx, kind)
	if err != nil {
		return err
	}

	features, err := w.selectFeatures(ctx, stack, preset)
	if err != nil {
		return err
	}

	variables, err := w.askVariables(ctx, stack)
	if err != nil {
		return err
	}

	w.Result = WizardResult{
		Kind:      kind,
		Stack:     stack.ID,
		Name:      name,
		Features:  features,
		Variables: variables,
	}

	return nil
}

func availableKinds(m *manifest.Manifest) []string {
	kinds := make([]string, 0, len(m.Kinds))
	for _, name := range slices.Sorted(maps.Keys(m.Kinds)) {
		if m.Kinds[name].Max < 1 {
			continue
		}

		if len(stacksForKind(m, name)) == 0 {
			continue
		}

		kinds = append(kinds, name)
	}

	return kinds
}

func stacksForKind(m *manifest.Manifest, kind string) []manifest.Stack {
	stacks := make([]manifest.Stack, 0, len(m.Stacks))
	for _, stack := range m.Stacks {
		if stack.Kind == kind {
			stacks = append(stacks, stack)
		}
	}

	return stacks
}

func kindOptions(kinds []string) []huh.Option[string] {
	options := make([]huh.Option[string], 0, len(kinds))
	for _, kind := range kinds {
		label := strings.ToUpper(kind[:1]) + kind[1:]
		options = append(options, huh.NewOption(label, kind))
	}

	return options
}

func (w *Wizard) selectStack(ctx context.Context, kind string) (manifest.Stack, error) {
	stacks := stacksForKind(w.Manifest, kind)

	options := make([]huh.Option[string], 0, len(stacks))
	for _, stack := range stacks {
		options = append(options, huh.NewOption(stack.Name+" ("+stack.ID+")", stack.ID))
	}

	selected := w.Result.Stack
	if selected == "" {
		selected = stacks[0].ID
	}

	form := newForm(w, huh.NewGroup(
		huh.NewSelect[string]().
			Title("Which stack should it use?").
			Options(options...).
			Value(&selected),
	))
	if err := form.RunWithContext(ctx); err != nil {
		return manifest.Stack{}, fmt.Errorf("wizard: %w", err)
	}

	for _, stack := range stacks {
		if stack.ID == selected {
			return stack, nil
		}
	}

	return manifest.Stack{}, fmt.Errorf("wizard: unknown stack %q", selected)
}

func (w *Wizard) askName(ctx context.Context) (string, error) {
	name := w.Result.Name

	form := newForm(w, huh.NewGroup(
		huh.NewInput().
			Title("Project name").
			Placeholder("Acme App").
			Value(&name).
			Validate(nonEmpty("project name")),
	))
	if err := form.RunWithContext(ctx); err != nil {
		return "", fmt.Errorf("wizard: %w", err)
	}

	return strings.TrimSpace(name), nil
}

func (w *Wizard) selectPreset(
	ctx context.Context,
	kind string,
) (*manifest.Preset, error) {
	presets := presetsForKind(w.Manifest, kind)
	if len(presets) == 0 || len(w.Result.Features) > 0 {
		return nil, nil
	}

	selected := w.Result.Preset
	if !presetIDIn(selected, presets) {
		selected = ""
	}

	if selected == "" {
		if preset := w.Manifest.DefaultPresetForKind(kind); preset != nil {
			selected = preset.ID
		} else {
			selected = presets[0].ID
		}
	}

	options := make([]huh.Option[string], 0, len(presets))
	for _, preset := range presets {
		label := preset.Name
		if preset.Description != "" {
			label += " — " + preset.Description
		}

		options = append(options, huh.NewOption(label, preset.ID))
	}

	form := newForm(w, huh.NewGroup(
		huh.NewSelect[string]().
			Title("Which preset should it start from?").
			Description("You can adjust individual features next.").
			Options(options...).
			Value(&selected),
	))
	if err := form.RunWithContext(ctx); err != nil {
		return nil, fmt.Errorf("wizard: %w", err)
	}

	for index := range presets {
		if presets[index].ID == selected {
			return &presets[index], nil
		}
	}

	return nil, fmt.Errorf("wizard: unknown preset %q", selected)
}

func presetsForKind(m *manifest.Manifest, kind string) []manifest.Preset {
	presets := make([]manifest.Preset, 0, len(m.Presets))
	for _, preset := range m.Presets {
		if preset.AppliesToKind(kind) {
			presets = append(presets, preset)
		}
	}

	return presets
}

func presetIDIn(id string, presets []manifest.Preset) bool {
	for _, preset := range presets {
		if preset.ID == id {
			return true
		}
	}

	return false
}

func (w *Wizard) selectFeatures(
	ctx context.Context,
	stack manifest.Stack,
	preset *manifest.Preset,
) ([]string, error) {
	options := featureOptions(w.Manifest, stack)
	if len(options) == 0 {
		return make([]string, 0), nil
	}

	selected := slices.Clone(w.Result.Features)
	if len(selected) == 0 && preset != nil {
		selected = slices.Clone(preset.Features)
	}

	selected = filterFeatureValues(selected, options)

	form := newForm(w, huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title("Which features should be included?").
			Description("Required companions are added automatically.").
			Options(options...).
			Value(&selected),
	))
	if err := form.RunWithContext(ctx); err != nil {
		return nil, fmt.Errorf("wizard: %w", err)
	}

	return selected, nil
}

func featureOptions(m *manifest.Manifest, stack manifest.Stack) []huh.Option[string] {
	options := make([]huh.Option[string], 0, len(m.Features))

	for _, feature := range m.Features {
		if !supportsStack(feature, stack) {
			continue
		}

		if len(feature.Variants) == 0 {
			if feature.Overlays[stack.ID] == "" {
				continue
			}

			options = append(options, huh.NewOption(feature.Name, feature.ID))

			continue
		}

		for _, variant := range feature.Variants {
			if variant.Overlays[stack.ID] == "" {
				continue
			}

			options = append(options, huh.NewOption(
				feature.Name+" — "+variant.Name,
				feature.ID+":"+variant.ID,
			))
		}
	}

	return options
}

func filterFeatureValues(values []string, options []huh.Option[string]) []string {
	allowed := make(map[string]bool, len(options))
	for _, option := range options {
		allowed[option.Value] = true
	}

	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if allowed[value] {
			filtered = append(filtered, value)
		}
	}

	return filtered
}

func supportsStack(feature manifest.Feature, stack manifest.Stack) bool {
	if len(feature.Stacks) > 0 && !slices.Contains(feature.Stacks, stack.ID) {
		return false
	}

	return len(feature.Kinds) == 0 || slices.Contains(feature.Kinds, stack.Kind)
}

func (w *Wizard) askVariables(ctx context.Context, stack manifest.Stack) (map[string]string, error) {
	path := filepath.Join(
		w.RegistryDir,
		filepath.FromSlash(stack.Template),
		"template.yaml",
	)

	tmpl, err := manifest.LoadTemplateFile(path)
	if err != nil {
		return nil, fmt.Errorf("wizard: load template variables: %w", err)
	}

	if err := tmpl.Validate(); err != nil {
		return nil, fmt.Errorf("wizard: validate template variables: %w", err)
	}

	if len(tmpl.Variables) == 0 {
		return make(map[string]string), nil
	}

	values := make([]string, len(tmpl.Variables))
	fields := make([]huh.Field, 0, len(tmpl.Variables))

	for index, variable := range tmpl.Variables {
		values[index] = variable.Default
		if provided := w.Result.Variables[variable.Name]; provided != "" {
			if len(variable.Choices) == 0 || slices.Contains(variable.Choices, provided) {
				values[index] = provided
			}
		}

		if len(variable.Choices) > 0 {
			fields = append(fields, huh.NewSelect[string]().
				Title(variable.Prompt).
				Options(huh.NewOptions(variable.Choices...)...).
				Value(&values[index]))

			continue
		}

		fields = append(fields, huh.NewInput().
			Title(variable.Prompt).
			Placeholder(variable.Default).
			Value(&values[index]).
			Validate(nonEmpty(variable.Name)))
	}

	form := newForm(w, huh.NewGroup(fields...))
	if err := form.RunWithContext(ctx); err != nil {
		return nil, fmt.Errorf("wizard: %w", err)
	}

	resolved := make(map[string]string, len(tmpl.Variables))
	for index, variable := range tmpl.Variables {
		resolved[variable.Name] = strings.TrimSpace(values[index])
	}

	return resolved, nil
}

func newForm(w *Wizard, groups ...*huh.Group) *huh.Form {
	return huh.NewForm(groups...).
		WithTheme(huh.ThemeCharm()).
		WithInput(w.In).
		WithOutput(w.Out)
}

func nonEmpty(label string) func(string) error {
	return func(value string) error {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s must not be empty", label)
		}

		return nil
	}
}
