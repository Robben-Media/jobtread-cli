package httpclient

import (
	"bytes"
	"context"
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

type Response struct {
	Body       []byte
	StatusCode int
	Headers    http.Header
	Method     string
	URL        string
	Duration   time.Duration
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
		httpClient: &http.Client{Timeout: cfg.Timeout},
	}
}

func (c *Client) Do(ctx context.Context, req Request) (Response, error) {
	u, err := url.Parse(c.baseURL + normalizePath(req.Path))
	if err != nil {
		return Response{}, fmt.Errorf("build request url: %w", err)
	}

	values := u.Query()
	for key, value := range req.Query {
		values.Set(key, value)
	}
	u.RawQuery = values.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, u.String(), bytes.NewReader(req.Body))
	if err != nil {
		return Response{}, fmt.Errorf("build request: %w", err)
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

	start := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return Response{}, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		if len(body) == 0 {
			return Response{}, fmt.Errorf("request failed with status %s", resp.Status)
		}

		return Response{}, fmt.Errorf("request failed with status %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	return Response{
		Body:       body,
		StatusCode: resp.StatusCode,
		Headers:    resp.Header.Clone(),
		Method:     req.Method,
		URL:        u.String(),
		Duration:   time.Since(start),
	}, nil
}

func normalizePath(path string) string {
	if strings.HasPrefix(path, "/") {
		return path
	}

	return "/" + path
}
