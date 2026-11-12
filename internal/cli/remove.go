package cli

import (
	"github.com/BoatengPeter/smokeforge/internal/engine"
	"github.com/spf13/cobra"
)

func newRemoveCmd(opts *options) *cobra.Command {
	projectDir := "."

	cmd := &cobra.Command{
		Use:   "remove <feature[:variant]>...",
		Short: "Remove features from an existing project",
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

			result, err := engine.Remove(cmd.Context(), engine.RemoveRequest{
				ProjectDir: projectContext.root,
				Registry:   reg,
				Features:   args,
			})
			if err != nil {
				return err
			}

			return printMutation(opts.out, "Features removed", result, result.Removed)
		},
	}

	cmd.Flags().StringVar(&projectDir, "project-dir", ".", "project directory")

	return cmd
}
