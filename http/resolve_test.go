package http

import (
	"raco/model"
	"strconv"
	"testing"
)

func TestPreviewRequestMasksSecrets(t *testing.T) {
	req := &model.Request{
		Method: "GET",
		URL:    "https://example.com?token={{API_TOKEN}}",
		Headers: map[string]string{
			"Authorization": "Bearer {{API_TOKEN}}",
		},
	}
	env := &model.ResolvedEnvironment{
		Variables: map[string]string{
			"API_TOKEN": "super-secret",
		},
		SecretKeys:  map[string]struct{}{"API_TOKEN": {}},
		SecretPlain: []string{"super-secret"},
	}

	preview := PreviewRequest(req, env, env)
	if preview == nil {
		t.Fatal("expected preview")
	}
	if preview.URL == req.URL {
		t.Fatalf("expected resolved URL to differ from template")
	}
	if preview.Headers["Authorization"] == "Bearer super-secret" {
		t.Fatalf("expected secret header to be masked")
	}
	if !preview.Masked {
		t.Fatalf("expected preview to indicate masking")
	}
}

func BenchmarkPreviewRequest(b *testing.B) {
	req := &model.Request{
		Method: "POST",
		URL:    "https://example.com/{{TENANT}}/users",
		Headers: map[string]string{
			"Authorization": "Bearer {{TOKEN}}",
		},
		Body: `{"tenant":"{{TENANT}}","token":"{{TOKEN}}"}`,
	}
	env := &model.ResolvedEnvironment{
		Variables: map[string]string{
			"TENANT": "acme",
			"TOKEN":  "super-secret-token",
		},
		SecretKeys:  map[string]struct{}{"TOKEN": {}},
		SecretPlain: []string{"super-secret-token"},
	}
	for i := 0; i < b.N; i++ {
		req.Headers["X-Req"] = strconv.Itoa(i)
		_ = PreviewRequest(req, env, env)
	}
}
