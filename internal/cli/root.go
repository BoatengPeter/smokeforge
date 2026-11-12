// Package cli implements the smokeforge command tree.
package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/BoatengPeter/smokeforge/internal/registry"
	"github.com/spf13/cobra"
)

type options struct {
	out      io.Writer
	errOut   io.Writer
	registry string
	tag      string
	cacheDir string
	offline  bool
}

// NewRootCmd builds the smokeforge command tree with the given output writers.
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
	root.PersistentFlags().StringVar(&opts.cacheDir, "cache-dir", "", "registry cache directory")
	root.PersistentFlags().BoolVar(&opts.offline, "offline", false, "use cached registry only")

	root.AddCommand(newVersionCmd(out), newListCmd(opts), newRegistryCmd(opts), newDoctorCmd(opts))

	return root
}

func loadRegistry(ctx context.Context, opts *options) (*registry.Result, error) {
	return registry.Load(ctx, registry.Options{
		Source:   opts.registry,
		Tag:      opts.tag,
		CacheDir: opts.cacheDir,
		Offline:  opts.offline,
	})
}

func writef(out io.Writer, format string, args ...any) error {
	_, err := fmt.Fprintf(out, format, args...)

	return err
}
