package cmd

import (
	"fmt"
	"os"
	"raco/storage"
)

func RunExport(ctx *Context, args []string) int {
	if len(args) < 4 {
		printExportUsage()
		return 1
	}

	format := args[0]
	collectionID := args[1]
	flagName := args[2]
	outputPath := args[3]

	if flagName != "-o" {
		fmt.Fprintln(os.Stderr, "Usage: raco export openapi|har <collection-id> -o <file>")
		return 1
	}

	col, err := ctx.Storage().LoadCollection(collectionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if format == "openapi" {
		if err := storage.ExportOpenAPICollection(col, outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "Export failed: %v\n", err)
			return 1
		}
		fmt.Printf("Exported %s to %s\n", collectionID, outputPath)
		return 0
	}

	if format == "har" {
		if err := storage.ExportHARCollection(col, outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "Export failed: %v\n", err)
			return 1
		}
		fmt.Printf("Exported %s to %s\n", collectionID, outputPath)
		return 0
	}

	fmt.Fprintf(os.Stderr, "Unknown format: %s\n", format)
	printExportUsage()
	return 1
}

func printExportUsage() {
	fmt.Println(`Usage: raco export <format> <collection-id> -o <file>

Formats:
  openapi   Export collection as OpenAPI 3.1
  har       Export collection as HAR 1.2`)
}
