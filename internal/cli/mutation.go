package cli

import (
	"github.com/BoatengPeter/smokeforge/internal/registry"
	"github.com/spf13/cobra"
)

func loadMutationTargets(
	cmd *cobra.Command,
	opts *options,
	projectDir string,
) (*mutationProject, *registry.Result, error) {
	projectContext, err := loadMutationProject(projectDir)
	if err != nil {
		return nil, nil, err
	}

	reg, err := loadMutationRegistry(cmd.Context(), opts, projectContext)
	if err != nil {
		return nil, nil, err
	}

	return projectContext, reg, nil
}
