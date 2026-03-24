package util

import (
	"strings"
	"testing"
)

func TestRedactWithSecrets(t *testing.T) {
	input := "Authorization: Bearer super-secret-token and mail user@example.com"
	got := RedactWithSecrets(input, []string{"super-secret-token"})
	if got == input {
		t.Fatalf("expected redaction to change output")
	}
	if strings.Contains(got, "super-secret-token") {
		t.Fatalf("expected raw secret to be removed, got %s", got)
	}
}

func TestRedactHeadersWithSecrets(t *testing.T) {
	headers := map[string]string{
		"Authorization": "Bearer top-secret",
		"X-Trace":       "plain",
	}
	got := RedactHeadersWithSecrets(headers, []string{"top-secret"})
	if got["Authorization"] == headers["Authorization"] {
		t.Fatalf("expected authorization header to be redacted")
	}
	if got["X-Trace"] != "plain" {
		t.Fatalf("expected non-secret header to remain unchanged")
	}
}

func BenchmarkRedactWithSecrets(b *testing.B) {
	payload := strings.Repeat("token=super-secret-value;", 128)
	secrets := []string{"super-secret-value", "other-secret", "third-secret"}
	for i := 0; i < b.N; i++ {
		_ = RedactWithSecrets(payload, secrets)
	}
}
