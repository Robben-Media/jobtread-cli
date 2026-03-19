package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Robben-Media/jobtread-cli/internal/catalog"
	"github.com/Robben-Media/jobtread-cli/internal/httpclient"
)

var reservedGeneratedFlags = map[string]struct{}{
	"all":         {},
	"body":        {},
	"cursor":      {},
	"dry-run":     {},
	"fields":      {},
	"from-file":   {},
	"from-select": {},
	"limit":       {},
	"page":        {},
	"public":      {},
	"select":      {},
}

func addGeneratedResourceCommands(root *cobra.Command, options *globalOptions, manifest catalog.Manifest) {
	groups := make(map[string][]catalog.Operation)
	for _, operation := range manifest.Operations {
		groups[operation.Group] = append(groups[operation.Group], operation)
	}

	groupNames := make([]string, 0, len(groups))
	for group := range groups {
		groupNames = append(groupNames, group)
	}
	sort.Strings(groupNames)

	for _, group := range groupNames {
		operations := groups[group]
		resourceCmd := &cobra.Command{
			Use:   group,
			Short: "Generated JobTread resource commands for " + group,
		}

		for _, operation := range operations {
			resourceCmd.AddCommand(newOperationCmd(options, operation))
		}

		root.AddCommand(resourceCmd)
	}
}

func newOperationCmd(options *globalOptions, operation catalog.Operation) *cobra.Command {
	switch operation.Transport {
	case "pave":
		return newPaveOperationCmd(options, operation)
	default:
		return newHTTPOperationCmd(options, operation)
	}
}

func newHTTPOperationCmd(options *globalOptions, operation catalog.Operation) *cobra.Command {
	var (
		bodyValue string
		bodyFile  string
		fields    []string
		expand    []string
		filter    []string
		sortValue string
		limit     int
		page      int
		cursor    string
		allPages  bool
		dryRun    bool
	)

	cmd := &cobra.Command{
		Use:   operation.Command,
		Short: fallbackString(operation.Summary, operation.Method+" "+operation.PathTemplate),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := operation.PathTemplate
			for _, parameter := range operation.PathParams {
				value, err := cmd.Flags().GetString(parameter.Name)
				if err != nil {
					return err
				}
				value = strings.TrimSpace(value)
				if parameter.Required && value == "" {
					return fmt.Errorf("--%s is required", parameter.Name)
				}
				path = strings.ReplaceAll(path, "{"+parameter.Name+"}", value)
			}

			query := make(map[string]string)
			for _, parameter := range operation.QueryParams {
				if _, reserved := reservedGeneratedFlags[parameter.Name]; reserved {
					continue
				}
				value, err := cmd.Flags().GetString(parameter.Name)
				if err != nil {
					return err
				}
				if strings.TrimSpace(value) != "" {
					query[parameter.Name] = value
				}
			}

			if len(fields) > 0 {
				query["fields"] = strings.Join(fields, ",")
			}
			if len(expand) > 0 {
				query["expand"] = strings.Join(expand, ",")
			}
			if len(filter) > 0 {
				query["filter"] = strings.Join(filter, ",")
			}
			if strings.TrimSpace(sortValue) != "" {
				query["sort"] = strings.TrimSpace(sortValue)
			}
			if limit > 0 {
				query["limit"] = strconv.Itoa(limit)
			}
			if page > 0 {
				query["page"] = strconv.Itoa(page)
			}
			if strings.TrimSpace(cursor) != "" {
				query["cursor"] = strings.TrimSpace(cursor)
			}

			if dryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", operation.Method, path)
				return nil
			}

			body, contentType, err := resolveBody(bodyValue, bodyFile)
			if err != nil {
				return err
			}

			client, _, err := options.resolveRequestClient(context.Background())
			if err != nil {
				return err
			}

			response, err := client.Do(context.Background(), httpclient.Request{
				Method:      operation.Method,
				Path:        path,
				Query:       query,
				Headers:     parseKeyValuePairs(options.headerPairs),
				Body:        body,
				ContentType: contentType,
			})
			if err != nil {
				return err
			}

			if allPages && operation.Method == "GET" && operation.PaginationMode != "" {
				return fmt.Errorf("--all pagination is not implemented for %s yet", operation.OperationID)
			}

			return options.emit(cmd, response)
		},
	}

	for _, parameter := range operation.PathParams {
		cmd.Flags().String(parameter.Name, "", "Path parameter "+parameter.Name)
	}
	for _, parameter := range operation.QueryParams {
		if _, reserved := reservedGeneratedFlags[parameter.Name]; reserved {
			continue
		}
		cmd.Flags().String(parameter.Name, "", "Query parameter "+parameter.Name)
	}

	cmd.Flags().StringArrayVar(&fields, "fields", nil, "Restrict fields")
	cmd.Flags().StringArrayVar(&expand, "expand", nil, "Expand related objects")
	cmd.Flags().StringArrayVar(&filter, "filter", nil, "Apply filter clauses")
	cmd.Flags().StringVar(&sortValue, "sort", "", "Sort expression")
	cmd.Flags().IntVar(&limit, "limit", 0, "Page size limit")
	cmd.Flags().IntVar(&page, "page", 0, "Page number")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Cursor token")
	cmd.Flags().BoolVar(&allPages, "all", false, "Fetch all pages when supported")
	cmd.Flags().StringVar(&bodyValue, "body", "", "Raw request body")
	cmd.Flags().StringVar(&bodyFile, "from-file", "", "Read request body from file")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print request without executing")

	return cmd
}

func newPaveOperationCmd(options *globalOptions, operation catalog.Operation) *cobra.Command {
	var (
		bodyValue  string
		bodyFile   string
		selectBody string
		selectFile string
		fields     []string
		limit      int
		page       string
		allPages   bool
		public     bool
		dryRun     bool
	)

	cmd := &cobra.Command{
		Use:   operation.Command,
		Short: fallbackString(operation.Summary, operation.PaveField),
		RunE: func(cmd *cobra.Command, args []string) error {
			input, err := buildPaveInput(cmd, operation, bodyValue, bodyFile)
			if err != nil {
				return err
			}
			if limit > 0 {
				input["size"] = limit
			}
			if strings.TrimSpace(page) != "" {
				input["page"] = strings.TrimSpace(page)
			}

			selection, err := buildPaveSelection(operation, selectBody, selectFile, fields)
			if err != nil {
				return err
			}

			query := buildPaveQuery(operation, input, selection)
			if dryRun {
				raw, marshalErr := json.Marshal(query)
				if marshalErr != nil {
					return marshalErr
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(raw))
				return nil
			}

			client, _, err := options.resolvePaveClient(context.Background(), public || operation.Public)
			if err != nil {
				return err
			}

			if allPages && operation.SupportsPagination {
				response, paginateErr := executeAllPavePages(context.Background(), client, operation, input, selection, public || operation.Public)
				if paginateErr != nil {
					return paginateErr
				}
				return options.emit(cmd, response)
			}

			response, err := client.Execute(context.Background(), query, public || operation.Public)
			if err != nil {
				return err
			}
			return options.emit(cmd, response)
		},
	}

	for _, field := range operation.InputFields {
		if _, reserved := reservedGeneratedFlags[field.Name]; reserved {
			continue
		}
		cmd.Flags().String(field.Name, "", inputFieldHelp(field))
	}

	cmd.Flags().StringVar(&bodyValue, "body", "", "Raw JSON object merged into the operation input")
	cmd.Flags().StringVar(&bodyFile, "from-file", "", "Read the operation input JSON object from file")
	cmd.Flags().StringVar(&selectBody, "select", "", "Override the selection set with a raw JSON object")
	cmd.Flags().StringVar(&selectFile, "select-file", "", "Read the selection set JSON object from file")
	cmd.Flags().StringArrayVar(&fields, "fields", nil, "Select response fields, or node fields for connection responses")
	cmd.Flags().IntVar(&limit, "limit", 0, "Page size for paginated operations")
	cmd.Flags().StringVar(&page, "page", "", "Page token for paginated operations")
	cmd.Flags().BoolVar(&allPages, "all", false, "Fetch all pages for connection operations")
	cmd.Flags().BoolVar(&public, "public", false, "Skip grant-key resolution for public operations")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print the generated Pave query without executing")

	return cmd
}

func inputFieldHelp(field catalog.InputField) string {
	parts := []string{field.Type}
	if field.Description != "" {
		parts = append(parts, field.Description)
	}
	if field.Required {
		parts = append(parts, "required")
	}
	return strings.Join(parts, " - ")
}

func buildPaveInput(cmd *cobra.Command, operation catalog.Operation, bodyValue string, bodyFile string) (map[string]any, error) {
	input := make(map[string]any)

	if strings.TrimSpace(bodyValue) != "" || strings.TrimSpace(bodyFile) != "" {
		rawInput, err := readPaveJSON(bodyValue, bodyFile)
		if err != nil {
			return nil, err
		}
		for key, value := range rawInput {
			input[key] = value
		}
	}

	for _, field := range operation.InputFields {
		if _, reserved := reservedGeneratedFlags[field.Name]; reserved {
			continue
		}
		value, err := cmd.Flags().GetString(field.Name)
		if err != nil {
			return nil, err
		}
		value = strings.TrimSpace(value)
		if value == "" {
			if field.Required && input[field.Name] == nil {
				return nil, fmt.Errorf("--%s is required", field.Name)
			}
			continue
		}
		parsed, err := parsePaveValue(field.Type, value)
		if err != nil {
			return nil, fmt.Errorf("parse --%s: %w", field.Name, err)
		}
		input[field.Name] = parsed
	}

	return input, nil
}

func buildPaveSelection(operation catalog.Operation, selectBody string, selectFile string, fields []string) (map[string]any, error) {
	if strings.TrimSpace(selectBody) != "" || strings.TrimSpace(selectFile) != "" {
		return readPaveJSON(selectBody, selectFile)
	}

	selection := cloneSelection(operation.DefaultSelection)
	if len(fields) == 0 {
		return selection, nil
	}

	custom := make(map[string]any, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		custom[field] = map[string]any{}
	}
	if len(custom) == 0 {
		return selection, nil
	}

	if _, ok := selection["nodes"]; ok {
		selection["nodes"] = custom
		return selection, nil
	}
	return custom, nil
}

func buildPaveQuery(operation catalog.Operation, input map[string]any, selection map[string]any) map[string]any {
	query := map[string]any{
		operation.PaveField: selection,
	}
	if len(input) > 0 {
		selection["$"] = input
	}
	return query
}

func cloneSelection(selection map[string]any) map[string]any {
	if len(selection) == 0 {
		return map[string]any{}
	}
	raw, err := json.Marshal(selection)
	if err != nil {
		return map[string]any{}
	}
	var cloned map[string]any
	if err := json.Unmarshal(raw, &cloned); err != nil {
		return map[string]any{}
	}
	return cloned
}

func parsePaveValue(typeName string, value string) (any, error) {
	switch {
	case strings.HasPrefix(typeName, "nullable("):
		return parsePaveValue(strings.TrimSuffix(strings.TrimPrefix(typeName, "nullable("), ")"), value)
	case strings.HasPrefix(typeName, "array("), typeName == "object", typeName == "enum":
		var decoded any
		if err := json.Unmarshal([]byte(value), &decoded); err != nil {
			return nil, fmt.Errorf("expected JSON value")
		}
		return decoded, nil
	case typeName == "boolean":
		return strconv.ParseBool(value)
	case typeName == "int":
		return strconv.Atoi(value)
	case typeName == "number":
		return strconv.ParseFloat(value, 64)
	default:
		return value, nil
	}
}

func executeAllPavePages(ctx context.Context, client interface {
	Execute(context.Context, map[string]any, bool) (httpclient.Response, error)
}, operation catalog.Operation, input map[string]any, selection map[string]any, public bool) (httpclient.Response, error) {
	aggregated := make([]any, 0)
	queryInput := cloneMap(input)
	var lastResponse httpclient.Response

	for {
		query := buildPaveQuery(operation, queryInput, cloneSelection(selection))
		response, err := client.Execute(ctx, query, public)
		if err != nil {
			return httpclient.Response{}, err
		}
		lastResponse = response

		var envelope map[string]any
		if err := json.Unmarshal(response.Body, &envelope); err != nil {
			return httpclient.Response{}, fmt.Errorf("decode paginated response: %w", err)
		}

		operationBody, _ := envelope[operation.PaveField].(map[string]any)
		nodes, _ := operationBody["nodes"].([]any)
		aggregated = append(aggregated, nodes...)
		nextPage, _ := operationBody["nextPage"].(string)
		if strings.TrimSpace(nextPage) == "" {
			envelope[operation.PaveField] = map[string]any{
				"nodes": aggregated,
				"count": len(aggregated),
			}
			raw, err := json.Marshal(envelope)
			if err != nil {
				return httpclient.Response{}, fmt.Errorf("encode aggregated response: %w", err)
			}
			lastResponse.Body = raw
			return lastResponse, nil
		}
		queryInput["page"] = nextPage
	}
}

func cloneMap(values map[string]any) map[string]any {
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func fallbackString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}

	return ""
}
