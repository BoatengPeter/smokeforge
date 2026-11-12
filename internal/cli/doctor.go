package cli

import "github.com/spf13/cobra"

func newDoctorCmd(opts *options) *cobra.Command {
	return &cobra.Command{Use: "doctor", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
}
