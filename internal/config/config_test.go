package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("JOBTREAD_BASE_URL", "")
	t.Setenv("JOBTREAD_API_TOKEN", "")
	t.Setenv("JOBTREAD_AUTH_SCHEME", "")

	cfg := Load()

	if cfg.BaseURL != "https://api.jobtread.com" {
		t.Fatalf("expected default base url, got %q", cfg.BaseURL)
	}

	if cfg.AuthScheme != "Bearer" {
		t.Fatalf("expected default auth scheme, got %q", cfg.AuthScheme)
	}

	if cfg.Timeout != 60*time.Second {
		t.Fatalf("expected default timeout, got %s", cfg.Timeout)
	}
}

func TestValidateRequestAuth(t *testing.T) {
	cfg := Config{}
	if err := cfg.ValidateRequestAuth(); err == nil {
		t.Fatal("expected missing token validation error")
	}
}
