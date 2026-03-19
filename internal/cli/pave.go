package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newPaveCmd(options *globalOptions) *cobra.Command {
	var (
		bodyValue string
		bodyFile  string
		public    bool
		dryRun    bool
	)

	cmd := &cobra.Command{
		Use:   "pave",
		Short: "Execute raw JobTread Pave queries",
	}

	queryCmd := &cobra.Command{
		Use:   "query",
		Short: "Run a raw Pave query object",
		RunE: func(cmd *cobra.Command, args []string) error {
			query, err := readPaveJSON(bodyValue, bodyFile)
			if err != nil {
				return err
			}

			if dryRun {
				raw, marshalErr := json.Marshal(query)
				if marshalErr != nil {
					return marshalErr
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(raw))
				return nil
			}

			client, _, err := options.resolvePaveClient(context.Background(), public)
			if err != nil {
				return err
			}

			response, err := client.Execute(context.Background(), query, public)
			if err != nil {
				return err
			}

			return options.emit(cmd, response)
		},
	}

	queryCmd.Flags().StringVar(&bodyValue, "body", "", "Raw Pave query JSON object")
	queryCmd.Flags().StringVar(&bodyFile, "from-file", "", "Read raw Pave query JSON object from file")
	queryCmd.Flags().BoolVar(&public, "public", false, "Skip grant-key resolution for public Pave operations")
	queryCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print the final query without executing")
	cmd.AddCommand(queryCmd)

	return cmd
}

func readPaveJSON(bodyValue string, bodyFile string) (map[string]any, error) {
	bodyValue = strings.TrimSpace(bodyValue)
	bodyFile = strings.TrimSpace(bodyFile)
	if bodyValue == "" && bodyFile == "" {
		return nil, errors.New("either --body or --from-file is required")
	}

	var raw []byte
	switch {
	case bodyFile != "":
		data, err := os.ReadFile(bodyFile)
		if err != nil {
			return nil, fmt.Errorf("read query file: %w", err)
		}
		raw = data
	default:
		raw = []byte(bodyValue)
	}

	var query map[string]any
	if err := json.Unmarshal(raw, &query); err != nil {
		return nil, fmt.Errorf("decode query JSON: %w", err)
	}
	return query, nil
}
