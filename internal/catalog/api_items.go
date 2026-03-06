package catalog

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed api_items.yaml
var rawItems []byte

type Catalog struct {
	Items []Item `yaml:"items"`
}

type Item struct {
	Name        string   `yaml:"name"`
	Method      string   `yaml:"method"`
	Path        string   `yaml:"path"`
	Description string   `yaml:"description"`
	Status      string   `yaml:"status"`
	Tags        []string `yaml:"tags"`
}

func Load() (Catalog, error) {
	var catalog Catalog
	if err := yaml.Unmarshal(rawItems, &catalog); err != nil {
		return Catalog{}, fmt.Errorf("decode API item catalog: %w", err)
	}

	sort.Slice(catalog.Items, func(i, j int) bool {
		return strings.ToLower(catalog.Items[i].Name) < strings.ToLower(catalog.Items[j].Name)
	})

	return catalog, nil
}
