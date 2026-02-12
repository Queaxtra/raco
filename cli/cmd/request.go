package cmd

import (
	"flag"
	"fmt"
	"os"
	"raco/cli/output"
	"raco/http"
	"raco/model"
	"strings"
)

type requestConfig struct {
	Method      string
	URL         string
	Headers     map[string]string
	Body        string
	Output      string
	Environment string
}

func RunRequest(ctx *Context, args []string) int {
	cfg, err := parseRequestArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	req := &model.Request{
		Method:  strings.ToUpper(cfg.Method),
		URL:     cfg.URL,
		Headers: cfg.Headers,
		Body:    cfg.Body,
	}

	if cfg.Environment != "" {
		env, err := ctx.Storage().LoadEnvironment(cfg.Environment)
		if err == nil {
			req.URL = http.ReplaceEnvVars(req.URL, env)
			req.Body = http.ReplaceEnvVars(req.Body, env)
			for k, v := range req.Headers {
				req.Headers[k] = http.ReplaceEnvVars(v, env)
			}
		}
	}

	client := http.NewClient()
	resp, err := client.Execute(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	return output.PrintResponse(resp, cfg.Output)
}

func parseRequestArgs(args []string) (*requestConfig, error) {
	fs := flag.NewFlagSet("request", flag.ContinueOnError)

	method := fs.String("m", "GET", "HTTP method")
	url := fs.String("r", "", "Request URL")
	body := fs.String("d", "", "Request body")
	headers := fs.String("H", "", "Headers (format: Key:Value, multiple separated by ;)")
	outputFmt := fs.String("o", "body", "Output format: body, json, full")
	env := fs.String("e", "", "Environment name")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if *url == "" {
		return nil, fmt.Errorf("URL is required (-r)")
	}

	cfg := &requestConfig{
		Method:      *method,
		URL:         *url,
		Body:        *body,
		Headers:     make(map[string]string),
		Output:      *outputFmt,
		Environment: *env,
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

	return cfg, nil
}

func ParseRequestArgsPublic(args []string) (method, url, body string, headers map[string]string, err error) {
	cfg, err := parseRequestArgs(args)
	if err != nil {
		return "", "", "", nil, err
	}
	return cfg.Method, cfg.URL, cfg.Body, cfg.Headers, nil
}
