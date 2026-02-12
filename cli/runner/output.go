package runner

import (
	"encoding/json"
	"fmt"
)

func PrintResult(result *Result, format string) {
	if format == "json" {
		printJSON(result)
		return
	}

	printText(result)
}

func printJSON(result *Result) {
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
}

func printText(result *Result) {
	fmt.Printf("\nCollection: %s\n", result.CollectionName)
	fmt.Printf("Duration: %dms\n", result.Duration.Milliseconds())
	fmt.Println("---")

	for _, req := range result.RequestResults {
		status := "✓"
		if !req.Passed {
			status = "✗"
		}
		if req.Skipped {
			status = "○"
		}

		fmt.Printf("%s %s %s [%d] %dms\n",
			status,
			req.Method,
			req.Name,
			req.StatusCode,
			req.Duration.Milliseconds(),
		)

		if req.ErrorMessage != "" {
			fmt.Printf("  Error: %s\n", req.ErrorMessage)
		}

		for _, assertion := range req.Assertions {
			assertStatus := "  ✓"
			if !assertion.Passed {
				assertStatus = "  ✗"
			}
			fmt.Printf("%s [%s] %s\n", assertStatus, assertion.Type, assertion.Message)
		}
	}

	fmt.Println("---")
	fmt.Printf("Total: %d | Passed: %d | Failed: %d | Skipped: %d\n",
		result.TotalCount,
		result.PassedCount,
		result.FailedCount,
		result.SkippedCount,
	)
}
