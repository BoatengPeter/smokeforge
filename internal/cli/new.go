package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/BoatengPeter/smokeforge/internal/engine"
	"github.com/BoatengPeter/smokeforge/internal/manifest"
	"github.com/BoatengPeter/smokeforge/internal/registry"
	"github.com/BoatengPeter/smokeforge/internal/render"
	"github.com/BoatengPeter/smokeforge/internal/resolve"
	"github.com/BoatengPeter/smokeforge/internal/tui"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var errProjectAborted = errors.New("project creation aborted")

type newOptions struct {
	kind        string
	stack       string
	name        string
	preset      string
	features    []string
	variables   []string
	noInput     bool
	noGit       bool
	dryRun      bool
	keepStaging bool
}

func newNewCmd(opts *options) *cobra.Command {
	newOpts := &newOptions{}

	cmd := &cobra.Command{
		Use:   "new [dir]",
		Short: "Create a project from the template registry",
		Long: `Create a project from the official registry.

Presets:
  frontend       Next.js app with shadcn/ui, SEO, API helpers, and Jest
  fullstack      Frontend plus Postgres, Better Auth, and Zustand
  production     Production Next.js plus Docker, GitHub Actions, E2E, and PostHog
  api            Go/Gin API with Postgres and JWT authentication
  api-minimal    Go/Gin base without database or authentication
  api-production Production Go/Gin API with Postgres, auth, TOTP, Docker, and CI

Required companions are added automatically: auth adds database, and CI adds
E2E. Use --feature to select individual features instead of a preset.`,
		Example: `  smokeforge new my-app
  smokeforge new my-app --preset frontend
  smokeforge new my-saas --preset fullstack
  smokeforge new my-app --preset production
  smokeforge new my-app --preset frontend --feature docker --feature ci:github
  smokeforge new my-app --registry ./smokeforge-registry --tag v0.1.0`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := ""
			if len(args) == 1 {
				dir = args[0]
			}

			return runNew(cmd, opts, newOpts, dir)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&newOpts.kind, "kind", "", "project kind (backend, frontend, mobile, desktop)")
	flags.StringVar(&newOpts.stack, "stack", "", "stack ID")
	flags.StringVar(&newOpts.name, "name", "", "project name")
	flags.StringVar(&newOpts.preset, "preset", "", "feature preset")
	flags.StringArrayVar(&newOpts.features, "feature", nil, "feature selection (repeatable, id[:variant])")
	flags.StringArrayVar(&newOpts.variables, "var", nil, "template variable (repeatable, key=value)")
	flags.BoolVar(&newOpts.noInput, "no-input", false, "never prompt")
	flags.BoolVar(&newOpts.noGit, "no-git", false, "do not initialize a git repository")
	flags.BoolVar(&newOpts.dryRun, "dry-run", false, "render and validate without writing the project")
	flags.BoolVar(&newOpts.keepStaging, "keep-staging", false, "retain the staging directory after failure")

	return cmd
}

type newExecution struct {
	cmd      *cobra.Command
	opts     *options
	newOpts  *newOptions
	dir      string
	registry *registry.Result
	plan     *resolve.Plan
}

func runNew(cmd *cobra.Command, opts *options, newOpts *newOptions, dir string) error {
	registryResult, err := loadRegistry(cmd.Context(), opts)
	if err != nil {
		return err
	}

	selection, err := collectSelection(cmd, opts, newOpts, registryResult)
	if err != nil {
		return handleAbort(opts.out, err)
	}

	if dir == "" {
		dir = render.Slug(selection.Name)
	}

	plan, err := resolve.Resolve(registryResult.Manifest, registryResult.Dir, selection)
	if err != nil {
		return err
	}

	if err := printSummary(opts.out, plan, dir, registryResult.Tag); err != nil {
		return err
	}

	if err := confirmIfInteractive(cmd, opts, newOpts); err != nil {
		return err
	}

	execution := newExecution{
		cmd:      cmd,
		opts:     opts,
		newOpts:  newOpts,
		dir:      dir,
		registry: registryResult,
		plan:     plan,
	}

	return execution.generate()
}

func handleAbort(out io.Writer, err error) error {
	if errors.Is(err, errProjectAborted) {
		return writef(out, "%s\n", tui.Muted("Aborted."))
	}

	return err
}

func printSummary(out io.Writer, plan *resolve.Plan, dir, tag string) error {
	_, err := fmt.Fprintln(out, tui.Box("Project summary", summaryLines(plan, dir, tag)))

	return err
}

func confirmIfInteractive(cmd *cobra.Command, opts *options, newOpts *newOptions) error {
	if !isInteractive(cmd, newOpts) {
		return nil
	}

	confirmed, err := confirmGeneration(cmd, opts)
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return writef(opts.out, "%s\n", tui.Muted("Aborted."))
		}

		return err
	}

	if !confirmed {
		return writef(opts.out, "%s\n", tui.Muted("Aborted."))
	}

	return nil
}

func (e newExecution) generate() error {
	if _, err := fmt.Fprintln(e.opts.out, tui.Title("Generating project")); err != nil {
		return err
	}

	result, err := engine.Generate(e.cmd.Context(), engine.Request{
		Registry:    e.registry,
		Plan:        e.plan,
		Target:      e.dir,
		NoGit:       e.newOpts.noGit,
		DryRun:      e.newOpts.dryRun,
		KeepStaging: e.newOpts.keepStaging,
		Progress: func(stage string) {
			_, _ = fmt.Fprintln(e.opts.out, tui.Step(stage))
		},
	})
	if err != nil {
		return err
	}

	if result.DryRun {
		return printDryRun(e.opts.out, result)
	}

	return printSuccess(
		e.opts.out,
		result,
		e.newOpts.noGit,
		e.plan,
		isDockerInstalled(),
	)
}

func collectSelection(
	cmd *cobra.Command,
	opts *options,
	newOpts *newOptions,
	registryResult *registry.Result,
) (resolve.Selection, error) {
	variables, err := parseVariables(newOpts.variables)
	if err != nil {
		return resolve.Selection{}, err
	}

	selection := resolve.Selection{
		Kind:      newOpts.kind,
		Stack:     newOpts.stack,
		Name:      newOpts.name,
		Preset:    newOpts.preset,
		Features:  append([]string{}, newOpts.features...),
		Variables: variables,
	}

	if isInteractive(cmd, newOpts) {
		wizardSelection, err := runWizard(cmd, opts, registryResult, selection)
		if err != nil {
			return resolve.Selection{}, err
		}

		selection = wizardSelection
	} else {
		selection = withDefaultPreset(selection, registryResult.Manifest)
	}

	if strings.TrimSpace(selection.Kind) == "" {
		return resolve.Selection{}, errors.New("kind is required (use --kind)")
	}

	if strings.TrimSpace(selection.Stack) == "" {
		return resolve.Selection{}, errors.New("stack is required (use --stack)")
	}

	if strings.TrimSpace(selection.Name) == "" {
		return resolve.Selection{}, errors.New("project name is required (use --name)")
	}

	return selection, nil
}

func runWizard(
	cmd *cobra.Command,
	opts *options,
	registryResult *registry.Result,
	selection resolve.Selection,
) (resolve.Selection, error) {
	wizard := &tui.Wizard{
		Manifest:    registryResult.Manifest,
		RegistryDir: registryResult.Dir,
		In:          cmd.InOrStdin(),
		Out:         opts.out,
		Result: tui.WizardResult{
			Kind:      selection.Kind,
			Stack:     selection.Stack,
			Name:      selection.Name,
			Preset:    selection.Preset,
			Features:  selection.Features,
			Variables: selection.Variables,
		},
	}

	if err := wizard.Run(cmd.Context()); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return resolve.Selection{}, errProjectAborted
		}

		return resolve.Selection{}, err
	}

	return resolve.Selection{
		Kind:      wizard.Result.Kind,
		Stack:     wizard.Result.Stack,
		Name:      wizard.Result.Name,
		Features:  wizard.Result.Features,
		Variables: wizard.Result.Variables,
	}, nil
}

func withDefaultPreset(selection resolve.Selection, m *manifest.Manifest) resolve.Selection {
	if selection.Preset != "" || len(selection.Features) > 0 {
		return selection
	}

	if preset := m.DefaultPresetForKind(selection.Kind); preset != nil {
		selection.Preset = preset.ID
	}

	return selection
}

func parseVariables(values []string) (map[string]string, error) {
	variables := make(map[string]string, len(values))

	for _, raw := range values {
		key, value, found := strings.Cut(raw, "=")

		key = strings.TrimSpace(key)
		if !found || key == "" {
			return nil, fmt.Errorf("invalid --var %q: want key=value", raw)
		}

		if _, exists := variables[key]; exists {
			return nil, fmt.Errorf("duplicate variable %q", key)
		}

		variables[key] = value
	}

	return variables, nil
}

func summaryLines(plan *resolve.Plan, dir, registryTag string) []string {
	features := make([]string, 0, len(plan.Target.Overlays))
	for _, overlay := range plan.Target.Overlays {
		feature := overlay.FeatureID
		if overlay.VariantID != "" {
			feature += ":" + overlay.VariantID
		}

		features = append(features, feature)
	}

	featureSummary := "none"
	if len(features) > 0 {
		featureSummary = strings.Join(features, ", ")
	}

	return []string{
		tui.Field("Name", plan.Name),
		tui.Field("Kind", plan.Kind),
		tui.Field("Stack", plan.Target.Stack.Name+" ("+plan.Target.Stack.ID+")"),
		tui.Field("Features", featureSummary),
		tui.Field("Target", dir),
		tui.Field("Registry", registryTag),
	}
}

func confirmGeneration(cmd *cobra.Command, opts *options) (bool, error) {
	confirmed := true

	form := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title("Create this project?").
			Value(&confirmed),
	)).
		WithTheme(huh.ThemeCharm()).
		WithInput(cmd.InOrStdin()).
		WithOutput(opts.out)

	if err := form.RunWithContext(cmd.Context()); err != nil {
		return false, fmt.Errorf("confirm generation: %w", err)
	}

	return confirmed, nil
}

func printDryRun(out io.Writer, result *engine.Result) error {
	lines := []string{
		tui.Field("Target", result.Target),
		tui.Field("Files", strconv.Itoa(len(result.Files))),
		tui.Field("Registry", result.Lock.Registry.Tag),
		"",
		tui.Muted("No files were written."),
	}

	_, err := fmt.Fprintln(out, tui.Box("Dry run complete", lines))

	return err
}

func printSuccess(
	out io.Writer,
	result *engine.Result,
	noGit bool,
	plan *resolve.Plan,
	dockerInstalled bool,
) error {
	lines := []string{
		tui.Field("Path", result.Target),
		tui.Field("Files", strconv.Itoa(len(result.Files))),
		tui.Field("Registry", result.Lock.Registry.Tag),
	}
	if !noGit {
		lines = append(lines, tui.Field("Git", "initialized"))
	}

	lines = append(lines, "", "")
	lines = append(lines, nextStepLines(plan, result.Target, dockerInstalled)...)

	_, err := fmt.Fprintln(out, tui.Box("Project created", lines))

	return err
}

func nextStepLines(plan *resolve.Plan, path string, dockerInstalled bool) []string {
	lines := []string{
		tui.Muted("Next steps:"),
		"  cd " + path,
	}

	if plan.Target.Stack.ID != "next" {
		return append(lines, "  See README.md for project commands.")
	}

	packageManager := plan.Target.Variables["package_manager"]
	if packageManager == "" {
		packageManager = "pnpm"
	}

	lines = append(
		lines,
		"  make install        # or: "+packageInstallCommand(packageManager),
	)

	if hasFeature(plan, "database") {
		if dockerInstalled {
			lines = append(
				lines,
				"  make db-up          # start Postgres with Docker",
				"  make db-migrate     # create the database schema",
			)
		} else {
			lines = append(
				lines,
				"  # Docker is not installed: https://docs.docker.com/get-docker/",
				"  # Install it, then run: make db-up",
				"  # Or point DATABASE_URL in .env at an existing Postgres and run:",
				"  make db-migrate",
			)
		}
	}

	return append(lines, "  make dev")
}

func isDockerInstalled() bool {
	_, err := exec.LookPath("docker")

	return err == nil
}

func packageInstallCommand(packageManager string) string {
	switch packageManager {
	case "npm":
		return "npm install"
	case "bun":
		return "bun install"
	default:
		return "pnpm install"
	}
}

func hasFeature(plan *resolve.Plan, featureID string) bool {
	for _, overlay := range plan.Target.Overlays {
		if overlay.FeatureID == featureID {
			return true
		}
	}

	return false
}

func isInteractive(cmd *cobra.Command, newOpts *newOptions) bool {
	if newOpts.noInput {
		return false
	}

	reader := cmd.InOrStdin()

	file, ok := reader.(*os.File)
	if !ok {
		return false
	}

	//nolint:gosec // file descriptors fit in int on supported platforms.
	return term.IsTerminal(int(file.Fd()))
}
