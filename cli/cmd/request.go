package cmd

import (
	"flag"
	"fmt"
	"os"
	"raco/cli/output"
	"raco/http"
	"raco/model"
	"raco/util/osnotify"
	"strings"
)

type requestConfig struct {
	Method         string
	URL            string
	Query          map[string]string
	Headers        map[string]string
	Body           string
	Files          []model.FileUpload
	TimeoutSeconds int
	Output         string
	Environment    string
	DryRun         bool
}

// RunRequest uses the same resolution path for dry-run previews and real execution.
// That keeps CLI behavior aligned with the runner and the TUI.
func RunRequest(ctx *Context, args []string) int {
	cfg, err := parseRequestArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		if err.Error() == "URL is required (-r)" {
			printRequestUsage()
		}
		return 1
	}

	req := &model.Request{
		Method:         strings.ToUpper(cfg.Method),
		URL:            cfg.URL,
		Query:          cfg.Query,
		Headers:        cfg.Headers,
		Body:           cfg.Body,
		Files:          cfg.Files,
		TimeoutSeconds: cfg.TimeoutSeconds,
	}

	var resolvedEnv *model.ResolvedEnvironment
	if cfg.Environment != "" {
		resolvedEnv, err = ctx.ResolveEnvironment(cfg.Environment)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
	}

	if cfg.DryRun {
		return output.PrintPreview(http.PreviewRequest(req, resolvedEnv, resolvedEnv), previewOutputFormat(cfg.Output))
	}

	if resolvedEnv != nil {
		req = http.ResolveRequest(req, resolvedEnv)
	}

	client := http.NewClient()
	resp, err := client.Execute(req)
	if err != nil {
		osnotify.Send("Raco", "Request failed: "+err.Error())
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	osnotify.Send("Raco", fmt.Sprintf("Request completed: %d", resp.StatusCode))
	return output.PrintResponse(resp, cfg.Output)
}

func parseRequestArgs(args []string) (*requestConfig, error) {
	fs := flag.NewFlagSet("request", flag.ContinueOnError)

	method := fs.String("m", "GET", "HTTP method")
	url := fs.String("r", "", "Request URL")
	body := fs.String("d", "", "Request body")
	headers := fs.String("H", "", "Headers (format: Key:Value, multiple separated by ;)")
	query := fs.String("q", "", "Query params (format: key=value, multiple separated by ;)")
	timeout := fs.Int("t", 0, "Request timeout in seconds (0 = default 30)")
	file := fs.String("f", "", "File upload (format: field_name:file_path)")
	outputFmt := fs.String("o", "body", "Output format: body, json, full")
	env := fs.String("e", "", "Environment name")
	dryRun := fs.Bool("dry-run", false, "Resolve request without sending it")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if *url == "" {
		return nil, fmt.Errorf("URL is required (-r)")
	}

	cfg := &requestConfig{
		Method:         *method,
		URL:            *url,
		Body:           *body,
		Files:          make([]model.FileUpload, 0),
		Headers:        make(map[string]string),
		Query:          make(map[string]string),
		TimeoutSeconds: *timeout,
		Output:         *outputFmt,
		Environment:    *env,
		DryRun:         *dryRun,
	}

	if *query != "" {
		pairs := strings.Split(*query, ";")
		for _, pair := range pairs {
			parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
			if len(parts) == 2 {
				cfg.Query[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	if *headers != "" {
		pairs := strings.Split(*headers, ";")
		for _, pair := range pairs {
			parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
			if len(parts) == 2 {
				cfg.Headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	if *file != "" {
		fileParts := strings.SplitN(*file, ":", 2)
		if len(fileParts) == 2 {
			fileUpload := model.FileUpload{
				FieldName: strings.TrimSpace(fileParts[0]),
				FilePath:  strings.TrimSpace(fileParts[1]),
			}
			if err := fileUpload.Validate(); err != nil {
				return nil, fmt.Errorf("invalid file: %w", err)
			}
			cfg.Files = append(cfg.Files, fileUpload)
		}
	}

	return cfg, nil
}

func ParseRequestArgsPublic(args []string) (method, url, body string, headers, query map[string]string, timeoutSeconds int, err error) {
	cfg, err := parseRequestArgs(args)
	if err != nil {
		return "", "", "", nil, nil, 0, err
	}
	return cfg.Method, cfg.URL, cfg.Body, cfg.Headers, cfg.Query, cfg.TimeoutSeconds, nil
}

// previewOutputFormat narrows execution formats down to the preview-safe formats we support.
func previewOutputFormat(format string) string {
	if format == "json" {
		return "json"
	}
	return "text"
}

func printRequestUsage() {
	fmt.Println(`Usage: raco req [options]

Options:
  -m <method>   HTTP method (default: GET)
  -r <url>      Request URL (required)
  -d <body>     Request body
  -H <hdr>      Headers (Key:Value, multiple separated by ;)
  -q <query>    Query params (key=value, multiple separated by ;)
  -t <sec>      Timeout in seconds (0 = default 30)
  -f <file>     File upload (field_name:path)
  -o <format>   Output: body, json, full
  -e <name>     Environment name
  --dry-run     Resolve request without sending it

Examples:
  raco req -m GET -r https://api.example.org
  raco req -m GET -r https://api.example.org -q "page=1;limit=10"
  raco req -m POST -r https://api.example.org -d '{"key":"value"}' -t 60
  raco req -m POST -r https://api.example.org -e production --dry-run -o json`)
}
