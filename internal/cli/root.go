package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Robben-Media/jobtread-cli/internal/catalog"
)

func NewRootCmd() *cobra.Command {
	options := &globalOptions{}

	cmd := &cobra.Command{
		Use:   "jobtread",
		Short: "JobTread CLI bootstrap for full API coverage",
		Long:  "JobTread CLI provides a Go-based foundation for inventorying and calling the full JobTread API surface.",
	}

	addPersistentFlags(cmd, options)
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newRequestCmd(options))
	cmd.AddCommand(newPaveCmd(options))
	cmd.AddCommand(newItemsCmd())
	cmd.AddCommand(newSyncCmd(options))

	if manifest, err := catalog.Load(); err == nil {
		addGeneratedResourceCommands(cmd, options, manifest)
	}

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
