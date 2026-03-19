package auth

import (
	"context"
	"errors"
	"testing"
)

func TestResolverUsesOverrideWithoutEnumeration(t *testing.T) {
	var calls [][]string
	resolver := Resolver{
		run: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			calls = append(calls, append([]string{name}, args...))
			if len(args) >= 3 && args[2] == "exact-item" {
				return []byte("token-123"), nil
			}
			return nil, errors.New("not found")
		},
	}

	token, source, err := resolver.Resolve(context.Background(), "exact-item")
	if err != nil {
		t.Fatalf("unexpected resolve error: %v", err)
	}

	if token != "token-123" {
		t.Fatalf("unexpected token %q", token)
	}

	if source != "keychain:exact-item" {
		t.Fatalf("unexpected source %q", source)
	}

	if got := calls[0][3]; got != "exact-item" {
		t.Fatalf("expected first lookup to use override service, got %q", got)
	}
}

func TestParseInternetAccount(t *testing.T) {
	raw := `
keychain: "/Users/test/Library/Keychains/login.keychain-db"
attributes:
    "acct"<blob>="person@example.com"
    "srvr"<blob>="app.jobtread.com"
`

	account := parseInternetAccount(raw)
	if account != "person@example.com" {
		t.Fatalf("unexpected account %q", account)
	}
}
