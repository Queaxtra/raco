package environment

import (
	"os"
	"path/filepath"
	"raco/model"
	"raco/secretstore"
	"testing"
)

func TestLoadLegacyEnvironment(t *testing.T) {
	baseDir := t.TempDir()
	envDir := filepath.Join(baseDir, "environments")
	if err := os.MkdirAll(envDir, 0750); err != nil {
		t.Fatal(err)
	}
	data := []byte("name: legacy\nvariables:\n  API_URL: https://example.com\n")
	if err := os.WriteFile(filepath.Join(envDir, "legacy.yaml"), data, 0600); err != nil {
		t.Fatal(err)
	}

	env, err := Load(baseDir, "legacy")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if env.GetVariable("API_URL") != "https://example.com" {
		t.Fatalf("unexpected API_URL: %s", env.GetVariable("API_URL"))
	}
}

func TestResolveEnvironmentWithSecret(t *testing.T) {
	baseDir := t.TempDir()
	env := &model.Environment{
		Name: "prod",
		Variables: map[string]model.EnvironmentVariable{
			"API_URL":   {Kind: model.EnvironmentVariablePlain, Value: "https://example.com"},
			"API_TOKEN": {Kind: model.EnvironmentVariableSecret, Ref: "raco/prod/API_TOKEN"},
		},
	}
	if err := Save(baseDir, env); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	fake := secretstore.NewFakeStore()
	if err := fake.Set("prod", "API_TOKEN", "secret-value"); err != nil {
		t.Fatal(err)
	}

	resolved, err := Resolve(baseDir, "prod", fake)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.GetVariable("API_TOKEN") != "secret-value" {
		t.Fatalf("unexpected secret value: %s", resolved.GetVariable("API_TOKEN"))
	}
	if !resolved.IsSecretKey("API_TOKEN") {
		t.Fatalf("expected API_TOKEN to be tracked as secret")
	}
}
