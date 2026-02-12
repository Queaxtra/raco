package runner

import (
	"raco/http"
	"raco/model"
	"time"
)

type EnvironmentProvider interface {
	GetVariable(key string) string
	GetVariables() map[string]string
}

type Config struct {
	Collection   *model.Collection
	Environment  EnvironmentProvider
	StopOnFail   bool
	OutputFormat string
}

type Result struct {
	CollectionName string
	TotalCount     int
	PassedCount    int
	FailedCount    int
	SkippedCount   int
	Duration       time.Duration
	RequestResults []RequestResult
}

type RequestResult struct {
	Name         string
	Method       string
	URL          string
	StatusCode   int
	Duration     time.Duration
	Passed       bool
	Skipped      bool
	Assertions   []AssertionResult
	ErrorMessage string
}

type AssertionResult struct {
	Type    string
	Passed  bool
	Message string
}

func Execute(cfg *Config) *Result {
	startTime := time.Now()

	result := &Result{
		CollectionName: cfg.Collection.Name,
		TotalCount:     len(cfg.Collection.Requests),
		RequestResults: make([]RequestResult, 0, len(cfg.Collection.Requests)),
	}

	var env *model.Environment
	if cfg.Environment != nil {
		env = &model.Environment{
			Variables: cfg.Environment.GetVariables(),
		}
	}

	client := http.NewClient()

	for _, req := range cfg.Collection.Requests {
		reqResult := executeRequest(client, req, env)
		result.RequestResults = append(result.RequestResults, reqResult)

		if reqResult.Passed {
			result.PassedCount++
		}
		if !reqResult.Passed && !reqResult.Skipped {
			result.FailedCount++
			if cfg.StopOnFail {
				result.SkippedCount = result.TotalCount - len(result.RequestResults)
				break
			}
		}
	}

	result.Duration = time.Since(startTime)
	return result
}

func executeRequest(client *http.Client, req *model.Request, env *model.Environment) RequestResult {
	result := RequestResult{
		Name:   req.Name,
		Method: req.Method,
		URL:    req.URL,
	}

	processedReq := &model.Request{
		ID:         req.ID,
		Name:       req.Name,
		Method:     req.Method,
		URL:        http.ReplaceEnvVars(req.URL, env),
		Headers:    make(map[string]string),
		Body:       http.ReplaceEnvVars(req.Body, env),
		Assertions: req.Assertions,
		Extractors: req.Extractors,
	}

	for k, v := range req.Headers {
		processedReq.Headers[k] = http.ReplaceEnvVars(v, env)
	}

	resp, err := client.Execute(processedReq)
	if err != nil {
		result.ErrorMessage = err.Error()
		result.Passed = false
		return result
	}

	result.StatusCode = resp.StatusCode
	result.Duration = resp.Duration
	result.Passed = true

	for _, assertion := range req.Assertions {
		assertResult := model.ValidateAssertion(assertion, resp)
		result.Assertions = append(result.Assertions, AssertionResult{
			Type:    string(assertion.Type),
			Passed:  assertResult.Passed,
			Message: assertResult.Message,
		})
		if !assertResult.Passed {
			result.Passed = false
		}
	}

	if env != nil {
		for _, extractor := range req.Extractors {
			model.ExtractValue(extractor, resp, env)
		}
	}

	return result
}
