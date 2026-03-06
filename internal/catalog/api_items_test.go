package catalog

import "testing"

func TestLoad(t *testing.T) {
	catalog, err := Load()
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if len(catalog.Items) == 0 {
		t.Fatal("expected seeded API item catalog")
	}
}
