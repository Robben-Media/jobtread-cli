package pave

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Robben-Media/jobtread-cli/internal/httpclient"
)

type Client struct {
	endpoint   string
	grantKey   string
	timeZone   string
	httpClient *http.Client
}

func New(endpoint string, grantKey string, timeout time.Duration, timeZone string) *Client {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if !strings.HasSuffix(endpoint, "/pave") {
		endpoint += "/pave"
	}

	return &Client{
		endpoint:   endpoint,
		grantKey:   strings.TrimSpace(grantKey),
		timeZone:   strings.TrimSpace(timeZone),
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *Client) Execute(ctx context.Context, query map[string]any, public bool) (httpclient.Response, error) {
	payload := map[string]any{
		"query": injectContext(query, c.grantKey, c.timeZone, public),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return httpclient.Response{}, fmt.Errorf("encode pave query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return httpclient.Response{}, fmt.Errorf("build pave request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return httpclient.Response{}, fmt.Errorf("execute pave request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return httpclient.Response{}, fmt.Errorf("read pave response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		if len(raw) == 0 {
			return httpclient.Response{}, fmt.Errorf("pave request failed with status %s", resp.Status)
		}
		return httpclient.Response{}, fmt.Errorf("pave request failed with status %s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}

	return httpclient.Response{
		Body:       raw,
		StatusCode: resp.StatusCode,
		Headers:    resp.Header.Clone(),
		Method:     http.MethodPost,
		URL:        c.endpoint,
		Duration:   time.Since(start),
	}, nil
}

func injectContext(query map[string]any, grantKey string, timeZone string, public bool) map[string]any {
	cloned := make(map[string]any, len(query)+1)
	for key, value := range query {
		cloned[key] = value
	}

	rootContext := make(map[string]any)
	if existing, ok := cloned["$"].(map[string]any); ok {
		for key, value := range existing {
			rootContext[key] = value
		}
	}
	if !public && strings.TrimSpace(grantKey) != "" {
		rootContext["grantKey"] = grantKey
	}
	if strings.TrimSpace(timeZone) != "" {
		rootContext["timeZone"] = timeZone
	}
	if len(rootContext) > 0 {
		cloned["$"] = rootContext
	}

	return cloned
}
