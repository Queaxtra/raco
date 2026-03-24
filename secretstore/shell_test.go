package secretstore

import "testing"

func TestCommandStoreEnsureAvailable(t *testing.T) {
	store := &commandStore{}
	if err := store.ensureAvailable("definitely-not-a-real-binary"); err != ErrUnsupported {
		t.Fatalf("expected ErrUnsupported, got %v", err)
	}
}

func TestParseDarwinList(t *testing.T) {
	output := `
    "svce"<blob>="raco/prod"
    "acct"<blob>="API_TOKEN"
    "svce"<blob>="raco/prod"
    "acct"<blob>="API_KEY"
`
	keys := parseDarwinList(output, "raco/prod")
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
}

func TestParseLinuxList(t *testing.T) {
	output := `
service = "raco/prod"
account = "API_TOKEN"

service = "raco/prod"
account = "API_KEY"
`
	keys := parseLinuxList(output, "raco/prod")
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
}
