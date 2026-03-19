package pave

import (
	"strings"
	"testing"
	"time"
)

func TestSignCaptchaToken(t *testing.T) {
	chars := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	encoder := encoderFromChars(chars)
	issuedAt := "2026-03-06T00:00:00.000Z"
	token := "prefix\x01" + encoder.encode([]byte(issuedAt)) + "\x01" + chars

	got := signCaptchaToken(token)
	if !strings.HasPrefix(got, token) {
		t.Fatalf("expected signed token to keep original prefix")
	}

	parsed, err := time.Parse("2006-01-02T15:04:05.000Z07:00", issuedAt)
	if err != nil {
		t.Fatalf("parse test timestamp: %v", err)
	}

	expectedSuffix := encoder.encode([]byte("1772755200000")) + "\x01"
	if !strings.HasSuffix(got, expectedSuffix) {
		t.Fatalf("expected suffix %q, got %q", expectedSuffix, got[len(token):])
	}

	if parsed.UnixMilli() != 1772755200000 {
		t.Fatalf("unexpected unix millis %d", parsed.UnixMilli())
	}
}
