// Package cli implements the smokeforge command tree.
package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/BoatengPeter/smokeforge/internal/registry"
	"github.com/BoatengPeter/smokeforge/internal/version"
	"github.com/spf13/cobra"
)

type options struct {
	out        io.Writer
	errOut     io.Writer
	registry   string
	tag        string
	cacheDir   string
	offline    bool
	cliVersion string
}

// registryOptions builds registry load options for source and tag with the
// shared cache, offline, and CLI version settings.
func (o *options) registryOptions(source, tag string) registry.Options {
	return registry.Options{
		Source:     source,
		Tag:        tag,
		CacheDir:   o.cacheDir,
		Offline:    o.offline,
		CLIVersion: o.cliVersion,
	}
}

// NewRootCmd builds the smokeforge command tree with the given output writers.
func NewRootCmd(out, errOut io.Writer) *cobra.Command {
	return newRootCmd(out, errOut, version.Current().Version)
}

// newRootCmd builds the command tree for an explicit CLI version, which is
// compared against a registry's minCliVersion.
func newRootCmd(out, errOut io.Writer, cliVersion string) *cobra.Command {
	opts := &options{out: out, errOut: errOut, cliVersion: cliVersion}

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
  smokeforge update --check
  smokeforge add auth --project-dir my-app

Documentation: https://github.com/BoatengPeter/smokeforge`,
		Example: `  smokeforge new my-app
  smokeforge new my-app --preset fullstack
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
		newUpdateCmd(out),
		newListCmd(opts),
		newRegistryCmd(opts),
		newDoctorCmd(opts),
		newCompletionCmd(root),
	)

	return root
}

func loadRegistry(ctx context.Context, opts *options) (*registry.Result, error) {
	return registry.Load(ctx, opts.registryOptions(opts.registry, opts.tag))
}

func writef(out io.Writer, format string, args ...any) error {
	_, err := fmt.Fprintf(out, format, args...)

	return err
}
