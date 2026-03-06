package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Robben-Media/jobtread-cli/internal/config"
)

type Client struct {
	baseURL    string
	token      string
	authScheme string
	httpClient *http.Client
}

type Request struct {
	Method      string
	Path        string
	Query       map[string]string
	Headers     map[string]string
	Body        []byte
	ContentType string
}

func New(cfg config.Config) *Client {
	return &Client{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		token:      cfg.Token,
		authScheme: cfg.AuthScheme,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) Do(ctx context.Context, req Request) ([]byte, int, error) {
	u, err := url.Parse(c.baseURL + normalizePath(req.Path))
	if err != nil {
		return nil, 0, fmt.Errorf("build request url: %w", err)
	}

	values := u.Query()
	for key, value := range req.Query {
		values.Set(key, value)
	}
	u.RawQuery = values.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, u.String(), bytes.NewReader(req.Body))
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}

	if req.ContentType != "" {
		httpReq.Header.Set("Content-Type", req.ContentType)
	}
	httpReq.Header.Set("Accept", "application/json")

	if c.token != "" {
		httpReq.Header.Set("Authorization", strings.TrimSpace(c.authScheme)+" "+c.token)
	}

	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, 0, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		if len(body) == 0 {
			return nil, resp.StatusCode, fmt.Errorf("request failed with status %s", resp.Status)
		}

		return nil, resp.StatusCode, fmt.Errorf("request failed with status %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	return prettyJSON(body), resp.StatusCode, nil
}

func normalizePath(path string) string {
	if strings.HasPrefix(path, "/") {
		return path
	}

	return "/" + path
}

func prettyJSON(body []byte) []byte {
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return body
	}

	pretty, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		return body
	}

	return pretty
}
