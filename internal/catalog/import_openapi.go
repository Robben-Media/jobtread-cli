package catalog

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type openAPIDocument struct {
	OpenAPI string                 `yaml:"openapi"`
	Info    openAPIInfo            `yaml:"info"`
	Paths   map[string]openAPIPath `yaml:"paths"`
}

type openAPIInfo struct {
	Title   string `yaml:"title"`
	Version string `yaml:"version"`
}

type openAPIPath struct {
	Parameters []openAPIParameter `yaml:"parameters"`
	Get        *openAPIOperation  `yaml:"get"`
	Post       *openAPIOperation  `yaml:"post"`
	Put        *openAPIOperation  `yaml:"put"`
	Patch      *openAPIOperation  `yaml:"patch"`
	Delete     *openAPIOperation  `yaml:"delete"`
}

type openAPIOperation struct {
	OperationID string             `yaml:"operationId"`
	Summary     string             `yaml:"summary"`
	Tags        []string           `yaml:"tags"`
	Parameters  []openAPIParameter `yaml:"parameters"`
	RequestBody map[string]any     `yaml:"requestBody"`
}

type openAPIParameter struct {
	Name     string `yaml:"name"`
	In       string `yaml:"in"`
	Required bool   `yaml:"required"`
}

func ImportOpenAPI(raw []byte, sourceURI string) (Manifest, error) {
	var document openAPIDocument
	if err := yaml.Unmarshal(raw, &document); err != nil {
		return Manifest{}, fmt.Errorf("decode OpenAPI document: %w", err)
	}

	manifest := Manifest{
		Source: NewSourceMeta("openapi", sourceURI, raw),
	}

	pathNames := make([]string, 0, len(document.Paths))
	for path := range document.Paths {
		pathNames = append(pathNames, path)
	}
	sort.Strings(pathNames)

	for _, path := range pathNames {
		item := document.Paths[path]
		manifest.Operations = append(manifest.Operations, normalizeOperation(path, "GET", item.Get, item.Parameters)...)
		manifest.Operations = append(manifest.Operations, normalizeOperation(path, "POST", item.Post, item.Parameters)...)
		manifest.Operations = append(manifest.Operations, normalizeOperation(path, "PUT", item.Put, item.Parameters)...)
		manifest.Operations = append(manifest.Operations, normalizeOperation(path, "PATCH", item.Patch, item.Parameters)...)
		manifest.Operations = append(manifest.Operations, normalizeOperation(path, "DELETE", item.Delete, item.Parameters)...)
	}

	manifest.Source.OperationCount = len(manifest.Operations)
	manifest.sort()

	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}

	return manifest, nil
}

func normalizeOperation(path string, method string, operation *openAPIOperation, sharedParams []openAPIParameter) []Operation {
	if operation == nil {
		return nil
	}

	group := normalizeGroup(operation.Tags, path)
	allParams := append(append([]openAPIParameter{}, sharedParams...), operation.Parameters...)
	pathParams, queryParams := splitParams(allParams)

	op := Operation{
		Group:            group,
		Command:          normalizeCommand(method, path, group, operation.OperationID),
		Method:           method,
		PathTemplate:     path,
		OperationID:      firstNonEmpty(operation.OperationID, strings.ToLower(method)+"_"+strings.ReplaceAll(strings.Trim(path, "/"), "/", "_")),
		Summary:          operation.Summary,
		Tags:             append([]string{}, operation.Tags...),
		PathParams:       pathParams,
		QueryParams:      queryParams,
		HasRequestBody:   len(operation.RequestBody) > 0,
		PaginationMode:   detectPagination(queryParams),
		FilterableFields: []string{},
		ExpandableFields: []string{},
	}

	return []Operation{op}
}

func splitParams(params []openAPIParameter) ([]Parameter, []Parameter) {
	var pathParams []Parameter
	var queryParams []Parameter

	seen := make(map[string]struct{})
	for _, parameter := range params {
		key := parameter.In + ":" + parameter.Name
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		normalized := Parameter{Name: parameter.Name, In: parameter.In, Required: parameter.Required}
		switch parameter.In {
		case "path":
			pathParams = append(pathParams, normalized)
		case "query":
			queryParams = append(queryParams, normalized)
		}
	}

	sort.Slice(pathParams, func(i, j int) bool { return pathParams[i].Name < pathParams[j].Name })
	sort.Slice(queryParams, func(i, j int) bool { return queryParams[i].Name < queryParams[j].Name })

	return pathParams, queryParams
}

func normalizeGroup(tags []string, path string) string {
	if len(tags) > 0 && strings.TrimSpace(tags[0]) != "" {
		return normalizeName(tags[0])
	}

	for _, segment := range strings.Split(strings.Trim(path, "/"), "/") {
		if segment == "" || strings.HasPrefix(segment, "{") || strings.HasPrefix(strings.ToLower(segment), "v") && len(segment) <= 3 {
			continue
		}

		return normalizeName(segment)
	}

	return "misc"
}

func normalizeCommand(method string, path string, group string, operationID string) string {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	last := ""
	if len(segments) > 0 {
		last = segments[len(segments)-1]
	}

	pathEndsWithParam := strings.HasPrefix(last, "{") && strings.HasSuffix(last, "}")
	if pathEndsWithParam {
		switch method {
		case "GET":
			return "get"
		case "PUT", "PATCH":
			return "update"
		case "DELETE":
			return "delete"
		}
	}

	switch method {
	case "GET":
		return "list"
	case "POST":
		if last != "" && !strings.EqualFold(normalizeName(last), group) && !pathEndsWithParam {
			return normalizeName(last)
		}
		return "create"
	case "PUT", "PATCH":
		return "update"
	case "DELETE":
		return "delete"
	default:
		if operationID != "" {
			return normalizeName(operationID)
		}
		return normalizeName(strings.ToLower(method))
	}
}

func detectPagination(params []Parameter) string {
	var foundLimit bool
	var foundPage bool
	var foundCursor bool
	for _, parameter := range params {
		switch strings.ToLower(parameter.Name) {
		case "limit", "pageSize":
			foundLimit = true
		case "page":
			foundPage = true
		case "cursor", "nextCursor":
			foundCursor = true
		}
	}

	switch {
	case foundCursor:
		return "cursor"
	case foundLimit || foundPage:
		return "page"
	default:
		return ""
	}
}

func normalizeName(value string) string {
	value = strings.TrimSpace(value)
	value = splitCamelCase(value)
	replacer := strings.NewReplacer(".", "-", "_", "-", " ", "-", "/", "-", "{", "", "}", "")
	value = replacer.Replace(value)
	value = strings.ToLower(value)
	value = strings.Trim(value, "-")
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	if value == "" {
		return "unnamed"
	}
	return value
}

func splitCamelCase(value string) string {
	if value == "" {
		return value
	}

	var builder strings.Builder
	runes := []rune(value)
	for index, r := range runes {
		if index > 0 && needsWordBreak(runes[index-1], r, nextRune(runes, index+1)) {
			builder.WriteByte('-')
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

func nextRune(values []rune, index int) rune {
	if index >= len(values) {
		return 0
	}
	return values[index]
}

func needsWordBreak(previous rune, current rune, next rune) bool {
	if isLower(previous) && isUpper(current) {
		return true
	}
	if isUpper(previous) && isUpper(current) && next != 0 && isLower(next) {
		return true
	}
	return false
}

func isLower(value rune) bool {
	return value >= 'a' && value <= 'z'
}

func isUpper(value rune) bool {
	return value >= 'A' && value <= 'Z'
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}

	return ""
}
