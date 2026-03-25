package runner

import (
	"fmt"
	"raco/http"
	"raco/model"
	"raco/util"
	"strings"
	"time"
)

// RequestFilter describes the non-mutating selection rules applied before execution starts.
type RequestFilter struct {
	Refs         []string
	ExactNames   []string
	NameContains []string
	Methods      []string
	Tags         []string
}

// Config holds all execution-time options for a collection run.
type Config struct {
	Collection      *model.Collection
	Environment     *model.ResolvedEnvironment
	StopOnFail      bool
	OutputFormat    string
	FailIfNoTests   bool
	MaxParallel     int
	RequestFilter   RequestFilter
	EnvironmentName string
	SnapshotDir     string
	SnapshotUpdate  bool
	FlakyRetries    int
	Contracts       []string
}

// Result is the serializable run summary used by text, JSON, and JUnit outputs.
type Result struct {
	CollectionName       string          `json:"collection_name"`
	TotalCount           int             `json:"total_count"`
	SelectedCount        int             `json:"selected_count"`
	PassedCount          int             `json:"passed_count"`
	FailedCount          int             `json:"failed_count"`
	SkippedCount         int             `json:"skipped_count"`
	SkippedByFilterCount int             `json:"skipped_by_filter_count"`
	Duration             time.Duration   `json:"duration"`
	StartedAt            time.Time       `json:"started_at"`
	FinishedAt           time.Time       `json:"finished_at"`
	EnvironmentName      string          `json:"environment_name,omitempty"`
	FailureSummary       []string        `json:"failure_summary,omitempty"`
	RequestResults       []RequestResult `json:"request_results"`
}

// RequestResult captures the developer-facing outcome of a single request execution.
type RequestResult struct {
	RequestRef     string            `json:"request_ref"`
	Name           string            `json:"name"`
	Method         string            `json:"method"`
	URL            string            `json:"url"`
	StatusCode     int               `json:"status_code"`
	Duration       time.Duration     `json:"duration"`
	Passed         bool              `json:"passed"`
	Skipped        bool              `json:"skipped"`
	SkipReason     string            `json:"skip_reason,omitempty"`
	Assertions     []AssertionResult `json:"assertions,omitempty"`
	AssertionCount int               `json:"assertion_count"`
	ExtractorCount int               `json:"extractor_count"`
	ExtractedKeys  []string          `json:"extracted_keys,omitempty"`
	ErrorMessage   string            `json:"error_message,omitempty"`
	ErrorCategory  string            `json:"error_category,omitempty"`
	RetriesUsed    int               `json:"retries_used"`
	Warnings       []string          `json:"warnings,omitempty"`
	FlakyRecovered bool              `json:"flaky_recovered,omitempty"`
}

// AssertionResult is a minimal, report-friendly view of assertion evaluation.
type AssertionResult struct {
	Type    string `json:"type"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

// Execute walks the collection in order so extractor output remains deterministic.
func Execute(cfg *Config) *Result {
	startTime := time.Now()

	result := &Result{
		CollectionName:  cfg.Collection.Name,
		TotalCount:      len(cfg.Collection.Requests),
		StartedAt:       startTime,
		EnvironmentName: cfg.EnvironmentName,
		RequestResults:  make([]RequestResult, 0, len(cfg.Collection.Requests)),
	}

	client := http.NewClient()
	plan, planErr := BuildExecutionPlan(cfg.Collection, cfg.RequestFilter)
	if planErr != nil {
		result.FailedCount = 1
		result.FailureSummary = append(result.FailureSummary, planErr.Error())
		result.FinishedAt = time.Now()
		result.Duration = result.FinishedAt.Sub(startTime)
		return result
	}

	for idx, req := range cfg.Collection.Requests {
		if containsIndex(plan.OrderedIndices, idx) {
			continue
		}
		if requestMatches(req, idx, cfg.RequestFilter, cfg.Collection.Tags) {
			continue
		}
		result.RequestResults = append(result.RequestResults, RequestResult{
			RequestRef: fmt.Sprintf("%d", idx),
			Name:       req.Name,
			Method:     req.Method,
			URL:        req.URL,
			Skipped:    true,
			SkipReason: "filtered out",
		})
		result.SkippedCount++
		result.SkippedByFilterCount++
	}

	for _, ref := range cfg.Collection.Hooks.Setup {
		_, req, err := ResolveRequestRef(cfg.Collection, ref)
		if err != nil {
			result.RequestResults = append(result.RequestResults, RequestResult{
				RequestRef: ref,
				Name:       "setup",
				Skipped:    true,
				SkipReason: err.Error(),
			})
			result.FailedCount++
			if cfg.StopOnFail {
				result.FinishedAt = time.Now()
				result.Duration = result.FinishedAt.Sub(startTime)
				return result
			}
		}
		if err == nil {
			hookResult := executeRequest(client, -1, req, cfg.Environment, cfg.FailIfNoTests, cfg)
			hookResult.RequestRef = ref
			hookResult.Name = "[setup] " + req.Name
			result.RequestResults = append(result.RequestResults, hookResult)
			if !hookResult.Passed {
				result.FailedCount++
				result.FailureSummary = append(result.FailureSummary, hookResult.Name)
				if cfg.StopOnFail {
					result.FinishedAt = time.Now()
					result.Duration = result.FinishedAt.Sub(startTime)
					return result
				}
			}
		}
	}

	for _, idx := range plan.OrderedIndices {
		req := cfg.Collection.Requests[idx]
		result.SelectedCount++
		reqResult := executeRequest(client, idx, req, cfg.Environment, cfg.FailIfNoTests, cfg)
		result.RequestResults = append(result.RequestResults, reqResult)

		if reqResult.Passed {
			result.PassedCount++
		}
		if !reqResult.Passed && !reqResult.Skipped {
			result.FailedCount++
			result.FailureSummary = append(result.FailureSummary, fmt.Sprintf("%s %s", req.Method, req.Name))
			if cfg.StopOnFail {
				break
			}
		}
		if reqResult.Skipped {
			result.SkippedCount++
		}
	}

	for _, ref := range cfg.Collection.Hooks.Teardown {
		_, req, err := ResolveRequestRef(cfg.Collection, ref)
		if err != nil {
			result.RequestResults = append(result.RequestResults, RequestResult{
				RequestRef: ref,
				Name:       "teardown",
				Skipped:    true,
				SkipReason: err.Error(),
			})
			result.FailedCount++
			continue
		}
		hookResult := executeRequest(client, -1, req, cfg.Environment, cfg.FailIfNoTests, cfg)
		hookResult.RequestRef = ref
		hookResult.Name = "[teardown] " + req.Name
		result.RequestResults = append(result.RequestResults, hookResult)
		if !hookResult.Passed {
			result.FailedCount++
			result.FailureSummary = append(result.FailureSummary, hookResult.Name)
		}
	}

	result.FinishedAt = time.Now()
	result.Duration = result.FinishedAt.Sub(startTime)
	return result
}

func hasActiveFilters(filter RequestFilter) bool {
	return len(filter.Refs) > 0 || len(filter.ExactNames) > 0 || len(filter.NameContains) > 0 || len(filter.Methods) > 0 || len(filter.Tags) > 0
}

// requestMatches applies all active filters in a single pass to keep large runs cheap.
func requestMatches(req *model.Request, idx int, filter RequestFilter, collectionTags []string) bool {
	if req == nil {
		return false
	}
	if !hasActiveFilters(filter) {
		return true
	}
	if len(filter.Refs) > 0 {
		match := false
		for _, ref := range filter.Refs {
			if ref == fmt.Sprintf("%d", idx) || ref == req.ID || ref == req.Name {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	if len(filter.ExactNames) > 0 {
		match := false
		for _, name := range filter.ExactNames {
			if req.Name == name {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	if len(filter.NameContains) > 0 {
		match := false
		for _, fragment := range filter.NameContains {
			if strings.Contains(strings.ToLower(req.Name), strings.ToLower(fragment)) {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	if len(filter.Methods) > 0 {
		method := strings.ToUpper(req.Method)
		match := false
		for _, allowed := range filter.Methods {
			if method == strings.ToUpper(allowed) {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	if len(filter.Tags) > 0 {
		allTags := append([]string(nil), collectionTags...)
		allTags = append(allTags, req.Tags...)
		match := false
		for _, filterTag := range filter.Tags {
			for _, tag := range allTags {
				if strings.EqualFold(filterTag, tag) {
					match = true
					break
				}
			}
			if match {
				break
			}
		}
		if !match {
			return false
		}
	}
	return true
}

// executeRequest resolves variables, executes the request, evaluates assertions,
// and pushes extractor output back into the shared runtime environment.
func executeRequest(client *http.Client, idx int, req *model.Request, env *model.ResolvedEnvironment, failIfNoTests bool, cfg *Config) RequestResult {
	secrets := secretValues(env)
	result := RequestResult{
		RequestRef:     fmt.Sprintf("%d", idx),
		Name:           req.Name,
		Method:         req.Method,
		URL:            req.URL,
		Assertions:     make([]AssertionResult, 0, len(req.Assertions)),
		AssertionCount: len(req.Assertions),
		ExtractorCount: len(req.Extractors),
	}

	processedReq := http.ResolveRequest(req, env)
	if processedReq != nil {
		result.URL = util.RedactWithSecrets(processedReq.URL, secrets)
	}

	attempts := 1
	if cfg != nil && cfg.FlakyRetries > 0 {
		attempts += cfg.FlakyRetries
	}

	for attempt := 0; attempt < attempts; attempt++ {
		resp, err := client.Execute(processedReq)
		if err != nil {
			result.ErrorMessage = util.RedactWithSecrets(err.Error(), secrets)
			result.ErrorCategory = "transport"
			result.Passed = false
			result.RetriesUsed = attempt
			continue
		}

		result.StatusCode = resp.StatusCode
		result.Duration = resp.Duration
		result.Passed = true
		result.Assertions = result.Assertions[:0]

		combinedAssertions := append([]model.Assertion(nil), req.Assertions...)
		for _, contractName := range cfg.Contracts {
			for _, contract := range cfg.Collection.Contracts {
				if contract.Name == contractName {
					combinedAssertions = append(combinedAssertions, contract.Assertions...)
				}
			}
		}

		if len(combinedAssertions) == 0 {
			result.Warnings = append(result.Warnings, "request has no assertions")
			if failIfNoTests {
				result.Passed = false
				result.ErrorCategory = "no_assertions"
				result.ErrorMessage = "request has no assertions"
			}
		}

		for _, assertion := range combinedAssertions {
			assertResult := model.ValidateAssertion(assertion, resp)
			result.Assertions = append(result.Assertions, AssertionResult{
				Type:    string(assertion.Type),
				Passed:  assertResult.Passed,
				Message: util.RedactWithSecrets(assertResult.Message, secrets),
			})
			if !assertResult.Passed {
				result.Passed = false
				result.ErrorCategory = "assertion"
				result.ErrorMessage = assertResult.Message
			}
		}

		if snapshotEnabled(req, cfg) {
			snapshotResult := applySnapshot(cfg.Collection.ID, req, resp, cfg)
			result.Assertions = append(result.Assertions, snapshotResult)
			if !snapshotResult.Passed {
				result.Passed = false
				result.ErrorCategory = "snapshot"
				result.ErrorMessage = snapshotResult.Message
			}
		}

		if env != nil {
			wrappedEnv := &model.Environment{Variables: toEnvironmentVariables(env.Variables)}
			for _, extractor := range req.Extractors {
				if err := model.ExtractValue(extractor, resp, wrappedEnv); err == nil {
					result.ExtractedKeys = append(result.ExtractedKeys, extractor.Target)
				}
			}
			for key, variable := range wrappedEnv.Variables {
				env.Variables[key] = variable.Value
			}
		}

		result.RetriesUsed = attempt
		if attempt > 0 && result.Passed {
			result.FlakyRecovered = true
			result.Warnings = append(result.Warnings, fmt.Sprintf("request recovered after %d retry", attempt))
		}
		if result.Passed {
			break
		}
	}

	return result
}

func containsIndex(items []int, target int) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

// toEnvironmentVariables adapts resolved runtime values back into the model shape
// required by the extractor helpers.
func toEnvironmentVariables(values map[string]string) map[string]model.EnvironmentVariable {
	out := make(map[string]model.EnvironmentVariable, len(values))
	for key, value := range values {
		out[key] = model.EnvironmentVariable{Kind: model.EnvironmentVariablePlain, Value: value}
	}
	return out
}

func secretValues(env *model.ResolvedEnvironment) []string {
	if env == nil {
		return nil
	}
	return env.SecretValues()
}
