package cli

import (
	"io"

	"github.com/spf13/cobra"
)

type options struct {
	out      io.Writer
	errOut   io.Writer
	registry string
	tag      string
	offline  bool
}

func NewRootCmd(out, errOut io.Writer) *cobra.Command {
	opts := &options{out: out, errOut: errOut}

	root := &cobra.Command{
		Use:           "smokeforge",
		Short:         "Scaffold projects from a versioned template registry",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&opts.registry, "registry", "", "registry URL or local directory")
	root.PersistentFlags().StringVar(&opts.tag, "tag", "latest", "registry version tag")
	root.PersistentFlags().BoolVar(&opts.offline, "offline", false, "use cached registry only")

	root.AddCommand(newVersionCmd(out), newListCmd(opts), newRegistryCmd(opts), newDoctorCmd(opts))

	return root
}
