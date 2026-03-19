package output

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	jmespath "github.com/jmespath/go-jmespath"
)

type Format string

const (
	FormatAuto  Format = "auto"
	FormatJSON  Format = "json"
	FormatTOON  Format = "toon"
	FormatTable Format = "table"
	FormatRaw   Format = "raw"
)

func Process(body []byte, format Format, query string) ([]byte, Format, error) {
	if format == FormatRaw && strings.TrimSpace(query) == "" {
		return body, FormatRaw, nil
	}

	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		if format == FormatRaw {
			return body, FormatRaw, nil
		}
		return nil, "", fmt.Errorf("decode response JSON: %w", err)
	}

	if strings.TrimSpace(query) != "" {
		result, err := jmespath.Search(query, decoded)
		if err != nil {
			return nil, "", fmt.Errorf("apply query: %w", err)
		}
		decoded = result
	}

	if format == FormatAuto {
		if canEncodeTabularToon(decoded) {
			format = FormatTOON
		} else {
			format = FormatJSON
		}
	}

	switch format {
	case FormatJSON:
		rendered, err := json.Marshal(decoded)
		return rendered, format, err
	case FormatTOON:
		rendered, err := EncodeTOON(decoded)
		return rendered, format, err
	case FormatTable:
		rendered, err := encodeTable(decoded)
		return rendered, format, err
	case FormatRaw:
		rendered, err := json.Marshal(decoded)
		return rendered, format, err
	default:
		return nil, "", fmt.Errorf("unsupported output format %q", format)
	}
}

func canEncodeTabularToon(value any) bool {
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		return false
	}

	keys, ok := objectKeys(items[0])
	if !ok || len(keys) == 0 {
		return false
	}

	for _, item := range items[1:] {
		itemKeys, itemOK := objectKeys(item)
		if !itemOK || len(itemKeys) != len(keys) {
			return false
		}
		for index := range keys {
			if itemKeys[index] != keys[index] {
				return false
			}
		}
	}

	return true
}

func EncodeTOON(value any) ([]byte, error) {
	lines, err := encodeNode("items", value, "", true)
	if err != nil {
		return nil, err
	}

	return []byte(strings.Join(lines, "\n")), nil
}

func encodeNode(name string, value any, indent string, root bool) ([]string, error) {
	switch typed := value.(type) {
	case map[string]any:
		keys, _ := objectKeys(typed)
		lines := make([]string, 0, len(keys)+1)
		if !root {
			lines = append(lines, indent+name+":")
			indent += "  "
		}
		for _, key := range keys {
			child, err := encodeNode(key, typed[key], indent, false)
			if err != nil {
				return nil, err
			}
			lines = append(lines, child...)
		}
		return lines, nil
	case []any:
		return encodeArray(name, typed, indent, root)
	case string, float64, bool, nil, int:
		return []string{indent + scalarLine(name, typed)}, nil
	default:
		raw, err := json.Marshal(typed)
		if err != nil {
			return nil, fmt.Errorf("encode toon fallback: %w", err)
		}
		return []string{indent + scalarLine(name, string(raw))}, nil
	}
}

func encodeArray(name string, items []any, indent string, root bool) ([]string, error) {
	label := name
	if root {
		label = "items"
	}

	if len(items) == 0 {
		return []string{fmt.Sprintf("%s%s[0]:", indent, label)}, nil
	}

	if keys, ok := objectKeys(items[0]); ok {
		if canEncodeTabularToon(items) {
			lines := []string{fmt.Sprintf("%s%s[%d]{%s}:", indent, label, len(items), strings.Join(keys, ","))}
			for _, item := range items {
				rowMap := item.(map[string]any)
				record := make([]string, 0, len(keys))
				for _, key := range keys {
					record = append(record, csvScalar(rowMap[key]))
				}
				lines = append(lines, indent+"  "+strings.Join(record, ","))
			}
			return lines, nil
		}
	}

	primitives := true
	record := make([]string, 0, len(items))
	for _, item := range items {
		if !isScalar(item) {
			primitives = false
			break
		}
		record = append(record, csvScalar(item))
	}
	if primitives {
		return []string{fmt.Sprintf("%s%s[%d]: %s", indent, label, len(items), strings.Join(record, ","))}, nil
	}

	lines := []string{fmt.Sprintf("%s%s[%d]:", indent, label, len(items))}
	for _, item := range items {
		child, err := encodeNode("-", item, indent+"  ", false)
		if err != nil {
			return nil, err
		}
		if len(child) == 1 && strings.HasPrefix(child[0], indent+"  -:") {
			lines = append(lines, strings.Replace(child[0], "-:", "-", 1))
			continue
		}
		lines = append(lines, indent+"  -")
		lines = append(lines, child...)
	}
	return lines, nil
}

func scalarLine(name string, value any) string {
	return name + ": " + scalarValue(value)
}

func scalarValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return "null"
	case string:
		if needsQuotes(typed) {
			return strconv.Quote(typed)
		}
		return typed
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return fmt.Sprint(typed)
	}
}

func csvScalar(value any) string {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	_ = writer.Write([]string{scalarValue(value)})
	writer.Flush()
	return strings.TrimSpace(buffer.String())
}

func objectKeys(value any) ([]string, bool) {
	typed, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	keys := make([]string, 0, len(typed))
	for key, child := range typed {
		if !isScalar(child) {
			return nil, false
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys, true
}

func isScalar(value any) bool {
	switch value.(type) {
	case nil, string, float64, bool, int:
		return true
	default:
		return false
	}
}

func needsQuotes(value string) bool {
	return value == "" || strings.ContainsAny(value, ",\n\r\"")
}

func encodeTable(value any) ([]byte, error) {
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		return nil, fmt.Errorf("table output requires a non-empty array")
	}

	keys, ok := objectKeys(items[0])
	if !ok {
		return nil, fmt.Errorf("table output requires an array of flat objects")
	}

	var builder strings.Builder
	builder.WriteString(strings.Join(keys, "\t"))
	builder.WriteByte('\n')
	for index, item := range items {
		if index > 0 {
			builder.WriteByte('\n')
		}
		row := item.(map[string]any)
		cells := make([]string, 0, len(keys))
		for _, key := range keys {
			cells = append(cells, scalarValue(row[key]))
		}
		builder.WriteString(strings.Join(cells, "\t"))
	}

	return []byte(builder.String()), nil
}
