package cli

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/BoatengPeter/smokeforge/internal/manifest"
	"github.com/spf13/cobra"
)

func newListCmd(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "list [stacks|features]",
		Short: "List registry contents",
		Long:  "List the stacks or features available in the selected registry.",
		Example: `  smokeforge list stacks
  smokeforge list features
  smokeforge list features --registry ./smokeforge-registry`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			what := "stacks"
			if len(args) == 1 {
				what = args[0]
			}

			switch what {
			case "stacks", "features":
			default:
				return fmt.Errorf("unknown list target %q: want stacks or features", what)
			}

			res, err := loadRegistry(cmd.Context(), opts)
			if err != nil {
				return err
			}

			return printRegistryList(opts.out, what, res.Manifest)
		},
	}
}

func printRegistryList(out io.Writer, what string, m *manifest.Manifest) error {
	var buf strings.Builder

	tw := tabwriter.NewWriter(&buf, 0, 4, 2, ' ', 0)

	if what == "stacks" {
		if err := printStacks(tw, m); err != nil {
			return err
		}
	} else {
		if err := printFeatures(tw, m); err != nil {
			return err
		}
	}

	if err := tw.Flush(); err != nil {
		return err
	}

	_, err := io.WriteString(out, buf.String())

	return err
}

func printStacks(tw *tabwriter.Writer, m *manifest.Manifest) error {
	if _, err := fmt.Fprintln(tw, "ID\tKIND\tNAME"); err != nil {
		return err
	}

	for _, stack := range m.Stacks {
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\n", stack.ID, stack.Kind, stack.Name); err != nil {
			return err
		}
	}

	return nil
}

func printFeatures(tw *tabwriter.Writer, m *manifest.Manifest) error {
	if _, err := fmt.Fprintln(tw, "ID\tVARIANTS\tREQUIRES"); err != nil {
		return err
	}

	for _, feature := range m.Features {
		variants := make([]string, 0, len(feature.Variants))
		for _, variant := range feature.Variants {
			variants = append(variants, variant.ID)
		}

		if _, err := fmt.Fprintf(
			tw,
			"%s\t%s\t%s\n",
			feature.ID,
			strings.Join(variants, ","),
			strings.Join(feature.Requires, ","),
		); err != nil {
			return err
		}
	}

	return nil
}
