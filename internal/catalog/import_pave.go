package catalog

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type paveSchemaEnvelope struct {
	Schema map[string]any `json:"schema"`
}

var scalarTypeNames = map[string]struct{}{
	"base64url":  {},
	"boolean":    {},
	"countryCode": {},
	"date":       {},
	"datetime":   {},
	"int":        {},
	"jobtreadId": {},
	"number":     {},
	"string":     {},
	"time":       {},
	"timeZone":   {},
}

func ImportPaveSchema(raw []byte, sourceURI string) (Manifest, error) {
	schema, err := decodePaveSchema(raw)
	if err != nil {
		return Manifest{}, err
	}

	rootObject := objectFields(schema["root"])
	if len(rootObject) == 0 {
		return Manifest{}, fmt.Errorf("pave schema missing root object")
	}

	manifest := Manifest{
		Source: NewSourceMeta("jobtread-pave-schema", sourceURI, raw),
	}

	names := make([]string, 0, len(rootObject))
	for name := range rootObject {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		definition, ok := asMap(rootObject[name])
		if !ok {
			continue
		}

		group, command := normalizePaveCommand(name)
		manifest.Operations = append(manifest.Operations, Operation{
			Transport:          "pave",
			Group:              group,
			Command:            command,
			OperationID:        name,
			Summary:            stringValue(definition["description"]),
			HasRequestBody:     len(inputFields(definition["input"])) > 0,
			PaginationMode:     paginationModeForPave(schema, definition["type"]),
			RawFallbackAllowed: true,
			PaveField:          name,
			Public:             isPublicPaveOperation(name),
			SupportsPagination: isConnectionType(schema, definition["type"]),
			InputFields:        inputFields(definition["input"]),
			DefaultSelection:   buildDefaultSelection(schema, definition["type"], 0),
		})
	}

	manifest.Source.OperationCount = len(manifest.Operations)
	manifest.sort()
	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}

	return manifest, nil
}

func decodePaveSchema(raw []byte) (map[string]any, error) {
	var envelope paveSchemaEnvelope
	if err := json.Unmarshal(raw, &envelope); err == nil && len(envelope.Schema) > 0 {
		return envelope.Schema, nil
	}

	var direct map[string]any
	if err := json.Unmarshal(raw, &direct); err != nil {
		return nil, fmt.Errorf("decode pave schema: %w", err)
	}
	if schema, ok := asMap(direct["schema"]); ok && len(schema) > 0 {
		return schema, nil
	}

	return direct, nil
}

func normalizePaveCommand(name string) (string, string) {
	lower := strings.TrimSpace(name)
	switch {
	case hasAnyPrefix(lower, "create", "update", "delete", "find", "query", "read", "draft", "export", "cancel", "copy", "close", "open", "register"):
		action, resource := splitVerbResource(lower)
		group := normalizePaveGroup(resource)
		if group == "misc" {
			group = normalizePaveGroup(action)
		}
		command := normalizeName(action)
		if command == "" || command == "read" {
			command = "get"
		}
		if command == "query" {
			command = "find"
		}
		return group, command
	default:
		return normalizePaveGroup(lower), "get"
	}
}

func normalizePaveGroup(resource string) string {
	resource = strings.TrimSpace(resource)
	if resource == "" {
		return "misc"
	}

	group := singularize(normalizeName(resource))
	if group == "" || group == "root" {
		return "system"
	}
	return pluralize(group)
}

func splitVerbResource(name string) (string, string) {
	prefixes := []string{"create", "update", "delete", "find", "query", "read", "draft", "export", "cancel", "copy", "close", "open", "register"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) && len(name) > len(prefix) {
			return prefix, name[len(prefix):]
		}
	}
	return name, ""
}

func hasAnyPrefix(value string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func singularize(value string) string {
	switch {
	case strings.HasSuffix(value, "ies"):
		return strings.TrimSuffix(value, "ies") + "y"
	case strings.HasSuffix(value, "ses"),
		strings.HasSuffix(value, "xes"),
		strings.HasSuffix(value, "zes"),
		strings.HasSuffix(value, "ches"),
		strings.HasSuffix(value, "shes"):
		return strings.TrimSuffix(value, "es")
	case strings.HasSuffix(value, "s") && !strings.HasSuffix(value, "ss"):
		return strings.TrimSuffix(value, "s")
	default:
		return value
	}
}

func pluralize(value string) string {
	switch {
	case strings.HasSuffix(value, "y") && len(value) > 1 && !strings.ContainsRune("aeiou", rune(value[len(value)-2])):
		return strings.TrimSuffix(value, "y") + "ies"
	case strings.HasSuffix(value, "s"),
		strings.HasSuffix(value, "x"),
		strings.HasSuffix(value, "z"),
		strings.HasSuffix(value, "ch"),
		strings.HasSuffix(value, "sh"):
		return value + "es"
	default:
		return value + "s"
	}
}

func inputFields(input any) []InputField {
	definition, ok := asMap(input)
	if !ok {
		return nil
	}

	fields := objectMembers(definition["object"])
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]InputField, 0, len(fields))
	for _, name := range names {
		fieldDef := fields[name]
		result = append(result, InputField{
			Name:        name,
			Type:        describePaveType(fieldDef),
			Description: stringValue(asMapValue(fieldDef, "description")),
			Required:    isRequiredField(fieldDef),
		})
	}
	return result
}

func isRequiredField(field any) bool {
	definition, ok := asMap(field)
	if !ok {
		return true
	}

	if _, ok := definition["defaultValue"]; ok {
		return false
	}
	if _, ok := definition["nullable"]; ok {
		return false
	}
	typeDef, ok := definition["type"]
	if !ok {
		return true
	}
	if typed, ok := asMap(typeDef); ok {
		if _, nullable := typed["nullable"]; nullable {
			return false
		}
	}
	return true
}

func paginationModeForPave(schema map[string]any, typeDef any) string {
	if !isConnectionType(schema, typeDef) {
		return ""
	}
	return "page"
}

func isConnectionType(schema map[string]any, typeDef any) bool {
	fields := resolvedObjectFields(schema, typeDef)
	if len(fields) == 0 {
		return false
	}
	_, hasNodes := fields["nodes"]
	_, hasNextPage := fields["nextPage"]
	return hasNodes && hasNextPage
}

func buildDefaultSelection(schema map[string]any, typeDef any, depth int) map[string]any {
	if depth > 2 {
		return map[string]any{}
	}

	fields := resolvedObjectFields(schema, typeDef)
	if len(fields) == 0 {
		return map[string]any{}
	}

	if _, ok := fields["nodes"]; ok {
		selection := map[string]any{
			"nextPage": map[string]any{},
			"nodes":    buildDefaultSelection(schema, fieldType(fields["nodes"]), depth+1),
		}
		if _, hasCount := fields["count"]; hasCount {
			selection["count"] = map[string]any{}
		}
		return selection
	}

	preferred := []string{"id", "name", "number", "status", "emailAddress", "createdAt", "updatedAt", "closedOn"}
	selection := make(map[string]any)
	for _, name := range preferred {
		field, ok := fields[name]
		if !ok || !isScalarLike(schema, fieldType(field)) {
			continue
		}
		selection[name] = map[string]any{}
	}

	if len(selection) == 0 {
		names := make([]string, 0, len(fields))
		for name, field := range fields {
			if isScalarLike(schema, fieldType(field)) {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		for _, name := range names {
			selection[name] = map[string]any{}
			if len(selection) == 4 {
				break
			}
		}
	}

	if len(selection) > 0 {
		return selection
	}

	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		child := buildDefaultSelection(schema, fieldType(fields[name]), depth+1)
		if len(child) == 0 {
			continue
		}
		return map[string]any{name: child}
	}

	return map[string]any{}
}

func describePaveType(field any) string {
	switch typed := field.(type) {
	case string:
		return typed
	case map[string]any:
		switch {
		case typed["nullable"] != nil:
			return "nullable(" + describePaveType(typed["nullable"]) + ")"
		case typed["arrayOf"] != nil:
			return "array(" + describePaveType(typed["arrayOf"]) + ")"
		case typed["oneOf"] != nil:
			return "enum"
		case typed["object"] != nil:
			return "object"
		case typed["type"] != nil:
			return describePaveType(typed["type"])
		default:
			return "any"
		}
	default:
		return "any"
	}
}

func resolvedObjectFields(schema map[string]any, typeDef any) map[string]any {
	switch typed := typeDef.(type) {
	case string:
		if definition, ok := asMap(schema[typed]); ok {
			return objectFields(definition)
		}
	case map[string]any:
		switch {
		case typed["nullable"] != nil:
			return resolvedObjectFields(schema, typed["nullable"])
		case typed["arrayOf"] != nil:
			return resolvedObjectFields(schema, typed["arrayOf"])
		case typed["type"] != nil:
			return resolvedObjectFields(schema, typed["type"])
		case typed["object"] != nil:
			return objectMembers(typed["object"])
		}
	}

	return nil
}

func fieldType(field any) any {
	switch typed := field.(type) {
	case string:
		return typed
	case map[string]any:
		if typeDef, ok := typed["type"]; ok {
			return typeDef
		}
		return typed
	default:
		return field
	}
}

func isScalarLike(schema map[string]any, typeDef any) bool {
	switch typed := typeDef.(type) {
	case string:
		if _, ok := scalarTypeNames[typed]; ok {
			return true
		}
		definition, ok := asMap(schema[typed])
		if !ok {
			return true
		}
		if _, ok := definition["oneOf"]; ok {
			return true
		}
		return len(objectFields(definition)) == 0
	case map[string]any:
		switch {
		case typed["nullable"] != nil:
			return isScalarLike(schema, typed["nullable"])
		case typed["arrayOf"] != nil:
			return false
		case typed["oneOf"] != nil:
			return true
		case typed["object"] != nil:
			return false
		case typed["type"] != nil:
			return isScalarLike(schema, typed["type"])
		default:
			return true
		}
	default:
		return true
	}
}

func objectFields(definition any) map[string]any {
	if typed, ok := asMap(definition); ok {
		if typeDef, ok := typed["type"]; ok {
			if nested, ok := asMap(typeDef); ok {
				if fields, ok := nested["object"]; ok {
					return objectMembers(fields)
				}
			}
		}
		if fields, ok := typed["object"]; ok {
			return objectMembers(fields)
		}
	}
	return nil
}

func objectMembers(value any) map[string]any {
	typed, ok := asMap(value)
	if !ok {
		return nil
	}
	return typed
}

func asMap(value any) (map[string]any, bool) {
	typed, ok := value.(map[string]any)
	return typed, ok
}

func asMapValue(value any, key string) any {
	typed, ok := asMap(value)
	if !ok {
		return nil
	}
	return typed[key]
}

func stringValue(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func isPublicPaveOperation(name string) bool {
	switch name {
	case "captchaToken", "createGrant", "schema", "version":
		return true
	default:
		return false
	}
}
