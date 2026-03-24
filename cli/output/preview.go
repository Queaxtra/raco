package output

import (
	"encoding/json"
	"fmt"
	"raco/model"
	"sort"
)

// PrintPreview renders a redacted dry-run payload in text or JSON format.
func PrintPreview(preview *model.PreviewResult, format string) int {
	if preview == nil {
		return 1
	}
	if format == "json" {
		data, _ := json.MarshalIndent(preview, "", "  ")
		fmt.Println(string(data))
		return 0
	}

	fmt.Printf("Method: %s\n", preview.Method)
	fmt.Printf("URL: %s\n", preview.URL)
	if len(preview.Query) > 0 {
		fmt.Println("Query:")
		keys := make([]string, 0, len(preview.Query))
		for key := range preview.Query {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Printf("  %s=%s\n", key, preview.Query[key])
		}
	}
	if len(preview.Headers) > 0 {
		fmt.Println("Headers:")
		keys := make([]string, 0, len(preview.Headers))
		for key := range preview.Headers {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Printf("  %s: %s\n", key, preview.Headers[key])
		}
	}
	if preview.Body != "" {
		fmt.Println("Body:")
		fmt.Println(preview.Body)
	}
	if len(preview.Files) > 0 {
		fmt.Println("Files:")
		for _, file := range preview.Files {
			fmt.Printf("  %s=%s\n", file.FieldName, file.FilePath)
		}
	}
	for _, warning := range preview.Warnings {
		fmt.Printf("Warning: %s\n", warning)
	}
	return 0
}
