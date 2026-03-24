package runner

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"raco/util"
)

func PrintResult(result *Result, format string) {
	PrintResultTo(os.Stdout, result, format)
}

func PrintResultTo(w io.Writer, result *Result, format string) {
	result = sanitizeResult(result)
	if format == "json" {
		printJSON(w, result)
		return
	}

	printText(w, result)
}

func printJSON(w io.Writer, result *Result) {
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Fprintln(w, string(data))
}

func printText(w io.Writer, result *Result) {
	fmt.Fprintf(w, "\nCollection: %s\n", result.CollectionName)
	if result.EnvironmentName != "" {
		fmt.Fprintf(w, "Environment: %s\n", result.EnvironmentName)
	}
	fmt.Fprintf(w, "Duration: %dms\n", result.Duration.Milliseconds())
	fmt.Fprintln(w, "---")

	for _, req := range result.RequestResults {
		status := "✓"
		if !req.Passed {
			status = "✗"
		}
		if req.Skipped {
			status = "○"
		}

		fmt.Fprintf(w, "%s %s %s [%d] %dms\n",
			status,
			req.Method,
			req.Name,
			req.StatusCode,
			req.Duration.Milliseconds(),
		)

		if req.SkipReason != "" {
			fmt.Fprintf(w, "  Skip: %s\n", req.SkipReason)
		}
		if req.ErrorMessage != "" {
			fmt.Fprintf(w, "  Error: %s\n", req.ErrorMessage)
		}
		for _, warning := range req.Warnings {
			fmt.Fprintf(w, "  Warning: %s\n", warning)
		}

		for _, assertion := range req.Assertions {
			assertStatus := "  ✓"
			if !assertion.Passed {
				assertStatus = "  ✗"
			}
			fmt.Fprintf(w, "%s [%s] %s\n", assertStatus, assertion.Type, assertion.Message)
		}
	}

	fmt.Fprintln(w, "---")
	fmt.Fprintf(w, "Total: %d | Selected: %d | Passed: %d | Failed: %d | Skipped: %d\n",
		result.TotalCount,
		result.SelectedCount,
		result.PassedCount,
		result.FailedCount,
		result.SkippedCount,
	)
}

func sanitizeResult(result *Result) *Result {
	if result == nil {
		return nil
	}
	out := *result
	if len(result.FailureSummary) > 0 {
		out.FailureSummary = make([]string, len(result.FailureSummary))
		for i, item := range result.FailureSummary {
			out.FailureSummary[i] = util.RedactSensitiveData(item)
		}
	}
	if len(result.RequestResults) > 0 {
		out.RequestResults = make([]RequestResult, len(result.RequestResults))
		for i, req := range result.RequestResults {
			copied := req
			copied.URL = util.RedactSensitiveData(req.URL)
			copied.ErrorMessage = util.RedactSensitiveData(req.ErrorMessage)
			if len(req.Warnings) > 0 {
				copied.Warnings = make([]string, len(req.Warnings))
				for j, warning := range req.Warnings {
					copied.Warnings[j] = util.RedactSensitiveData(warning)
				}
			}
			if len(req.Assertions) > 0 {
				copied.Assertions = make([]AssertionResult, len(req.Assertions))
				for j, assertion := range req.Assertions {
					copied.Assertions[j] = AssertionResult{
						Type:    assertion.Type,
						Passed:  assertion.Passed,
						Message: util.RedactSensitiveData(assertion.Message),
					}
				}
			}
			out.RequestResults[i] = copied
		}
	}
	return &out
}
