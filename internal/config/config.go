package config

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/Robben-Media/jobtread-cli/internal/auth"
)

const (
	defaultBaseURL    = "https://api.jobtread.com"
	defaultAuthScheme = "Bearer"
	defaultTimeout    = 60 * time.Second
)

type Config struct {
	BaseURL         string
	Token           string
	AuthScheme      string
	KeychainService string
	Timeout         time.Duration
}

func Load() Config {
	timeout := defaultTimeout
	if value := strings.TrimSpace(os.Getenv("JOBTREAD_TIMEOUT")); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			timeout = parsed
		}
	}

	return Config{
		BaseURL:         firstNonEmpty(os.Getenv("JOBTREAD_BASE_URL"), defaultBaseURL),
		Token:           strings.TrimSpace(os.Getenv("JOBTREAD_API_TOKEN")),
		AuthScheme:      firstNonEmpty(strings.TrimSpace(os.Getenv("JOBTREAD_AUTH_SCHEME")), defaultAuthScheme),
		KeychainService: strings.TrimSpace(os.Getenv("JOBTREAD_KEYCHAIN_SERVICE")),
		Timeout:         timeout,
	}
}

func (c Config) ValidateRequestAuth() error {
	if strings.TrimSpace(c.Token) == "" {
		return errors.New("JobTread token is required for authenticated API requests")
	}

	return nil
}

func (c Config) ResolveToken(ctx context.Context) (string, string, error) {
	if strings.TrimSpace(c.Token) != "" {
		return strings.TrimSpace(c.Token), "config", nil
	}

	return auth.NewResolver().Resolve(ctx, c.KeychainService)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}

	return ""
}
