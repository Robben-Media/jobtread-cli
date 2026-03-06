package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "jobtread",
		Short: "JobTread CLI bootstrap for full API coverage",
		Long:  "JobTread CLI provides a Go-based foundation for inventorying and calling the full JobTread API surface.",
	}

	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newRequestCmd())
	cmd.AddCommand(newItemsCmd())

	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the CLI version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), "jobtread-cli bootstrap")
		},
	}
}
