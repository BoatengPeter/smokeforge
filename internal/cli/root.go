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
		Use:   "smokeforge",
		Short: "Scaffold projects from a versioned template registry",
		Long: `smokeforge scaffolds production-shaped projects from a versioned registry.

The official registry is downloaded automatically and cached locally.
Run "smokeforge new" with no arguments for the interactive wizard, or use
--preset to start from a known bundle.

Quick start:
  smokeforge new my-app --preset frontend
  smokeforge new my-saas --preset fullstack
  smokeforge list features
  smokeforge add auth --project-dir my-app

Documentation: https://github.com/BoatengPeter/smokeforge`,
		Example: `  smokeforge new my-app
  smokeforge new my-app --preset production
  smokeforge list stacks
  smokeforge doctor --project-dir my-app`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetOut(out)
	root.SetErr(errOut)

	root.PersistentFlags().StringVar(&opts.registry, "registry", "", "registry URL or local directory")
	root.PersistentFlags().StringVar(&opts.tag, "tag", "latest", "registry version tag")
	root.PersistentFlags().StringVar(&opts.cacheDir, "cache-dir", "", "registry cache directory")
	root.PersistentFlags().BoolVar(&opts.offline, "offline", false, "use cached registry only")

	root.AddCommand(
		newNewCmd(opts),
		newAddCmd(opts),
		newRemoveCmd(opts),
		newVersionCmd(out),
		newListCmd(opts),
		newRegistryCmd(opts),
		newDoctorCmd(opts),
		newCompletionCmd(root),
	)

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
