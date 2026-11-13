package cli

import (
	"github.com/BoatengPeter/smokeforge/internal/engine"
	"github.com/spf13/cobra"
)

//nolint:dupl // add and remove intentionally share the mutation command shape.
func newRemoveCmd(opts *options) *cobra.Command {
	projectDir := "."
	cascade := false

	cmd := &cobra.Command{
		Use:   "remove <feature[:variant]>...",
		Short: "Remove features from an existing project",
		Long: `Revert feature overlays from a generated project.

Removal is refused when another active feature requires the target. Use
--cascade to remove the dependent features first. Overlays that patch package
dependencies may still require manual removal.`,
		Example: `  smokeforge remove analytics --project-dir my-app
  smokeforge remove database --cascade --project-dir my-app`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectContext, reg, err := loadMutationTargets(cmd, opts, projectDir)
			if err != nil {
				return err
			}

			result, err := engine.Remove(cmd.Context(), engine.RemoveRequest{
				ProjectDir: projectContext.root,
				Registry:   reg,
				Features:   args,
				Cascade:    cascade,
			})
			if err != nil {
				return err
			}

			return printMutation(opts.out, "Features removed", result, result.Removed)
		},
	}

	cmd.Flags().StringVar(&projectDir, "project-dir", ".", "project directory")
	cmd.Flags().BoolVar(&cascade, "cascade", false, "remove dependent features too")

	return cmd
}
