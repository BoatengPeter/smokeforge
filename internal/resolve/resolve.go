// Package resolve validates a project selection against a registry manifest
// and builds an ordered, deterministic generation plan.
package resolve

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BoatengPeter/smokeforge/internal/manifest"
	"github.com/BoatengPeter/smokeforge/internal/patch"
)

// Selection is the user's project choice before validation.
type Selection struct {
	Name      string
	Kind      string
	Stack     string
	Backend   string
	Frontend  string
	Desktop   string
	Preset    string
	Features  []string
	Variables map[string]string
}

// Plan is a validated, ordered description of one generated target.
type Plan struct {
	Name    string
	Kind    string
	Target  TargetPlan
	Targets []TargetPlan
	Root    *TargetPlan
}

// TargetPlan describes the base template and overlays for one target.
type TargetPlan struct {
	Path        string
	Stack       manifest.Stack
	Template    manifest.Template
	TemplateDir string
	Overlays    []OverlayPlan
	Variables   map[string]string
}

// OverlayPlan binds a selected feature variant to a loaded overlay.
type OverlayPlan struct {
	FeatureID string
	VariantID string
	Path      string
	Dir       string
	Overlay   *patch.Overlay
}

type selection struct {
	feature     manifest.Feature
	variant     *manifest.Variant
	variantID   string
	overlayPath string
}

// Resolve validates the selection and returns an ordered plan. It fails before
// any filesystem mutation.
func Resolve(m *manifest.Manifest, registryDir string, sel Selection) (*Plan, error) {
	if strings.TrimSpace(sel.Name) == "" {
		return nil, errors.New("project name is required")
	}

	kind, ok := m.Kinds[sel.Kind]
	if !ok {
		return nil, fmt.Errorf("unknown kind %q", sel.Kind)
	}

	if kind.Max < 1 {
		return nil, fmt.Errorf("kind %q does not allow targets", sel.Kind)
	}

	if sel.Kind == "fullstack" {
		return resolveFullstack(m, registryDir, sel)
	}

	stack, err := findStack(m, sel)
	if err != nil {
		return nil, err
	}

	target, err := resolveTarget(m, registryDir, stack, sel, ".", sel.Variables)
	if err != nil {
		return nil, err
	}

	return &Plan{
		Name:    sel.Name,
		Kind:    sel.Kind,
		Target:  target,
		Targets: []TargetPlan{target},
	}, nil
}

func resolveFullstack(m *manifest.Manifest, registryDir string, sel Selection) (*Plan, error) {
	stacks, err := fullstackStacks(m, sel)
	if err != nil {
		return nil, err
	}

	provided, err := fullstackVariables(stacks.backend, stacks.frontend, stacks.root, stacks.desktop, sel.Variables)
	if err != nil {
		return nil, err
	}

	requested, err := requestedFeatures(m, sel)
	if err != nil {
		return nil, err
	}

	apiFeatures, webFeatures, desktopFeatures, rootFeatures, err := routeFullstackFeatures(
		m,
		requested,
		stacks.backend,
		stacks.frontend,
		stacks.desktop,
		stacks.root,
	)
	if err != nil {
		return nil, err
	}

	webFeatures = fullstackWebFeatures(apiFeatures, webFeatures)
	desktopFeatures = fullstackDesktopFeatures(apiFeatures, desktopFeatures, stacks.desktop != nil)
	apiFeatures = fullstackSyncAPIFeatures(apiFeatures, stacks.desktop != nil)

	if err := validateFullstackFeatures(apiFeatures, rootFeatures); err != nil {
		return nil, err
	}

	return buildFullstackPlan(m, registryDir, sel, stacks, provided, apiFeatures, webFeatures, desktopFeatures, rootFeatures)
}

type fullstackStackSet struct {
	backend  manifest.Stack
	frontend manifest.Stack
	desktop  *manifest.Stack
	root     manifest.Stack
}

func fullstackStacks(m *manifest.Manifest, sel Selection) (fullstackStackSet, error) {
	backendID := strings.TrimSpace(sel.Backend)
	if backendID == "" {
		backendID = strings.TrimSpace(sel.Stack)
	}

	if backendID == "" {
		return fullstackStackSet{}, errors.New("backend stack is required (use --backend)")
	}

	backend, err := findStackByID(m, backendID, "backend")
	if err != nil {
		return fullstackStackSet{}, err
	}

	frontendID := strings.TrimSpace(sel.Frontend)
	if frontendID == "" {
		frontendID = "next"
	}

	frontend, err := findStackByID(m, frontendID, "frontend")
	if err != nil {
		return fullstackStackSet{}, err
	}

	var desktop *manifest.Stack

	if strings.TrimSpace(sel.Desktop) != "" {
		desktopID := strings.TrimSpace(sel.Desktop)
		if !strings.Contains(desktopID, "/") {
			desktopID = "desktop/" + desktopID
		}

		found, findErr := findStackByID(m, desktopID, "desktop")
		if findErr != nil {
			return fullstackStackSet{}, findErr
		}

		desktop = &found
	}

	root, err := findStackByID(m, "workspace/fullstack", "fullstack")
	if err != nil {
		return fullstackStackSet{}, fmt.Errorf("fullstack workspace stack: %w", err)
	}

	return fullstackStackSet{backend: backend, frontend: frontend, desktop: desktop, root: root}, nil
}

func fullstackWebFeatures(apiFeatures, webFeatures []string) []string {
	if featureSelected(apiFeatures, "auth", "jwt") && !featureSelected(webFeatures, "auth", "bff") {
		return append([]string{"auth:bff"}, webFeatures...)
	}

	return webFeatures
}

func fullstackDesktopFeatures(apiFeatures, desktopFeatures []string, hasDesktop bool) []string {
	if !hasDesktop {
		return desktopFeatures
	}

	if featureSelected(apiFeatures, "auth", "jwt") && !featureSelected(desktopFeatures, "auth", "electron") {
		desktopFeatures = append([]string{"auth:electron"}, desktopFeatures...)
	}

	if !featureSelected(desktopFeatures, "state", "zustand") {
		desktopFeatures = append(desktopFeatures, "state:zustand")
	}

	if !featureSelected(desktopFeatures, "storage", "sqlite") {
		desktopFeatures = append(desktopFeatures, "storage:sqlite")
	}

	if featureSelected(apiFeatures, "database", "postgres") && featureSelected(apiFeatures, "auth", "jwt") && !featureSelected(desktopFeatures, "sync", "library-client") {
		desktopFeatures = append(desktopFeatures, "sync:library-client")
	}

	return desktopFeatures
}

func fullstackSyncAPIFeatures(apiFeatures []string, hasDesktop bool) []string {
	if !hasDesktop {
		return apiFeatures
	}

	if featureSelected(apiFeatures, "database", "postgres") && featureSelected(apiFeatures, "auth", "jwt") && !featureSelected(apiFeatures, "sync", "library") {
		return append(apiFeatures, "sync:library")
	}

	return apiFeatures
}

func validateFullstackFeatures(apiFeatures, rootFeatures []string) error {
	if slices.Contains(rootFeatures, "docker") && !featureSelected(apiFeatures, "database", "") {
		return errors.New("fullstack docker requires database:postgres for the API")
	}

	return nil
}

func buildFullstackPlan(
	m *manifest.Manifest,
	registryDir string,
	sel Selection,
	stacks fullstackStackSet,
	provided map[string]string,
	apiFeatures, webFeatures, desktopFeatures, rootFeatures []string,
) (*Plan, error) {
	api, err := resolveTargetWithFeatures(m, registryDir, stacks.backend, "apps/api", provided, apiFeatures, true)
	if err != nil {
		return nil, err
	}

	web, err := resolveTargetWithFeatures(m, registryDir, stacks.frontend, "apps/web", provided, webFeatures, true)
	if err != nil {
		return nil, err
	}

	targets := []TargetPlan{api, web}

	if stacks.desktop != nil {
		desktop, resolveErr := resolveTargetWithFeatures(m, registryDir, *stacks.desktop, "apps/desktop", provided, desktopFeatures, true)
		if resolveErr != nil {
			return nil, resolveErr
		}

		targets = append(targets, desktop)
	}

	root, err := resolveTargetWithFeatures(m, registryDir, stacks.root, ".", provided, rootFeatures, true)
	if err != nil {
		return nil, err
	}

	if stacks.desktop != nil {
		desktopOverlay, overlayErr := loadRootOverlay(registryDir, "overlays/workspace/fullstack/desktop", "desktop")
		if overlayErr != nil {
			return nil, overlayErr
		}

		root.Overlays = append(root.Overlays, desktopOverlay)
	}

	return &Plan{
		Name:    sel.Name,
		Kind:    sel.Kind,
		Target:  api,
		Targets: targets,
		Root:    &root,
	}, nil
}

func loadRootOverlay(registryDir, overlayPath, featureID string) (OverlayPlan, error) {
	dir := filepath.Join(registryDir, filepath.FromSlash(overlayPath))

	overlay, err := patch.LoadOverlayFile(filepath.Join(dir, "overlay.yaml"))
	if err != nil {
		return OverlayPlan{}, fmt.Errorf("load workspace overlay %q: %w", overlayPath, err)
	}

	if err := overlay.Validate(); err != nil {
		return OverlayPlan{}, fmt.Errorf("validate workspace overlay %q: %w", overlayPath, err)
	}

	return OverlayPlan{FeatureID: featureID, Path: overlayPath, Dir: dir, Overlay: overlay}, nil
}

func findStackByID(m *manifest.Manifest, id, kind string) (manifest.Stack, error) {
	for _, stack := range m.Stacks {
		if stack.ID != id {
			continue
		}

		if stack.Kind != kind {
			return manifest.Stack{}, fmt.Errorf(
				"stack %q has kind %q, want %q",
				stack.ID,
				stack.Kind,
				kind,
			)
		}

		return stack, nil
	}

	return manifest.Stack{}, fmt.Errorf("unknown %s stack %q", kind, id)
}

func fullstackVariables(
	backend, frontend, root manifest.Stack,
	desktop *manifest.Stack,
	provided map[string]string,
) (map[string]string, error) {
	declared := make(map[string]bool)

	stacks := []manifest.Stack{backend, frontend, root}
	if desktop != nil {
		stacks = append(stacks, *desktop)
	}

	for _, stack := range stacks {
		for _, name := range stack.Variables {
			declared[name] = true
		}
	}

	for name := range provided {
		if !declared[name] {
			return nil, fmt.Errorf("unknown variable %q", name)
		}
	}

	return provided, nil
}

func resolveTarget(
	m *manifest.Manifest,
	registryDir string,
	stack manifest.Stack,
	sel Selection,
	path string,
	provided map[string]string,
) (TargetPlan, error) {
	requested, err := requestedFeatures(m, sel)
	if err != nil {
		return TargetPlan{}, err
	}

	return resolveTargetWithFeatures(m, registryDir, stack, path, provided, requested, false)
}

func resolveTargetWithFeatures(
	m *manifest.Manifest,
	registryDir string,
	stack manifest.Stack,
	path string,
	provided map[string]string,
	requested []string,
	filterProvided bool,
) (TargetPlan, error) {
	tmpl, err := loadTemplate(registryDir, stack)
	if err != nil {
		return TargetPlan{}, err
	}

	filtered := provided
	if filterProvided {
		filtered = filterVariables(stack, provided)
	}

	variables, err := Variables(stack, tmpl, filtered)
	if err != nil {
		return TargetPlan{}, err
	}

	selections, err := selectFeatures(m, stack, requested)
	if err != nil {
		return TargetPlan{}, err
	}

	selections, err = expandRequirements(m, stack, selections)
	if err != nil {
		return TargetPlan{}, err
	}

	if err := validateRelationships(selections); err != nil {
		return TargetPlan{}, err
	}

	ordered, err := orderFeatures(selections)
	if err != nil {
		return TargetPlan{}, err
	}

	overlays, err := buildOverlays(registryDir, stack, ordered)
	if err != nil {
		return TargetPlan{}, err
	}

	return TargetPlan{
		Path:        path,
		Stack:       stack,
		Template:    tmpl,
		TemplateDir: filepath.Join(registryDir, filepath.FromSlash(stack.Template)),
		Overlays:    overlays,
		Variables:   variables,
	}, nil
}

func filterVariables(stack manifest.Stack, provided map[string]string) map[string]string {
	filtered := make(map[string]string, len(provided))

	for name, value := range provided {
		if slices.Contains(stack.Variables, name) {
			filtered[name] = value
		}
	}

	return filtered
}

func routeFullstackFeatures(
	m *manifest.Manifest,
	requested []string,
	backend, frontend manifest.Stack,
	desktop *manifest.Stack,
	root manifest.Stack,
) (api, web, desktopFeatures, rootFeatures []string, err error) {
	for _, raw := range requested {
		featureID, variantID, _ := strings.Cut(raw, ":")

		feature, ok := findFeature(m, featureID)
		if !ok {
			return nil, nil, nil, nil, fmt.Errorf("unknown feature %q", featureID)
		}

		apiOK := stackSupportsFeature(feature, variantID, backend)
		webOK := stackSupportsFeature(feature, variantID, frontend)
		desktopOK := desktop != nil && stackSupportsFeature(feature, variantID, *desktop)
		rootOK := stackSupportsFeature(feature, variantID, root)

		// Persistence belongs to the API in a fullstack project. The web app
		// reaches it through the BFF, not with its own database overlay.
		if feature.ID == "database" {
			if !apiOK {
				return nil, nil, nil, nil, fmt.Errorf("feature %q requires a backend stack", raw)
			}

			api = append(api, raw)

			continue
		}

		// Deployment features are owned by the workspace root; Docker also
		// installs the per-target Dockerfiles used by the root compose file.
		if rootOK && feature.ID == "ci" {
			rootFeatures = append(rootFeatures, raw)
			continue
		}

		routed := false

		if apiOK {
			api = append(api, raw)
			routed = true
		}

		if webOK {
			web = append(web, raw)
			routed = true
		}

		if desktopOK {
			desktopFeatures = append(desktopFeatures, raw)
			routed = true
		}

		if rootOK {
			rootFeatures = append(rootFeatures, raw)
			routed = true
		}

		if !routed {
			return nil, nil, nil, nil, fmt.Errorf(
				"feature %q does not support the fullstack API, web, desktop, or workspace stack",
				raw,
			)
		}
	}

	return api, web, desktopFeatures, rootFeatures, nil
}

func stackSupportsFeature(feature manifest.Feature, variantID string, stack manifest.Stack) bool {
	if len(feature.Kinds) > 0 && !slices.Contains(feature.Kinds, stack.Kind) {
		return false
	}

	if len(feature.Stacks) > 0 && !slices.Contains(feature.Stacks, stack.ID) {
		return false
	}

	if len(feature.Variants) == 0 {
		return strings.TrimSpace(feature.Overlays[stack.ID]) != ""
	}

	for _, variant := range feature.Variants {
		if variantID != "" && variant.ID != variantID {
			continue
		}

		if variantID == "" && !variant.Default {
			continue
		}

		if variantSupportsStack(variant, stack) {
			return true
		}
	}

	return false
}

func featureSelected(refs []string, featureID, variantID string) bool {
	for _, ref := range refs {
		id, variant, _ := strings.Cut(ref, ":")
		if id == featureID && (variantID == "" || variant == variantID) {
			return true
		}
	}

	return false
}

func findStack(m *manifest.Manifest, sel Selection) (manifest.Stack, error) {
	if strings.TrimSpace(sel.Stack) == "" {
		return manifest.Stack{}, fmt.Errorf("stack is required for kind %q", sel.Kind)
	}

	for _, stack := range m.Stacks {
		if stack.ID != sel.Stack {
			continue
		}

		if stack.Kind != sel.Kind {
			return manifest.Stack{}, fmt.Errorf(
				"stack %q has kind %q, want %q",
				stack.ID,
				stack.Kind,
				sel.Kind,
			)
		}

		return stack, nil
	}

	return manifest.Stack{}, fmt.Errorf("unknown stack %q", sel.Stack)
}

func loadTemplate(registryDir string, stack manifest.Stack) (manifest.Template, error) {
	path := filepath.Join(
		registryDir,
		filepath.FromSlash(stack.Template),
		"template.yaml",
	)

	tmpl, err := manifest.LoadTemplateFile(path)
	if err != nil {
		return manifest.Template{}, fmt.Errorf("load template for stack %q: %w", stack.ID, err)
	}

	if err := tmpl.Validate(); err != nil {
		return manifest.Template{}, fmt.Errorf("validate template for stack %q: %w", stack.ID, err)
	}

	if tmpl.ID != stack.ID {
		return manifest.Template{}, fmt.Errorf(
			"template id %q does not match stack %q",
			tmpl.ID,
			stack.ID,
		)
	}

	return *tmpl, nil
}

// Variables merges provided values with template defaults and rejects
// unknown or missing variables.
func Variables(
	stack manifest.Stack,
	tmpl manifest.Template,
	provided map[string]string,
) (map[string]string, error) {
	defaults, choices := templateVariables(tmpl)

	if err := validateDeclaredVariables(stack, defaults); err != nil {
		return nil, err
	}

	declared := variableSet(stack)
	if err := validateProvidedVariables(provided, declared); err != nil {
		return nil, err
	}

	return resolveVariableValues(stack, defaults, choices, provided)
}

func templateVariables(tmpl manifest.Template) (map[string]string, map[string][]string) {
	defaults := make(map[string]string, len(tmpl.Variables))
	choices := make(map[string][]string, len(tmpl.Variables))

	for _, variable := range tmpl.Variables {
		defaults[variable.Name] = variable.Default
		choices[variable.Name] = variable.Choices
	}

	return defaults, choices
}

func variableSet(stack manifest.Stack) map[string]bool {
	declared := make(map[string]bool, len(stack.Variables))
	for _, name := range stack.Variables {
		declared[name] = true
	}

	return declared
}

func validateDeclaredVariables(stack manifest.Stack, defaults map[string]string) error {
	for _, name := range stack.Variables {
		if _, ok := defaults[name]; !ok {
			return fmt.Errorf("variables mismatch: stack %q declares %q", stack.ID, name)
		}
	}

	for name := range defaults {
		if !slices.Contains(stack.Variables, name) {
			return fmt.Errorf(
				"variables mismatch: template declares undeclared variable %q",
				name,
			)
		}
	}

	return nil
}

func validateProvidedVariables(provided map[string]string, declared map[string]bool) error {
	for name := range provided {
		if !declared[name] {
			return fmt.Errorf("unknown variable %q", name)
		}
	}

	return nil
}

func resolveVariableValues(
	stack manifest.Stack,
	defaults map[string]string,
	choices map[string][]string,
	provided map[string]string,
) (map[string]string, error) {
	values := make(map[string]string, len(stack.Variables))

	missing := make([]string, 0, len(stack.Variables))

	for _, name := range stack.Variables {
		value, ok := provided[name]
		if !ok || value == "" {
			value = defaults[name]
		}

		if value == "" {
			missing = append(missing, name)

			continue
		}

		if len(choices[name]) > 0 && !slices.Contains(choices[name], value) {
			return nil, fmt.Errorf(
				"variable %q must be one of %s",
				name,
				strings.Join(choices[name], ", "),
			)
		}

		values[name] = value
	}

	if len(missing) > 0 {
		slices.Sort(missing)

		return nil, fmt.Errorf("missing required variable(s): %s", strings.Join(missing, ", "))
	}

	return values, nil
}

func requestedFeatures(m *manifest.Manifest, sel Selection) ([]string, error) {
	if strings.TrimSpace(sel.Preset) == "" {
		return sel.Features, nil
	}

	preset := m.FindPreset(sel.Preset)
	if preset == nil {
		return nil, fmt.Errorf("unknown preset %q", sel.Preset)
	}

	if !preset.AppliesToKind(sel.Kind) {
		valid := make([]string, 0, len(m.Presets))
		for _, candidate := range m.Presets {
			if candidate.AppliesToKind(sel.Kind) {
				valid = append(valid, candidate.ID)
			}
		}

		return nil, fmt.Errorf(
			"preset %q does not support kind %q (use one of: %s)",
			sel.Preset,
			sel.Kind,
			strings.Join(valid, ", "),
		)
	}

	features := make([]string, 0, len(preset.Features)+len(sel.Features))
	features = append(features, preset.Features...)
	features = append(features, sel.Features...)

	return features, nil
}

func selectFeatures(
	m *manifest.Manifest,
	stack manifest.Stack,
	requested []string,
) ([]*selection, error) {
	selected := make(map[string]*selection, len(requested))
	ordered := make([]*selection, 0, len(requested))

	for _, raw := range requested {
		entry, err := buildSelection(m, stack, raw)
		if err != nil {
			return nil, err
		}

		if _, exists := selected[entry.feature.ID]; exists {
			return nil, fmt.Errorf("duplicate feature %q", entry.feature.ID)
		}

		selected[entry.feature.ID] = entry
		ordered = append(ordered, entry)
	}

	return ordered, nil
}

func buildSelection(
	m *manifest.Manifest,
	stack manifest.Stack,
	raw string,
) (*selection, error) {
	featureID, variantID, _ := strings.Cut(raw, ":")
	if strings.TrimSpace(featureID) == "" {
		return nil, fmt.Errorf("invalid feature selection %q", raw)
	}

	feature, ok := findFeature(m, featureID)
	if !ok {
		return nil, fmt.Errorf("unknown feature %q", featureID)
	}

	if err := ensureSupported(feature, stack); err != nil {
		return nil, err
	}

	variant, resolvedVariantID, err := selectVariant(feature, stack, variantID)
	if err != nil {
		return nil, err
	}

	overlayPath := feature.Overlays[stack.ID]
	if variant != nil {
		overlayPath = variant.Overlays[stack.ID]
	}

	if strings.TrimSpace(overlayPath) == "" {
		return nil, fmt.Errorf(
			"feature %q has no overlay for stack %q",
			feature.ID,
			stack.ID,
		)
	}

	return &selection{
		feature:     feature,
		variant:     variant,
		variantID:   resolvedVariantID,
		overlayPath: overlayPath,
	}, nil
}

func expandRequirements(
	m *manifest.Manifest,
	stack manifest.Stack,
	selections []*selection,
) ([]*selection, error) {
	byID := make(map[string]*selection, len(selections))
	for _, entry := range selections {
		byID[entry.feature.ID] = entry
	}

	for {
		added := false

		for _, entry := range slices.Clone(selections) {
			for _, ref := range requirements(entry) {
				id, variantID := splitReference(ref)

				if existing, ok := byID[id]; ok {
					if variantID != "" && existing.variantID != variantID {
						return nil, fmt.Errorf(
							"feature %q requires %q but %q is selected",
							entry.feature.ID,
							ref,
							existing.variantID,
						)
					}

					continue
				}

				required, err := buildSelection(m, stack, ref)
				if err != nil {
					return nil, fmt.Errorf("feature %q requires %q: %w", entry.feature.ID, ref, err)
				}

				selections = append(selections, required)
				byID[id] = required
				added = true
			}
		}

		if !added {
			return selections, nil
		}
	}
}

func findFeature(m *manifest.Manifest, id string) (manifest.Feature, bool) {
	for _, feature := range m.Features {
		if feature.ID == id {
			return feature, true
		}
	}

	return manifest.Feature{}, false
}

func ensureSupported(feature manifest.Feature, stack manifest.Stack) error {
	if len(feature.Stacks) > 0 && !slices.Contains(feature.Stacks, stack.ID) {
		return fmt.Errorf("feature %q does not support stack %q", feature.ID, stack.ID)
	}

	if len(feature.Kinds) > 0 && !slices.Contains(feature.Kinds, stack.Kind) {
		return fmt.Errorf(
			"feature %q does not support kind %q",
			feature.ID,
			stack.Kind,
		)
	}

	return nil
}

func selectVariant(
	feature manifest.Feature,
	stack manifest.Stack,
	variantID string,
) (*manifest.Variant, string, error) {
	if len(feature.Variants) == 0 {
		if variantID != "" {
			return nil, "", fmt.Errorf("feature %q has no variants", feature.ID)
		}

		return nil, "", nil
	}

	compatible := make([]*manifest.Variant, 0, len(feature.Variants))
	for index := range feature.Variants {
		if variantSupportsStack(feature.Variants[index], stack) {
			compatible = append(compatible, &feature.Variants[index])
		}
	}

	if variantID != "" {
		for index := range feature.Variants {
			if feature.Variants[index].ID != variantID {
				continue
			}

			if !variantSupportsStack(feature.Variants[index], stack) {
				return nil, "", fmt.Errorf(
					"feature %q: variant %q does not support stack %q",
					feature.ID,
					variantID,
					stack.ID,
				)
			}

			return &feature.Variants[index], variantID, nil
		}

		return nil, "", fmt.Errorf(
			"feature %q has no variant %q",
			feature.ID,
			variantID,
		)
	}

	for _, variant := range compatible {
		if variant.Default {
			return variant, variant.ID, nil
		}
	}

	switch len(compatible) {
	case 1:
		return compatible[0], compatible[0].ID, nil
	case 0:
		return nil, "", fmt.Errorf(
			"feature %q has no variant for stack %q",
			feature.ID,
			stack.ID,
		)
	default:
		return nil, "", fmt.Errorf(
			"feature %q requires an explicit variant for stack %q",
			feature.ID,
			stack.ID,
		)
	}
}

func variantSupportsStack(variant manifest.Variant, stack manifest.Stack) bool {
	if len(variant.Stacks) > 0 && !slices.Contains(variant.Stacks, stack.ID) {
		return false
	}

	return strings.TrimSpace(variant.Overlays[stack.ID]) != ""
}

func validateRelationships(selections []*selection) error {
	byID := make(map[string]*selection, len(selections))
	for _, entry := range selections {
		byID[entry.feature.ID] = entry
	}

	for _, entry := range selections {
		for _, ref := range requirements(entry) {
			id, variantID := splitReference(ref)

			required, ok := byID[id]
			if !ok {
				return fmt.Errorf("feature %q requires %q", entry.feature.ID, ref)
			}

			if variantID != "" && required.variantID != variantID {
				return fmt.Errorf(
					"feature %q requires %q but %q is selected",
					entry.feature.ID,
					ref,
					required.variantID,
				)
			}
		}

		for _, ref := range conflicts(entry) {
			id, variantID := splitReference(ref)

			conflicting, ok := byID[id]
			if !ok {
				continue
			}

			if variantID == "" || conflicting.variantID == variantID {
				return fmt.Errorf(
					"feature %q conflicts with %q",
					entry.feature.ID,
					ref,
				)
			}
		}
	}

	return nil
}

func requirements(entry *selection) []string {
	refs := append([]string{}, entry.feature.Requires...)
	if entry.variant != nil {
		refs = append(refs, entry.variant.Requires...)
	}

	return refs
}

func conflicts(entry *selection) []string {
	refs := append([]string{}, entry.feature.Conflicts...)
	if entry.variant != nil {
		refs = append(refs, entry.variant.Conflicts...)
	}

	return refs
}

func splitReference(ref string) (string, string) {
	id, variantID, _ := strings.Cut(ref, ":")

	return id, variantID
}

func orderFeatures(selections []*selection) ([]*selection, error) {
	byID := make(map[string]*selection, len(selections))
	for _, entry := range selections {
		byID[entry.feature.ID] = entry
	}

	state := make(map[string]int, len(selections))
	ordered := make([]*selection, 0, len(selections))

	var visit func(entry *selection) error

	visit = func(entry *selection) error {
		switch state[entry.feature.ID] {
		case 2:
			return nil
		case 1:
			return fmt.Errorf("feature dependency cycle involving %q", entry.feature.ID)
		}

		state[entry.feature.ID] = 1
		for _, ref := range requirements(entry) {
			id, _ := splitReference(ref)
			if dependency, ok := byID[id]; ok {
				if err := visit(dependency); err != nil {
					return err
				}
			}
		}

		state[entry.feature.ID] = 2
		ordered = append(ordered, entry)

		return nil
	}

	for _, entry := range selections {
		if err := visit(entry); err != nil {
			return nil, err
		}
	}

	return ordered, nil
}

func buildOverlays(
	registryDir string,
	stack manifest.Stack,
	ordered []*selection,
) ([]OverlayPlan, error) {
	overlays := make([]OverlayPlan, 0, len(ordered))

	for _, entry := range ordered {
		dir := filepath.Join(registryDir, filepath.FromSlash(entry.overlayPath))

		overlay, err := patch.LoadOverlayFile(filepath.Join(dir, "overlay.yaml"))
		if err != nil {
			return nil, fmt.Errorf(
				"load overlay for feature %q on stack %q: %w",
				entry.feature.ID,
				stack.ID,
				err,
			)
		}

		if err := overlay.Validate(); err != nil {
			return nil, fmt.Errorf("validate overlay for feature %q: %w", entry.feature.ID, err)
		}

		overlays = append(overlays, OverlayPlan{
			FeatureID: entry.feature.ID,
			VariantID: entry.variantID,
			Path:      entry.overlayPath,
			Dir:       dir,
			Overlay:   overlay,
		})
	}

	return overlays, nil
}
