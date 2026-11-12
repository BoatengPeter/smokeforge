package cli

import (
	"fmt"
	"io"

	"github.com/BoatengPeter/smokeforge/internal/version"
	"github.com/spf13/cobra"
)

func newVersionCmd(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(_ *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(out, "smokeforge %s\n", version.String())
			return err
		},
	}
}
