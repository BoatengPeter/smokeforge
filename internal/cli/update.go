package cli

import (
	"encoding/json"
	"io"

	"github.com/BoatengPeter/smokeforge/internal/selfupdate"
	"github.com/spf13/cobra"
)

func newUpdateCmd(out io.Writer) *cobra.Command {
	var (
		check   bool
		force   bool
		jsonOut bool
		target  string
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update smokeforge to a release version",
		Long: `Update the installed smokeforge binary from GitHub releases.

Use --check to inspect the available version without changing the binary.
The --version flag accepts "latest" or a release tag such as v0.1.6.`,
		Example: `  smokeforge update --check
  smokeforge update
  smokeforge update --version v0.1.6
  smokeforge update --version latest --force`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := selfupdate.Update(cmd.Context(), selfupdate.Options{
				TargetVersion: target,
				CheckOnly:     check,
				Force:         force,
			})
			if err != nil {
				return err
			}

			if jsonOut {
				encoder := json.NewEncoder(out)
				encoder.SetIndent("", "  ")

				return encoder.Encode(result)
			}

			if check {
				if result.Available {
					return writef(out, "update available: %s -> %s\n", result.Current, result.Latest)
				}

				return writef(out, "smokeforge is up to date (%s)\n", result.Latest)
			}

			if !result.Updated {
				return writef(out, "smokeforge is up to date (%s)\n", result.Latest)
			}

			return writef(out, "updated smokeforge to %s\n", result.Latest)
		},
	}

	cmd.Flags().BoolVar(&check, "check", false, "check for an update without installing it")
	cmd.Flags().BoolVar(&force, "force", false, "install even when the version is unchanged")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print machine-readable JSON")
	cmd.Flags().StringVar(&target, "version", "latest", "target release version or latest")

	return cmd
}
