package cli

import "github.com/spf13/cobra"

func newRegistryCmd(opts *options) *cobra.Command {
	return &cobra.Command{Use: "registry", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
}
