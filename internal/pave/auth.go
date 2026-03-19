package pave

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Robben-Media/jobtread-cli/internal/auth"
	"github.com/Robben-Media/jobtread-cli/internal/config"
)

type grantEnvelope struct {
	CaptchaToken string `json:"captchaToken"`
	CreateGrant  struct {
		CreatedGrant struct {
			Key string `json:"key"`
		} `json:"createdGrant"`
	} `json:"createGrant"`
}

func ResolveGrantKey(ctx context.Context, cfg config.Config) (string, string, error) {
	if token := strings.TrimSpace(cfg.Token); token != "" {
		return token, "config", nil
	}

	login, err := auth.NewResolver().ResolveLogin(ctx, cfg.KeychainService)
	if err != nil {
		return "", "", err
	}

	client := New(cfg.BaseURL, "", cfg.Timeout, "")
	captchaToken, err := fetchCaptchaToken(ctx, client)
	if err != nil {
		return "", "", err
	}

	query := map[string]any{
		"createGrant": map[string]any{
			"$": map[string]any{
				"captchaToken": signCaptchaToken(captchaToken),
				"emailAddress": login.EmailAddress,
				"name":         "JobTread CLI",
				"password":     login.Password,
			},
			"createdGrant": map[string]any{
				"key": map[string]any{},
			},
		},
	}

	response, err := client.Execute(ctx, query, true)
	if err != nil {
		return "", "", err
	}

	var envelope grantEnvelope
	if err := json.Unmarshal(response.Body, &envelope); err != nil {
		return "", "", fmt.Errorf("decode createGrant response: %w", err)
	}
	if strings.TrimSpace(envelope.CreateGrant.CreatedGrant.Key) == "" {
		return "", "", fmt.Errorf("createGrant returned no key")
	}

	return envelope.CreateGrant.CreatedGrant.Key, login.Source, nil
}

func fetchCaptchaToken(ctx context.Context, client *Client) (string, error) {
	response, err := client.Execute(ctx, map[string]any{
		"captchaToken": map[string]any{},
	}, true)
	if err != nil {
		return "", err
	}

	var envelope grantEnvelope
	if err := json.Unmarshal(response.Body, &envelope); err != nil {
		return "", fmt.Errorf("decode captchaToken response: %w", err)
	}
	if strings.TrimSpace(envelope.CaptchaToken) == "" {
		return "", fmt.Errorf("captchaToken response was empty")
	}

	return envelope.CaptchaToken, nil
}

func signCaptchaToken(token string) string {
	parts := strings.Split(token, "\x01")
	if len(parts) < 3 {
		return token
	}

	encoder := encoderFromChars(parts[2])
	decoded := string(encoder.decode(parts[1]))
	unixMillis, ok := parseUnixMillis(decoded)
	if !ok {
		return token
	}
	timestamp := fmt.Sprintf("%d", unixMillis)
	return token + encoder.encode([]byte(timestamp)) + "\x01"
}

func parseUnixMillis(value string) (int64, bool) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err == nil {
		return parsed.UnixMilli(), true
	}

	parsed, err = time.Parse("2006-01-02T15:04:05.000Z07:00", value)
	if err == nil {
		return parsed.UnixMilli(), true
	}

	return 0, false
}

type encoder struct {
	chars []rune
	index map[rune]int
	base  int
}

func encoderFromChars(chars string) encoder {
	runes := []rune(chars)
	index := make(map[rune]int, len(runes))
	for i, char := range runes {
		index[char] = i
	}
	return encoder{chars: runes, index: index, base: len(runes)}
}

func (e encoder) decode(value string) []byte {
	values := make([]int, 0, len(value))
	for _, char := range value {
		values = append(values, e.index[char])
	}
	return convertBase(values, e.base, 256)
}

func (e encoder) encode(value []byte) string {
	values := make([]int, len(value))
	for i, b := range value {
		values[i] = int(b)
	}
	converted := convertBase(values, 256, e.base)
	var builder strings.Builder
	for _, v := range converted {
		builder.WriteRune(e.chars[v])
	}
	return builder.String()
}

func convertBase(values []int, from int, to int) []byte {
	if len(values) == 0 {
		return nil
	}

	ratio := math.Log(float64(from)) / math.Log(float64(to))
	size := len(values)
	if from < to {
		size = maxInt(1, int(float64(len(values))*ratio))
	} else {
		size = maxInt(1, intCeil(float64(len(values))*ratio))
	}

	output := make([]int, size)
	for _, value := range values {
		remainder := value
		for i := len(output) - 1; i >= 0; i-- {
			remainder += from * output[i]
			output[i] = remainder % to
			remainder /= to
		}
	}

	result := make([]byte, len(output))
	for i, value := range output {
		result[i] = byte(value)
	}
	return result
}

func intCeil(value float64) int {
	if float64(int(value)) == value {
		return int(value)
	}
	return int(value) + 1
}

func maxInt(values ...int) int {
	current := values[0]
	for _, value := range values[1:] {
		if value > current {
			current = value
		}
	}
	return current
}
