package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var likelyServiceNames = []string{
	"jobtread",
	"JobTread",
	"jobtread-api",
	"api.jobtread.com",
	"app.jobtread.com",
}

type Resolver struct {
	run         commandRunner
	runCombined commandRunner
}

type commandRunner func(context.Context, string, ...string) ([]byte, error)

type Login struct {
	EmailAddress string
	Password     string
	Source       string
}

func NewResolver() Resolver {
	return Resolver{
		run: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			cmd := exec.CommandContext(ctx, name, args...)
			return cmd.Output()
		},
		runCombined: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			cmd := exec.CommandContext(ctx, name, args...)
			return cmd.CombinedOutput()
		},
	}
}

func (r Resolver) Resolve(ctx context.Context, overrideService string) (token string, source string, err error) {
	if envToken := strings.TrimSpace(os.Getenv("JOBTREAD_API_TOKEN")); envToken != "" {
		return envToken, "env:JOBTREAD_API_TOKEN", nil
	}

	services := make([]string, 0, len(likelyServiceNames)+1)
	if override := strings.TrimSpace(overrideService); override != "" {
		services = append(services, override)
	}

	for _, service := range likelyServiceNames {
		if !containsString(services, service) {
			services = append(services, service)
		}
	}

	for _, service := range services {
		for _, args := range serviceLookups(service) {
			value, lookupErr := r.run(ctx, "security", args...)
			if lookupErr != nil {
				continue
			}

			token = strings.TrimSpace(string(value))
			if token != "" {
				return token, "keychain:" + service, nil
			}
		}
	}

	return "", "", errors.New("unable to resolve JobTread credential from JOBTREAD_API_TOKEN or targeted keychain lookup; set JOBTREAD_API_TOKEN or JOBTREAD_KEYCHAIN_SERVICE")
}

func (r Resolver) ResolveLogin(ctx context.Context, overrideService string) (Login, error) {
	if email := strings.TrimSpace(os.Getenv("JOBTREAD_EMAIL")); email != "" {
		password := strings.TrimSpace(os.Getenv("JOBTREAD_PASSWORD"))
		if password == "" {
			return Login{}, errors.New("JOBTREAD_EMAIL is set but JOBTREAD_PASSWORD is empty")
		}
		return Login{
			EmailAddress: email,
			Password:     password,
			Source:       "env:JOBTREAD_EMAIL",
		}, nil
	}

	services := make([]string, 0, len(likelyServiceNames)+1)
	if override := strings.TrimSpace(overrideService); override != "" {
		services = append(services, override)
	}
	for _, service := range likelyServiceNames {
		if !containsString(services, service) {
			services = append(services, service)
		}
	}

	for _, service := range services {
		email, found := r.lookupInternetAccount(ctx, service)
		if !found {
			continue
		}
		for _, args := range [][]string{
			{"find-internet-password", "-s", service, "-w"},
			{"find-internet-password", "-r", "htps", "-s", service, "-w"},
		} {
			password, err := r.run(ctx, "security", args...)
			if err != nil {
				continue
			}
			secret := strings.TrimSpace(string(password))
			if secret == "" {
				continue
			}
			return Login{
				EmailAddress: email,
				Password:     secret,
				Source:       "keychain:" + service,
			}, nil
		}
	}

	return Login{}, errors.New("unable to resolve JobTread login from targeted keychain lookup; set JOBTREAD_EMAIL/JOBTREAD_PASSWORD or JOBTREAD_KEYCHAIN_SERVICE")
}

func serviceLookups(service string) [][]string {
	return [][]string{
		{"find-internet-password", "-s", service, "-w"},
		{"find-internet-password", "-r", "htps", "-s", service, "-w"},
		{"find-generic-password", "-s", service, "-w"},
	}
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}

	return false
}

func MaskSource(source string) string {
	if source == "" {
		return "unknown"
	}

	if strings.HasPrefix(source, "keychain:") {
		return source
	}

	return fmt.Sprintf("%s", source)
}

func (r Resolver) lookupInternetAccount(ctx context.Context, service string) (string, bool) {
	lookups := [][]string{
		{"find-internet-password", "-s", service},
		{"find-internet-password", "-r", "htps", "-s", service},
	}

	for _, args := range lookups {
		output, err := r.runCombined(ctx, "security", args...)
		if err != nil {
			continue
		}
		account := parseInternetAccount(string(output))
		if account != "" {
			return account, true
		}
	}

	return "", false
}

func parseInternetAccount(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "\"acct\"") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		value := strings.TrimSpace(parts[1])
		value = strings.TrimPrefix(value, "<blob>=")
		value = strings.Trim(value, "\"")
		if value != "" {
			return value
		}
	}
	return ""
}
