package output

import (
	"strings"
	"testing"
)

func TestProcessAutoChoosesTOONForUniformArray(t *testing.T) {
	body := []byte(`[{"id":1,"name":"A"},{"id":2,"name":"B"}]`)
	result, format, err := Process(body, FormatAuto, "")
	if err != nil {
		t.Fatalf("unexpected process error: %v", err)
	}

	if format != FormatTOON {
		t.Fatalf("expected TOON format, got %s", format)
	}

	if !strings.Contains(string(result), "items[2]{id,name}:") {
		t.Fatalf("unexpected TOON output %q", string(result))
	}
}

func TestProcessAppliesJMESPathBeforeFormatting(t *testing.T) {
	body := []byte(`{"data":[{"id":1},{"id":2}]}`)
	result, format, err := Process(body, FormatAuto, "data")
	if err != nil {
		t.Fatalf("unexpected process error: %v", err)
	}

	if format != FormatTOON {
		t.Fatalf("expected TOON after query, got %s", format)
	}

	if !strings.Contains(string(result), "items[2]{id}:") {
		t.Fatalf("unexpected queried output %q", string(result))
	}
}
