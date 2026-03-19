package catalog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const DefaultPaveSchemaURL = "https://api.jobtread.com/pave"

func ReadSource(ctx context.Context, source string, token string, authScheme string) ([]byte, string, error) {
	if source == "" {
		return nil, "", fmt.Errorf("source is required")
	}

	if isHTTPSource(source) {
		return fetchHTTPSource(ctx, source, token, authScheme)
	}

	raw, err := os.ReadFile(source)
	if err != nil {
		return nil, "", fmt.Errorf("read source file: %w", err)
	}

	return raw, source, nil
}

func FetchPaveSchema(ctx context.Context, source string) ([]byte, string, error) {
	if strings.TrimSpace(source) == "" {
		source = DefaultPaveSchemaURL
	}

	body := []byte(`{"query":{"schema":{}}}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, source, bytes.NewReader(body))
	if err != nil {
		return nil, "", fmt.Errorf("build pave schema request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("fetch pave schema: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read pave schema response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, "", fmt.Errorf("fetch pave schema failed with status %s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}

	return raw, source, nil
}

func isHTTPSource(source string) bool {
	return strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://")
}

func fetchHTTPSource(ctx context.Context, source string, token string, authScheme string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return nil, "", fmt.Errorf("build source request: %w", err)
	}

	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", strings.TrimSpace(authScheme)+" "+strings.TrimSpace(token))
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("fetch source: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read source response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, "", fmt.Errorf("fetch source failed with status %s", resp.Status)
	}

	return raw, source, nil
}
