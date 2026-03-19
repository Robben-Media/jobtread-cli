package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Robben-Media/jobtread-cli/internal/httpclient"
)

func newRequestCmd(options *globalOptions) *cobra.Command {
	var (
		queryPairs []string
		bodyValue  string
		bodyFile   string
	)

	cmd := &cobra.Command{
		Use:   "request METHOD PATH",
		Short: "Call a JobTread API endpoint directly",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return errors.New("request requires METHOD and PATH")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := options.resolveRequestClient(context.Background())
			if err != nil {
				return err
			}

			body, contentType, err := resolveBody(bodyValue, bodyFile)
			if err != nil {
				return err
			}

			response, err := client.Do(context.Background(), httpclient.Request{
				Method:      strings.ToUpper(args[0]),
				Path:        args[1],
				Query:       parseKeyValuePairs(queryPairs),
				Headers:     parseKeyValuePairs(options.headerPairs),
				Body:        body,
				ContentType: contentType,
			})
			if err != nil {
				return err
			}

			return options.emit(cmd, response)
		},
	}

	cmd.Flags().StringArrayVar(&queryPairs, "query-param", nil, "Query parameter in key=value form")
	cmd.Flags().StringVar(&bodyValue, "body", "", "Raw request body")
	cmd.Flags().StringVar(&bodyFile, "from-file", "", "Read request body from file")

	return cmd
}

func resolveBody(bodyValue string, bodyFile string) ([]byte, string, error) {
	if strings.TrimSpace(bodyFile) != "" {
		body, err := os.ReadFile(strings.TrimSpace(bodyFile))
		if err != nil {
			return nil, "", fmt.Errorf("read request body file: %w", err)
		}

		return body, "application/json", nil
	}

	if strings.TrimSpace(bodyValue) == "" {
		return nil, "", nil
	}

	return []byte(bodyValue), "application/json", nil
}

func parseKeyValuePairs(values []string) map[string]string {
	parsed := make(map[string]string, len(values))
	for _, value := range values {
		key, rest, ok := strings.Cut(value, "=")
		if !ok {
			continue
		}
		parsed[strings.TrimSpace(key)] = strings.TrimSpace(rest)
	}

	return parsed
}
