package cmd

import (
	"raco/model"
	"testing"
)

// TestResolveRequestRef covers the supported request reference forms so CLI helpers
// do not diverge when collections are addressed by users or automation.
func TestResolveRequestRef(t *testing.T) {
	col := &model.Collection{
		Requests: []*model.Request{
			{ID: "req-1", Name: "Get Users"},
			{ID: "req-2", Name: "Create User"},
		},
	}

	idx, req, err := resolveRequestRef(col, "1")
	if err != nil || idx != 1 || req.Name != "Create User" {
		t.Fatalf("index resolve failed: idx=%d req=%v err=%v", idx, req, err)
	}

	idx, req, err = resolveRequestRef(col, "req-1")
	if err != nil || idx != 0 || req.Name != "Get Users" {
		t.Fatalf("id resolve failed: idx=%d req=%v err=%v", idx, req, err)
	}
}

// TestValidateExtractorConfig verifies that extractor validation accepts the
// intended DX-friendly happy path for regex-based chaining.
func TestValidateExtractorConfig(t *testing.T) {
	err := validateExtractorConfig(model.Extractor{
		Type:    model.ExtractRegex,
		Target:  "USER_ID",
		Pattern: `"id":"([^"]+)"`,
	})
	if err != nil {
		t.Fatalf("expected regex extractor to be valid: %v", err)
	}
}

// TestResolveRequestRefAmbiguous ensures exact-name lookups fail when multiple
// saved requests share the same name.
func TestResolveRequestRefAmbiguous(t *testing.T) {
	col := &model.Collection{
		Requests: []*model.Request{
			{Name: "Users"},
			{Name: "Users"},
		},
	}

	_, _, err := resolveRequestRef(col, "Users")
	if err == nil {
		t.Fatal("expected ambiguous request reference to fail")
	}
}
