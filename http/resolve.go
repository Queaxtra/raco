package http

import (
	"raco/model"
	"raco/util"
	"sort"
)

// VariableProvider exposes resolved variables without coupling request resolution
// to a specific storage or runtime implementation.
type VariableProvider interface {
	GetVariables() map[string]string
}

// SecretProvider exposes raw secret values strictly for redaction steps.
type SecretProvider interface {
	SecretValues() []string
}

// ResolveRequest centralizes interpolation so preview, runner, and CLI execution
// all resolve requests with identical rules.
func ResolveRequest(req *model.Request, env VariableProvider) *model.Request {
	if req == nil {
		return nil
	}

	resolved := &model.Request{
		ID:             req.ID,
		Name:           req.Name,
		Method:         req.Method,
		URL:            ReplaceEnvVars(req.URL, env),
		Query:          ReplaceEnvVarsInMap(req.Query, env),
		Headers:        make(map[string]string, len(req.Headers)),
		Body:           ReplaceEnvVars(req.Body, env),
		Files:          append([]model.FileUpload(nil), req.Files...),
		TimeoutSeconds: req.TimeoutSeconds,
		CreatedAt:      req.CreatedAt,
		CollectionID:   req.CollectionID,
		Assertions:     append([]model.Assertion(nil), req.Assertions...),
		Extractors:     append([]model.Extractor(nil), req.Extractors...),
	}
	for key, value := range req.Headers {
		resolved.Headers[key] = ReplaceEnvVars(value, env)
	}
	return resolved
}

// PreviewRequest creates a display-safe view of a request after interpolation.
// It resolves first and redacts second so the preview matches the real runtime shape.
func PreviewRequest(req *model.Request, env VariableProvider, secrets SecretProvider) *model.PreviewResult {
	resolved := ResolveRequest(req, env)
	if resolved == nil {
		return nil
	}

	secretValues := []string(nil)
	if secrets != nil {
		secretValues = secrets.SecretValues()
	}

	headers := util.RedactHeadersWithSecrets(resolved.Headers, secretValues)
	body := util.RedactWithSecrets(resolved.Body, secretValues)
	url := util.RedactWithSecrets(resolved.URL, secretValues)
	query := make(map[string]string, len(resolved.Query))
	for key, value := range resolved.Query {
		query[key] = util.RedactWithSecrets(value, secretValues)
	}

	out := &model.PreviewResult{
		Method:       resolved.Method,
		URL:          url,
		Query:        query,
		Headers:      headers,
		Body:         body,
		Files:        append([]model.FileUpload(nil), resolved.Files...),
		RequestName:  resolved.Name,
		CollectionID: resolved.CollectionID,
		Masked:       len(secretValues) > 0,
	}

	if env != nil {
		keys := env.GetVariables()
		out.SecretKeys = collectSecretKeys(keys, secrets)
	}

	if out.Masked {
		out.Warnings = append(out.Warnings, "secret values were redacted in preview output")
	}

	return out
}

// collectSecretKeys maps resolved secret values back to keys so previews can explain
// which environment entries were masked.
func collectSecretKeys(values map[string]string, secrets SecretProvider) []string {
	if secrets == nil || len(values) == 0 {
		return nil
	}
	secretSet := make(map[string]struct{})
	for _, value := range secrets.SecretValues() {
		if value == "" {
			continue
		}
		for key, candidate := range values {
			if candidate == value {
				secretSet[key] = struct{}{}
			}
		}
	}
	if len(secretSet) == 0 {
		return nil
	}
	out := make([]string, 0, len(secretSet))
	for key := range secretSet {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
