package config

import (
	"errors"
	"os"
	"strings"
)

const (
	defaultBaseURL    = "https://api.jobtread.com"
	defaultAuthScheme = "Bearer"
)

type Config struct {
	BaseURL    string
	Token      string
	AuthScheme string
}

func Load() Config {
	return Config{
		BaseURL:    firstNonEmpty(os.Getenv("JOBTREAD_BASE_URL"), defaultBaseURL),
		Token:      strings.TrimSpace(os.Getenv("JOBTREAD_API_TOKEN")),
		AuthScheme: firstNonEmpty(strings.TrimSpace(os.Getenv("JOBTREAD_AUTH_SCHEME")), defaultAuthScheme),
	}
}

func (c Config) ValidateRequestAuth() error {
	if c.Token == "" {
		return errors.New("JOBTREAD_API_TOKEN is required for authenticated API requests")
	}

	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}

	return ""
}
