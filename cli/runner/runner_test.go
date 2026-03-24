package runner

import (
	"bytes"
	"os"
	"path/filepath"
	"raco/model"
	"strings"
	"testing"
)

func TestRequestMatches(t *testing.T) {
	req := &model.Request{ID: "req-1", Name: "Get Users", Method: "GET"}
	filter := RequestFilter{
		Refs:         []string{"req-1"},
		NameContains: []string{"users"},
		Methods:      []string{"GET"},
	}
	if !requestMatches(req, 0, filter) {
		t.Fatalf("expected request to match filter")
	}
}

func TestWriteReportJSON(t *testing.T) {
	cwd := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldwd) }()
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	path := "report.json"
	result := &Result{CollectionName: "demo", SelectedCount: 1}
	if err := WriteReport(result, path, "json"); err != nil {
		t.Fatalf("WriteReport() error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(cwd, path))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), "\"collection_name\": \"demo\"") {
		t.Fatalf("unexpected report contents: %s", string(data))
	}
}

func TestWriteReportRejectsTraversal(t *testing.T) {
	cwd := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldwd) }()
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}

	err = WriteReport(&Result{CollectionName: "demo"}, "../escape.json", "json")
	if err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
}

func TestWriteReportRejectsSymlinkEscape(t *testing.T) {
	cwd := t.TempDir()
	outside := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldwd) }()
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(cwd, "link")); err != nil {
		t.Fatal(err)
	}

	err = WriteReport(&Result{CollectionName: "demo"}, "link/report.json", "json")
	if err == nil {
		t.Fatal("expected symlink escape to be rejected")
	}
}

func TestSecretValuesNil(t *testing.T) {
	if values := secretValues(nil); values != nil {
		t.Fatalf("expected nil secret list, got %v", values)
	}
}

func TestPrintResultRedactsSensitiveData(t *testing.T) {
	var buf bytes.Buffer
	PrintResultTo(&buf, &Result{
		CollectionName: "demo",
		RequestResults: []RequestResult{
			{
				Name:         "Get Users",
				Method:       "GET",
				URL:          "https://example.com?token=super-secret-token",
				ErrorMessage: "Bearer super-secret-token",
				Assertions: []AssertionResult{
					{Type: "header", Passed: false, Message: "Authorization Bearer super-secret-token"},
				},
			},
		},
	}, "text")
	output := buf.String()
	if strings.Contains(output, "super-secret-token") {
		t.Fatalf("expected sensitive data to be redacted, got %s", output)
	}
}

func BenchmarkRequestMatches(b *testing.B) {
	req := &model.Request{ID: "req-100", Name: "Get Users", Method: "GET"}
	filter := RequestFilter{
		Refs:         []string{"req-100"},
		ExactNames:   []string{"Get Users"},
		NameContains: []string{"users"},
		Methods:      []string{"GET"},
	}
	for i := 0; i < b.N; i++ {
		if !requestMatches(req, 100, filter) {
			b.Fatal("expected request to match filter")
		}
	}
}

func BenchmarkWriteReportJSON(b *testing.B) {
	cwd := b.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldwd) }()
	if err := os.Chdir(cwd); err != nil {
		b.Fatal(err)
	}

	result := &Result{
		CollectionName: "bench",
		SelectedCount:  100,
		RequestResults: make([]RequestResult, 100),
	}
	for i := 0; i < b.N; i++ {
		if err := WriteReport(result, "bench.json", "json"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteReportJUnit(b *testing.B) {
	cwd := b.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldwd) }()
	if err := os.Chdir(cwd); err != nil {
		b.Fatal(err)
	}

	result := &Result{
		CollectionName: "bench",
		SelectedCount:  100,
		RequestResults: make([]RequestResult, 100),
	}
	for i := 0; i < b.N; i++ {
		if err := WriteReport(result, "bench.xml", "junit"); err != nil {
			b.Fatal(err)
		}
	}
}
