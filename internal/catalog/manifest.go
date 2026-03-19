package catalog

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

//go:embed api_manifest.json
var embeddedManifest []byte

const defaultManifestPath = "internal/catalog/api_manifest.json"

type Manifest struct {
	Source     SourceMeta  `json:"source"`
	Operations []Operation `json:"operations"`
}

type SourceMeta struct {
	Kind           string `json:"kind"`
	URI            string `json:"uri"`
	ImportedAt     string `json:"imported_at"`
	OperationCount int    `json:"operation_count"`
	Checksum       string `json:"checksum"`
}

type Operation struct {
	Transport          string       `json:"transport,omitempty"`
	Group              string       `json:"group"`
	Command            string       `json:"command"`
	Method             string       `json:"method,omitempty"`
	PathTemplate       string       `json:"path_template,omitempty"`
	OperationID        string       `json:"operation_id"`
	Summary            string       `json:"summary"`
	Tags               []string     `json:"tags,omitempty"`
	PathParams         []Parameter  `json:"path_params,omitempty"`
	QueryParams        []Parameter  `json:"query_params,omitempty"`
	HasRequestBody     bool         `json:"has_request_body"`
	PaginationMode     string       `json:"pagination_mode,omitempty"`
	FilterableFields   []string     `json:"filterable_fields,omitempty"`
	ExpandableFields   []string     `json:"expandable_fields,omitempty"`
	RawFallbackAllowed bool         `json:"raw_fallback_allowed"`
	PaveField          string       `json:"pave_field,omitempty"`
	Public             bool         `json:"public,omitempty"`
	SupportsPagination bool         `json:"supports_pagination,omitempty"`
	InputFields        []InputField `json:"input_fields,omitempty"`
	DefaultSelection   map[string]any `json:"default_selection,omitempty"`
}

type Parameter struct {
	Name     string `json:"name"`
	In       string `json:"in"`
	Required bool   `json:"required"`
}

type InputField struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
}

func Load() (Manifest, error) {
	if manifestPath := strings.TrimSpace(os.Getenv("JOBTREAD_MANIFEST")); manifestPath != "" {
		return LoadFile(manifestPath)
	}

	if _, err := os.Stat(defaultManifestPath); err == nil {
		return LoadFile(defaultManifestPath)
	}

	return decodeManifest(embeddedManifest)
}

func LoadFile(path string) (Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest: %w", err)
	}

	return decodeManifest(raw)
}

func Save(path string, manifest Manifest) error {
	manifest.sort()
	manifest.Source.OperationCount = len(manifest.Operations)

	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}

	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	return nil
}

func (m Manifest) Validate() error {
	if m.Source.OperationCount != len(m.Operations) {
		return fmt.Errorf("manifest operation count mismatch: source=%d manifest=%d", m.Source.OperationCount, len(m.Operations))
	}

	seen := make(map[string]struct{}, len(m.Operations))
	for _, operation := range m.Operations {
		if strings.TrimSpace(operation.Group) == "" {
			return fmt.Errorf("operation %q missing group", operation.OperationID)
		}
		if strings.TrimSpace(operation.Command) == "" {
			return fmt.Errorf("operation %q missing command", operation.OperationID)
		}

		switch strings.TrimSpace(operation.Transport) {
		case "", "http":
			if strings.TrimSpace(operation.Method) == "" || strings.TrimSpace(operation.PathTemplate) == "" {
				return fmt.Errorf("operation %q missing method or path", operation.OperationID)
			}
		case "pave":
			if strings.TrimSpace(operation.PaveField) == "" {
				return fmt.Errorf("operation %q missing pave field", operation.OperationID)
			}
		default:
			return fmt.Errorf("operation %q uses unsupported transport %q", operation.OperationID, operation.Transport)
		}

		key := operation.Group + ":" + operation.Command + ":" + operation.Transport + ":" + operation.OperationID
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate manifest command mapping: %s", key)
		}
		seen[key] = struct{}{}
	}

	return nil
}

func (m Manifest) IsPopulated() bool {
	return len(m.Operations) > 0
}

func NewSourceMeta(kind string, uri string, raw []byte) SourceMeta {
	sum := sha256.Sum256(raw)
	return SourceMeta{
		Kind:           kind,
		URI:            uri,
		ImportedAt:     time.Now().UTC().Format(time.RFC3339),
		OperationCount: 0,
		Checksum:       hex.EncodeToString(sum[:]),
	}
}

func decodeManifest(raw []byte) (Manifest, error) {
	var manifest Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode manifest: %w", err)
	}

	manifest.sort()
	return manifest, nil
}

func (m *Manifest) sort() {
	sort.Slice(m.Operations, func(i, j int) bool {
		left := m.Operations[i].Group + "/" + m.Operations[i].Command + "/" + m.Operations[i].Transport + "/" + m.Operations[i].OperationID
		right := m.Operations[j].Group + "/" + m.Operations[j].Command + "/" + m.Operations[j].Transport + "/" + m.Operations[j].OperationID
		return left < right
	})
}
