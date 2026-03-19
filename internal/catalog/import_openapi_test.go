package catalog

import "testing"

func TestImportOpenAPI(t *testing.T) {
	raw := []byte(`
openapi: 3.0.0
paths:
  /v1/jobs:
    get:
      operationId: listJobs
      tags: [Jobs]
      parameters:
        - name: limit
          in: query
          required: false
    post:
      operationId: createJob
      tags: [Jobs]
      requestBody:
        required: true
  /v1/jobs/{jobId}:
    get:
      operationId: getJob
      tags: [Jobs]
      parameters:
        - name: jobId
          in: path
          required: true
`)

	manifest, err := ImportOpenAPI(raw, "memory://test")
	if err != nil {
		t.Fatalf("unexpected import error: %v", err)
	}

	if len(manifest.Operations) != 3 {
		t.Fatalf("expected 3 operations, got %d", len(manifest.Operations))
	}

	if manifest.Operations[0].Group != "jobs" {
		t.Fatalf("expected jobs group, got %q", manifest.Operations[0].Group)
	}
}
