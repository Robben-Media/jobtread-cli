package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Robben-Media/jobtread-cli/internal/config"
	"github.com/Robben-Media/jobtread-cli/internal/httpclient"
)

func newRequestCmd() *cobra.Command {
	var (
		queryPairs  []string
		headerPairs []string
		bodyValue   string
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
			cfg := config.Load()
			if err := cfg.ValidateRequestAuth(); err != nil {
				return err
			}

			body, contentType, err := resolveBody(bodyValue)
			if err != nil {
				return err
			}

			client := httpclient.New(cfg)
			response, statusCode, err := client.Do(context.Background(), httpclient.Request{
				Method:      strings.ToUpper(args[0]),
				Path:        args[1],
				Query:       parseKeyValuePairs(queryPairs),
				Headers:     parseKeyValuePairs(headerPairs),
				Body:        body,
				ContentType: contentType,
			})
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "HTTP %d\n", statusCode)
			fmt.Fprintln(cmd.OutOrStdout(), string(response))
			return nil
		},
	}

	cmd.Flags().StringArrayVar(&queryPairs, "query", nil, "Query parameter in key=value form")
	cmd.Flags().StringArrayVar(&headerPairs, "header", nil, "Header in key=value form")
	cmd.Flags().StringVar(&bodyValue, "body", "", "Raw request body or @path/to/file.json")

	return cmd
}

func resolveBody(bodyValue string) ([]byte, string, error) {
	if strings.TrimSpace(bodyValue) == "" {
		return nil, "", nil
	}

	if strings.HasPrefix(bodyValue, "@") {
		body, err := os.ReadFile(strings.TrimPrefix(bodyValue, "@"))
		if err != nil {
			return nil, "", fmt.Errorf("read request body file: %w", err)
		}

		return body, "application/json", nil
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
