package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/BoatengPeter/smokeforge/internal/engine"
	"github.com/BoatengPeter/smokeforge/internal/tui"
	"github.com/spf13/cobra"
)

func newAddCmd(opts *options) *cobra.Command {
	projectDir := "."
	force := false

	cmd := &cobra.Command{
		Use:   "add <feature[:variant]>...",
		Short: "Add features to an existing project",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectContext, err := loadMutationProject(projectDir)
			if err != nil {
				return err
			}

			reg, err := loadMutationRegistry(cmd.Context(), opts, projectContext)
			if err != nil {
				return err
			}

			result, err := engine.Add(cmd.Context(), engine.AddRequest{
				ProjectDir: projectContext.root,
				Registry:   reg,
				Features:   args,
				Force:      force,
			})
			if err != nil {
				return err
			}

			return printMutation(opts.out, "Features added", result, result.Added)
		},
	}

	cmd.Flags().StringVar(&projectDir, "project-dir", ".", "project directory")
	cmd.Flags().BoolVar(&force, "force", false, "apply despite local file drift")

	return cmd
}

func printMutation(
	out io.Writer,
	title string,
	result *engine.MutationResult,
	changed []string,
) error {
	features := "none"
	if len(result.State.Targets[0].Features) > 0 {
		features = strings.Join(result.State.Targets[0].Features, ", ")
	}

	lines := []string{
		tui.Field("Project", result.ProjectDir),
		tui.Field("Changed", strings.Join(changed, ", ")),
		tui.Field("Features", features),
	}

	_, err := fmt.Fprintln(out, tui.Box(title, lines))

	return err
}
