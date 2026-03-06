package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Robben-Media/jobtread-cli/internal/catalog"
)

func newItemsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "items",
		Short: "Inspect the API item coverage catalog",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List tracked API items",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := catalog.Load()
			if err != nil {
				return err
			}

			for _, item := range c.Items {
				fmt.Fprintf(cmd.OutOrStdout(), "%-22s %-6s %-24s %s\n", item.Status, item.Method, item.Path, item.Name)
			}

			return nil
		},
	})

	return cmd
}
