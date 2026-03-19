package catalog

import "testing"

func TestImportPaveSchema(t *testing.T) {
	raw := []byte(`{
  "schema": {
    "root": {
      "type": {
        "object": {
          "version": {
            "description": "Current version",
            "type": "string"
          },
          "createContact": {
            "description": "Create a contact",
            "input": {
              "object": {
                "name": { "type": "string" },
                "notify": { "type": "boolean", "defaultValue": true }
              }
            },
            "type": {
              "nullable": "contact"
            }
          }
        }
      }
    },
    "contact": {
      "type": {
        "object": {
          "id": { "type": "jobtreadId" },
          "name": { "type": "string" }
        }
      }
    }
  }
}`)

	manifest, err := ImportPaveSchema(raw, "memory://pave")
	if err != nil {
		t.Fatalf("unexpected import error: %v", err)
	}

	if len(manifest.Operations) != 2 {
		t.Fatalf("expected 2 operations, got %d", len(manifest.Operations))
	}

	create := manifest.Operations[0]
	if create.Transport != "pave" {
		t.Fatalf("expected pave transport, got %q", create.Transport)
	}
	if create.Group != "contacts" {
		t.Fatalf("expected contacts group, got %q", create.Group)
	}
	if create.Command != "create" {
		t.Fatalf("expected create command, got %q", create.Command)
	}
	if len(create.InputFields) != 2 {
		t.Fatalf("expected input fields, got %d", len(create.InputFields))
	}
}
