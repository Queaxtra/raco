package output

import (
	"encoding/json"
	"fmt"
	"raco/model"
)

func PrintResponse(resp *model.Response, format string) int {
	if format == "json" {
		return printJSON(resp)
	}

	if format == "full" {
		return printFull(resp)
	}

	return printBody(resp)
}

func printJSON(resp *model.Response) int {
	result := map[string]interface{}{
		"status_code": resp.StatusCode,
		"headers":     resp.Headers,
		"body":        resp.Body,
		"duration_ms": resp.Duration.Milliseconds(),
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
	return 0
}

func printFull(resp *model.Response) int {
	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Duration: %dms\n", resp.Duration.Milliseconds())
	fmt.Println("Headers:")
	for k, v := range resp.Headers {
		fmt.Printf("  %s: %s\n", k, v)
	}
	fmt.Println("Body:")
	fmt.Println(resp.Body)
	return 0
}

func printBody(resp *model.Response) int {
	fmt.Print(resp.Body)
	return 0
}
