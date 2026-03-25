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
	BodyFile       string
	Files          []model.FileUpload
	TimeoutSeconds int
	Output         string
	Environment    string
	DryRun         bool
	CookieJar      string
	ProxyURL       string
	CAFile         string
	CertFile       string
	KeyFile        string
	RetryMax       int
	RetryBaseMS    int
	RetryMaxMS     int
	RateLimit      int
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
		BodyFile:       cfg.BodyFile,
		Files:          cfg.Files,
		TimeoutSeconds: cfg.TimeoutSeconds,
		// Cookie jar input stays as a logical name here. The HTTP layer owns the
		// final on-disk mapping so all callers share the same policy.
		CookieJar: resolveCookieJarPath(cfg.CookieJar),
		Transport: model.TransportConfig{
			ProxyURL:        cfg.ProxyURL,
			CAFile:          cfg.CAFile,
			CertFile:        cfg.CertFile,
			KeyFile:         cfg.KeyFile,
			RateLimitPerMin: cfg.RateLimit,
		},
		Retry: model.RetryPolicy{
			MaxAttempts: cfg.RetryMax,
			BaseDelayMS: cfg.RetryBaseMS,
			MaxDelayMS:  cfg.RetryMaxMS,
		},
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

// parseRequestArgs keeps the CLI parsing rules in one place so collection add,
// direct execution, and tests all rely on the same normalization behavior.
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
	bodyFile := fs.String("body-file", "", "Load request body from file")
	cookieJar := fs.String("cookie-jar", "", "Cookie jar name or path")
	proxyURL := fs.String("proxy", "", "Proxy URL")
	caFile := fs.String("ca", "", "CA bundle path")
	certFile := fs.String("cert", "", "Client certificate path")
	keyFile := fs.String("key", "", "Client private key path")
	retryMax := fs.Int("retry-max", 0, "Maximum retry attempts")
	retryBaseMS := fs.Int("retry-base-ms", 0, "Base retry delay in milliseconds")
	retryMaxMS := fs.Int("retry-max-ms", 0, "Maximum retry delay in milliseconds")
	rateLimit := fs.Int("rate-limit", 0, "Maximum request rate per minute")

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
		BodyFile:       *bodyFile,
		Files:          make([]model.FileUpload, 0),
		Headers:        make(map[string]string),
		Query:          make(map[string]string),
		TimeoutSeconds: *timeout,
		Output:         *outputFmt,
		Environment:    *env,
		DryRun:         *dryRun,
		CookieJar:      *cookieJar,
		ProxyURL:       *proxyURL,
		CAFile:         *caFile,
		CertFile:       *certFile,
		KeyFile:        *keyFile,
		RetryMax:       *retryMax,
		RetryBaseMS:    *retryBaseMS,
		RetryMaxMS:     *retryMaxMS,
		RateLimit:      *rateLimit,
	}

	if *query != "" {
		// Query parsing is intentionally simple and deterministic so shell quoting
		// remains understandable for humans and scripts.
		pairs := strings.Split(*query, ";")
		for _, pair := range pairs {
			parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
			if len(parts) == 2 {
				cfg.Query[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	if *headers != "" {
		// Headers follow the same split strategy as query params to keep the CLI
		// grammar small and predictable.
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

// ParseRequestArgsPublic preserves the legacy helper signature used in older
// call sites while delegating all real parsing to parseRequestArgs.
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
  --body-file   Request body file
  -H <hdr>      Headers (Key:Value, multiple separated by ;)
  -q <query>    Query params (key=value, multiple separated by ;)
  -t <sec>      Timeout in seconds (0 = default 30)
  -f <file>     File upload (field_name:path)
  --cookie-jar  Cookie jar name or path
  --proxy       Proxy URL
  --ca          Custom CA bundle
  --cert        Client certificate
  --key         Client private key
  --retry-max   Maximum retry attempts
  --retry-base-ms Base retry delay
  --retry-max-ms Maximum retry delay
  --rate-limit  Max requests per minute
  -o <format>   Output: body, json, full
  -e <name>     Environment name
  --dry-run     Resolve request without sending it

Examples:
  raco req -m GET -r https://api.example.org
  raco req -m GET -r https://api.example.org -q "page=1;limit=10"
  raco req -m POST -r https://api.example.org -d '{"key":"value"}' -t 60
  raco req -m POST -r https://api.example.org -e production --dry-run -o json`)
}

func resolveCookieJarPath(cookieJar string) string {
	cookieJar = strings.TrimSpace(cookieJar)
	if cookieJar == "" {
		return ""
	}
	// The value is intentionally returned as-is because lower layers now own all
	// path validation and canonical storage decisions.
	return cookieJar
}
