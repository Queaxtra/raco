package cmd

import (
	"bytes"
	"io"
	"os"
	"raco/model"
	"raco/storage"
	"strings"
	"testing"
)

func TestRunRequestDryRun(t *testing.T) {
	home := t.TempDir()
	store := storage.NewStorage(home)
	if err := store.SaveEnvironment(&model.Environment{
		Name: "prod",
		Variables: map[string]model.EnvironmentVariable{
			"API_URL": {Kind: model.EnvironmentVariablePlain, Value: "https://example.com"},
		},
	}); err != nil {
		t.Fatal(err)
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	code := RunRequest(&Context{StoragePath: home}, []string{"-m", "GET", "-r", "{{API_URL}}/users", "-e", "prod", "--dry-run"})
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if code != 0 {
		t.Fatalf("expected success, got exit code %d", code)
	}
	if !strings.Contains(buf.String(), "https://example.com/users") {
		t.Fatalf("unexpected preview output: %s", buf.String())
	}
}

func TestResolveEnvironmentFailsClosedForSecrets(t *testing.T) {
	home := t.TempDir()
	store := storage.NewStorage(home)
	if err := store.SaveEnvironment(&model.Environment{
		Name: "prod",
		Variables: map[string]model.EnvironmentVariable{
			"API_TOKEN": {Kind: model.EnvironmentVariableSecret, Ref: "raco/prod/API_TOKEN"},
		},
	}); err != nil {
		t.Fatal(err)
	}

	t.Setenv("RACO_SECRET_BACKEND", "unsupported")
	_, err := (&Context{StoragePath: home}).ResolveEnvironment("prod")
	if err == nil {
		t.Fatal("expected secret-backed environment resolution to fail closed")
	}
}
