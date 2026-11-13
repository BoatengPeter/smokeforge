package cli

import (
	"fmt"

	"github.com/BoatengPeter/smokeforge/internal/registry"
	"github.com/spf13/cobra"
)

func newRegistryCmd(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Inspect and update the template registry",
		Long:  "Inspect or refresh the registry used for generation.",
		Example: `  smokeforge registry info
  smokeforge registry update`,

		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("unknown registry subcommand %q: want info or update", args[0])
			}

			return cmd.Help()
		},
	}

	cmd.AddCommand(
		newRegistryInfoCmd(opts),
		newRegistryUpdateCmd(opts),
	)

	return cmd
}

func newRegistryInfoCmd(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show registry information",
		Long:  "Show the source, tag, schema version, and catalog counts for the selected registry.",
		Example: `  smokeforge registry info
  smokeforge registry info --registry ./smokeforge-registry`,

		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := loadRegistry(cmd.Context(), opts)
			if err != nil {
				return err
			}

			lines := []string{
				"source: " + res.Source,
				"tag: " + res.Tag,
				fmt.Sprintf("schema: %d", res.Manifest.SchemaVersion),
				fmt.Sprintf("stacks: %d", len(res.Manifest.Stacks)),
				fmt.Sprintf("features: %d", len(res.Manifest.Features)),
			}
			for _, line := range lines {
				if err := writef(opts.out, "%s\n", line); err != nil {
					return err
				}
			}

			return nil
		},
	}
}

func newRegistryUpdateCmd(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Refresh the cached registry",
		Long:  "Re-download the selected registry release and replace its cached copy.",
		Example: `  smokeforge registry update
  smokeforge registry update --tag v0.1.0`,

		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := registry.Refresh(cmd.Context(), registry.Options{
				Source:   opts.registry,
				Tag:      opts.tag,
				CacheDir: opts.cacheDir,
				Offline:  opts.offline,
			})
			if err != nil {
				return err
			}

			return writef(opts.out, "updated registry %s@%s\n", res.Source, res.Tag)
		},
	}
}
