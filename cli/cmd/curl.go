package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"raco/util"
	"strings"
)

func RunCurl(ctx *Context, args []string) int {
	if len(args) < 2 {
		printCurlUsage()
		return 1
	}

	action := args[0]
	switch action {
	case "parse":
		return curlParse(strings.Join(args[1:], " "))
	case "convert":
		return curlConvert(ctx, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown action: %s\n", action)
		printCurlUsage()
		return 1
	}
}

func printCurlUsage() {
	fmt.Println(`Usage: raco curl <action> [options]

Actions:
  parse <curl-command>           Parse cURL command to request
  convert <collection-id> <req>  Convert saved request to cURL

Examples:
  raco curl parse 'curl -X POST https://api.example.org -H "Content-Type: application/json" -d "{\"key\":\"value\"}"'
  raco curl convert my-collection 0`)
}

func curlParse(curlCmd string) int {
	req, err := util.ParseCurl(curlCmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse failed: %v\n", err)
		return 1
	}

	data, _ := json.MarshalIndent(req, "", "  ")
	fmt.Println(string(data))
	return 0
}

func curlConvert(ctx *Context, args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: raco curl convert <collection-id> <request-index>")
		return 1
	}

	colID := args[0]
	reqIdx := 0
	fmt.Sscanf(args[1], "%d", &reqIdx)

	store := ctx.Storage()
	col, err := store.LoadCollection(colID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if reqIdx < 0 || reqIdx >= len(col.Requests) {
		fmt.Fprintf(os.Stderr, "Invalid request index: %d\n", reqIdx)
		return 1
	}

	curlCmd := util.ToCurl(col.Requests[reqIdx])
	fmt.Println(curlCmd)
	return 0
}
