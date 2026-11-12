package cli

import (
	"github.com/spf13/cobra"
)

func newListCmd(opts *options) *cobra.Command {
	return &cobra.Command{Use: "list", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
}
