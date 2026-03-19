package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Robben-Media/jobtread-cli/internal/catalog"
)

func newItemsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "items",
		Short: "Inspect the API coverage manifest",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List tracked API operations",
		RunE: func(cmd *cobra.Command, args []string) error {
			manifest, err := catalog.Load()
			if err != nil {
				return err
			}

			if len(manifest.Operations) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no manifest operations tracked yet")
				return nil
			}

			for _, operation := range manifest.Operations {
				target := operation.PathTemplate
				if operation.Transport == "pave" {
					target = operation.PaveField
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%-18s %-10s %-6s %s\n", operation.Group, operation.Command, operation.Transport, target)
			}
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "summary",
		Short: "Print manifest coverage metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			manifest, err := catalog.Load()
			if err != nil {
				return err
			}

			summary := map[string]any{
				"source_kind":        manifest.Source.Kind,
				"source_uri":         manifest.Source.URI,
				"imported_at":        manifest.Source.ImportedAt,
				"operation_count":    len(manifest.Operations),
				"source_checksum":    manifest.Source.Checksum,
				"manifest_populated": manifest.IsPopulated(),
			}
			raw, err := json.Marshal(summary)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			return nil
		},
	})

	return cmd
}
