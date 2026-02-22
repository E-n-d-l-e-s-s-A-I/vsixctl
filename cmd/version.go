package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number of vsixctl",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("V0.1.0")
		},
	}
}
