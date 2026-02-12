package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"raco/storage"
	"strings"
)

func RunImport(ctx *Context, args []string) int {
	if len(args) < 2 {
		printImportUsage()
		return 1
	}

	format := args[0]
	filePath := args[1]

	if strings.Contains(filePath, "..") {
		fmt.Fprintln(os.Stderr, "Error: invalid file path")
		return 1
	}

	absPath, err := filepath.Abs(filepath.Clean(filePath))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid path: %v\n", err)
		return 1
	}

	switch format {
	case "postman":
		return importPostman(ctx, absPath)
	default:
		fmt.Fprintf(os.Stderr, "Unknown format: %s\n", format)
		printImportUsage()
		return 1
	}
}

func printImportUsage() {
	fmt.Println(`Usage: raco import <format> <file>

Formats:
  postman    Import Postman collection (JSON)

Examples:
  raco import postman my-collection.json
  raco import postman ~/Downloads/api-tests.postman_collection.json`)
}

func importPostman(ctx *Context, filePath string) int {
	collection, err := storage.ImportPostmanCollection(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Import failed: %v\n", err)
		return 1
	}

	store := ctx.Storage()
	if err := store.SaveCollection(collection); err != nil {
		fmt.Fprintf(os.Stderr, "Save failed: %v\n", err)
		return 1
	}

	fmt.Printf("Imported collection: %s (%d requests)\n", collection.Name, len(collection.Requests))
	return 0
}
