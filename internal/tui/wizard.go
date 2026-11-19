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

const fullstackKind = "fullstack"

// WizardResult holds the choices collected by the interactive wizard.
type WizardResult struct {
	Kind      string
	Stack     string
	Backend   string
	Frontend  string
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

type wizardVariable struct {
	name         string
	prompt       string
	defaultValue string
	choices      []string
	value        *string
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

	initialStack := w.Result.Stack
	if kind == fullstackKind && w.Result.Backend != "" {
		initialStack = w.Result.Backend
	}

	selectedStackID := initialStackID(w.Manifest, kind, initialStack)
	name := w.Result.Name
	presetID := initialPresetID(w.Manifest, kind, w.Result.Preset)
	explicitFeatures := len(w.Result.Features) > 0
	features := slices.Clone(w.Result.Features)

	if !explicitFeatures {
		features = presetFeatures(w.Manifest, presetID)
	}

	lastPresetID := presetID
	mediaProvider := currentVariant(features, "media-provider")
	compressionEnabled := slices.Contains(features, "compression:gzip")

	variableValues, err := loadWizardVariables(w.RegistryDir, w.Manifest, w.Result.Variables)
	if err != nil {
		return err
	}

	groups := w.basicGroups(kinds, &kind, &selectedStackID, &name)
	groups = append(groups, w.presetGroup(&kind, &presetID, &features, &lastPresetID, explicitFeatures))
	groups = append(groups, w.featureGroup(&kind, &selectedStackID, &features))
	groups = append(groups, w.mediaProviderGroup(&selectedStackID, &features, &mediaProvider))
	groups = append(groups, w.compressionGroup(&features, &compressionEnabled))
	groups = append(groups, w.variableGroups(selectableStacks(w.Manifest, kinds), variableValues, &selectedStackID)...)

	form := newForm(w, groups...)
	if err := form.RunWithContext(ctx); err != nil {
		return fmt.Errorf("wizard: %w", err)
	}

	stack, ok := findStackByID(w.Manifest, selectedStackID)
	if !ok {
		return fmt.Errorf("wizard: unknown stack %q", selectedStackID)
	}

	features = filterFeatureValues(features, wizardFeatureOptions(w.Manifest, kind, stack))
	if err := validateFeatureVariants(features); err != nil {
		return err
	}
	features = syncMediaProvider(w.Manifest, stack, features, mediaProvider)
	features = syncCompression(stack, features, compressionEnabled)

	variables := make(map[string]string, len(variableValues[selectedStackID]))
	for _, variable := range variableValues[selectedStackID] {
		variables[variable.name] = strings.TrimSpace(*variable.value)
	}

	result := WizardResult{
		Kind:      kind,
		Stack:     selectedStackID,
		Name:      strings.TrimSpace(name),
		Preset:    presetID,
		Features:  features,
		Variables: variables,
	}

	if kind == fullstackKind {
		frontend := w.Result.Frontend
		if frontend == "" {
			frontend = "next"
		}

		result.Backend = selectedStackID
		result.Frontend = frontend
		result.Stack = ""
	}

	w.Result = result

	return nil
}

func (w *Wizard) basicGroups(
	kinds []string,
	kind, selectedStackID, name *string,
) []*huh.Group {
	return []*huh.Group{
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("What kind of project is this?").
				Options(kindOptions(kinds)...).
				Value(kind),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Which stack should it use?").
				OptionsFunc(func() []huh.Option[string] {
					stacks := wizardStacksForKind(w.Manifest, *kind)
					if !stackIDIn(*selectedStackID, stacks) {
						*selectedStackID = stacks[0].ID
					}

					return stackOptions(stacks)
				}, kind).
				Value(selectedStackID),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Project name").
				Placeholder("Acme App").
				Value(name).
				Validate(nonEmpty("project name")).
				WithWidth(60),
		),
	}
}

func (w *Wizard) presetGroup(
	kind, presetID *string,
	features *[]string,
	lastPresetID *string,
	explicitFeatures bool,
) *huh.Group {
	return huh.NewGroup(
		huh.NewSelect[string]().
			Title("Which preset should it start from?").
			Description("You can adjust individual features next. Shift+Tab goes back.").
			OptionsFunc(func() []huh.Option[string] {
				presets := presetsForKind(w.Manifest, *kind)
				if !presetIDIn(*presetID, presets) {
					*presetID = defaultPresetID(w.Manifest, *kind)
				}

				if !explicitFeatures && *presetID != *lastPresetID {
					*features = presetFeatures(w.Manifest, *presetID)
					*lastPresetID = *presetID
				}

				return presetOptions(presets)
			}, kind).
			Value(presetID),
	).WithHideFunc(func() bool {
		return explicitFeatures || len(presetsForKind(w.Manifest, *kind)) == 0
	})
}

func (w *Wizard) featureGroup(kind, selectedStackID *string, features *[]string) *huh.Group {
	return huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title("Which features should be included?").
			Description("Required companions are added automatically.").
			OptionsFunc(func() []huh.Option[string] {
				stack, ok := findStackByID(w.Manifest, *selectedStackID)
				if !ok {
					return nil
				}

				options := wizardFeatureOptions(w.Manifest, *kind, stack)
				*features = filterFeatureValues(*features, options)

				return options
			}, selectedStackID).
			Value(features).
			Validate(validateFeatureVariants),
	).WithHideFunc(func() bool {
		stack, ok := findStackByID(w.Manifest, *selectedStackID)

		return !ok || len(wizardFeatureOptions(w.Manifest, *kind, stack)) == 0
	})
}

func (w *Wizard) mediaProviderGroup(
	selectedStackID *string,
	features *[]string,
	mediaProvider *string,
) *huh.Group {
	return huh.NewGroup(
		huh.NewSelect[string]().
			Title("Which media provider should it use?").
			Description("Choose storage and transformation infrastructure.").
			OptionsFunc(func() []huh.Option[string] {
				stack, ok := findStackByID(w.Manifest, *selectedStackID)
				if !ok {
					return nil
				}

				return variantOptions(w.Manifest, "media-provider", stack)
			}, features).
			Value(mediaProvider),
	).WithHideFunc(func() bool {
		stack, ok := findStackByID(w.Manifest, *selectedStackID)

		return !ok || !slices.Contains(*features, "media") || len(variantOptions(w.Manifest, "media-provider", stack)) == 0
	})
}

func (w *Wizard) compressionGroup(features *[]string, enabled *bool) *huh.Group {
	return huh.NewGroup(
		huh.NewConfirm().
			Title("Enable gzip queue compression?").
			Description("Only available with the RabbitMQ queue.").
			Value(enabled),
	).WithHideFunc(func() bool {
		return !slices.Contains(*features, "queue:rabbitmq")
	})
}

func (w *Wizard) variableGroups(
	stacks []manifest.Stack,
	variableValues map[string][]wizardVariable,
	selectedStackID *string,
) []*huh.Group {
	groups := make([]*huh.Group, 0, len(stacks))

	for _, stack := range stacks {
		variables := variableValues[stack.ID]
		if len(variables) == 0 {
			continue
		}

		fields := make([]huh.Field, 0, len(variables))

		for index := range variables {
			variable := &variables[index]
			if len(variable.choices) > 0 {
				fields = append(fields, huh.NewSelect[string]().
					Title(variable.prompt).
					Options(huh.NewOptions(variable.choices...)...).
					Value(variable.value))

				continue
			}

			fields = append(fields, huh.NewInput().
				Title(variable.prompt).
				Placeholder(variable.defaultValue).
				Value(variable.value).
				Validate(nonEmpty(variable.name)).
				WithWidth(60))
		}

		groupStackID := stack.ID

		groups = append(groups, huh.NewGroup(fields...).WithHideFunc(func() bool {
			return *selectedStackID != groupStackID
		}))
	}

	return groups
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

func selectableStacks(m *manifest.Manifest, kinds []string) []manifest.Stack {
	stacks := make([]manifest.Stack, 0, len(m.Stacks))
	for _, stack := range m.Stacks {
		if slices.Contains(kinds, stack.Kind) || (slices.Contains(kinds, "fullstack") && (stack.Kind == "backend" || stack.Kind == "frontend")) {
			stacks = append(stacks, stack)
		}
	}

	return stacks
}

func wizardStacksForKind(m *manifest.Manifest, kind string) []manifest.Stack {
	if kind == fullstackKind {
		return stacksForKind(m, "backend")
	}

	return stacksForKind(m, kind)
}

func wizardFeatureOptions(m *manifest.Manifest, kind string, stack manifest.Stack) []huh.Option[string] {
	if kind != fullstackKind {
		return featureOptions(m, stack)
	}

	options := append([]huh.Option[string]{}, featureOptions(m, stack)...)

	seen := make(map[string]bool, len(options))
	for _, option := range options {
		seen[option.Value] = true
	}

	if frontend, ok := findStackByID(m, "next"); ok {
		for _, option := range featureOptions(m, frontend) {
			if option.Value == "auth:session" || seen[option.Value] {
				continue
			}

			options = append(options, option)
			seen[option.Value] = true
		}
	}

	return options
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

func initialStackID(m *manifest.Manifest, kind, selected string) string {
	stacks := wizardStacksForKind(m, kind)
	if len(stacks) == 0 {
		return ""
	}

	if stackIDIn(selected, stacks) {
		return selected
	}

	return stacks[0].ID
}

func stackIDIn(id string, stacks []manifest.Stack) bool {
	for _, stack := range stacks {
		if stack.ID == id {
			return true
		}
	}

	return false
}

func findStackByID(m *manifest.Manifest, id string) (manifest.Stack, bool) {
	for _, stack := range m.Stacks {
		if stack.ID == id {
			return stack, true
		}
	}

	return manifest.Stack{}, false
}

func stackOptions(stacks []manifest.Stack) []huh.Option[string] {
	options := make([]huh.Option[string], 0, len(stacks))
	for _, stack := range stacks {
		options = append(options, huh.NewOption(stack.Name+" ("+stack.ID+")", stack.ID))
	}

	return options
}

func kindOptions(kinds []string) []huh.Option[string] {
	options := make([]huh.Option[string], 0, len(kinds))
	for _, kind := range kinds {
		label := strings.ToUpper(kind[:1]) + kind[1:]
		options = append(options, huh.NewOption(label, kind))
	}

	return options
}

func initialPresetID(m *manifest.Manifest, kind, selected string) string {
	presets := presetsForKind(m, kind)
	if presetIDIn(selected, presets) {
		return selected
	}

	return defaultPresetID(m, kind)
}

func defaultPresetID(m *manifest.Manifest, kind string) string {
	if preset := m.DefaultPresetForKind(kind); preset != nil {
		return preset.ID
	}

	presets := presetsForKind(m, kind)
	if len(presets) == 0 {
		return ""
	}

	return presets[0].ID
}

func presetOptions(presets []manifest.Preset) []huh.Option[string] {
	options := make([]huh.Option[string], 0, len(presets))
	for _, preset := range presets {
		label := preset.Name
		if preset.Description != "" {
			label += " — " + preset.Description
		}

		options = append(options, huh.NewOption(label, preset.ID))
	}

	return options
}

func presetFeatures(m *manifest.Manifest, presetID string) []string {
	preset := m.FindPreset(presetID)
	if preset == nil {
		return make([]string, 0)
	}

	return slices.Clone(preset.Features)
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

func featureOptions(m *manifest.Manifest, stack manifest.Stack) []huh.Option[string] {
	options := allFeatureOptions(m, stack)
	filtered := make([]huh.Option[string], 0, len(options))

	for _, option := range options {
		featureID, _, _ := strings.Cut(option.Value, ":")
		if featureID == "media-provider" || featureID == "compression" {
			continue
		}

		filtered = append(filtered, option)
	}

	return filtered
}

func allFeatureOptions(m *manifest.Manifest, stack manifest.Stack) []huh.Option[string] {
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

func validateFeatureVariants(values []string) error {
	seen := make(map[string]string, len(values))
	for _, value := range values {
		featureID, variantID, _ := strings.Cut(value, ":")
		if variantID == "" {
			continue
		}
		if previous, ok := seen[featureID]; ok && previous != variantID {
			return fmt.Errorf("feature %q was selected as both %q and %q; choose one variant", featureID, previous, variantID)
		}
		seen[featureID] = variantID
	}

	return nil
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

func variantOptions(
	m *manifest.Manifest,
	featureID string,
	stack manifest.Stack,
) []huh.Option[string] {
	for _, feature := range m.Features {
		if feature.ID != featureID || !supportsStack(feature, stack) {
			continue
		}

		options := make([]huh.Option[string], 0, len(feature.Variants))
		for _, variant := range feature.Variants {
			if variant.Overlays[stack.ID] == "" {
				continue
			}

			options = append(options, huh.NewOption(variant.Name, variant.ID))
		}

		return options
	}

	return nil
}

func currentVariant(features []string, featureID string) string {
	for _, value := range features {
		id, variant, ok := strings.Cut(value, ":")
		if ok && id == featureID {
			return variant
		}
	}

	return ""
}

func syncMediaProvider(
	m *manifest.Manifest,
	stack manifest.Stack,
	features []string,
	provider string,
) []string {
	filtered := make([]string, 0, len(features))
	for _, value := range features {
		if !strings.HasPrefix(value, "media-provider:") {
			filtered = append(filtered, value)
		}
	}

	if !slices.Contains(filtered, "media") {
		return filtered
	}

	options := variantOptions(m, "media-provider", stack)
	if len(options) == 0 {
		return filtered
	}

	chosen := provider
	if chosen == "" {
		chosen = options[0].Value
	}

	return append(filtered, "media-provider:"+chosen)
}

func syncCompression(stack manifest.Stack, features []string, enabled bool) []string {
	filtered := make([]string, 0, len(features))
	for _, value := range features {
		if value != "compression:gzip" {
			filtered = append(filtered, value)
		}
	}

	if enabled && slices.Contains(features, "queue:rabbitmq") && supportsStack(manifest.Feature{
		ID: "compression",
	}, stack) {
		filtered = append(filtered, "compression:gzip")
	}

	return filtered
}

func loadWizardVariables(
	registryDir string,
	m *manifest.Manifest,
	provided map[string]string,
) (map[string][]wizardVariable, error) {
	result := make(map[string][]wizardVariable, len(m.Stacks))

	for _, stack := range m.Stacks {
		path := filepath.Join(
			registryDir,
			filepath.FromSlash(stack.Template),
			"template.yaml",
		)

		tmpl, err := manifest.LoadTemplateFile(path)
		if err != nil {
			return nil, fmt.Errorf("wizard: load template variables for %q: %w", stack.ID, err)
		}

		if err := tmpl.Validate(); err != nil {
			return nil, fmt.Errorf("wizard: validate template variables for %q: %w", stack.ID, err)
		}

		variables := make([]wizardVariable, 0, len(tmpl.Variables))
		for _, variable := range tmpl.Variables {
			value := variable.Default
			if providedValue := provided[variable.Name]; providedValue != "" {
				if len(variable.Choices) == 0 || slices.Contains(variable.Choices, providedValue) {
					value = providedValue
				}
			}

			valuePointer := new(string)
			*valuePointer = value

			variables = append(variables, wizardVariable{
				name:         variable.Name,
				prompt:       variable.Prompt,
				defaultValue: variable.Default,
				choices:      variable.Choices,
				value:        valuePointer,
			})
		}

		result[stack.ID] = variables
	}

	return result, nil
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
