package curl

import (
	"errors"
	"raco/model"
	"raco/util/func/id"
	"raco/util/func/validate"
	"regexp"
	"strings"
	"time"
)

var (
	urlPattern    = regexp.MustCompile(`curl\s+(?:-X\s+\w+\s+)?['"]?([^\s'"]+)`)
	methodPattern = regexp.MustCompile(`-X\s+(\w+)`)
	headerPattern = regexp.MustCompile(`-H\s+['"]([^:]+):\s*([^'"]+)['"]`)
)

func Parse(curlCmd string) (*model.Request, error) {
	if len(curlCmd) > 10000 {
		return nil, errors.New("command too long")
	}

	req := &model.Request{
		ID:        id.Generate(),
		Headers:   make(map[string]string),
		Method:    "GET",
		CreatedAt: time.Now(),
	}

	urlMatches := urlPattern.FindStringSubmatch(curlCmd)
	if len(urlMatches) > 1 {
		req.URL = urlMatches[1]
	}

	if req.URL == "" {
		return nil, errors.New("no URL found in command")
	}

	if !validate.URL(req.URL) {
		return nil, errors.New("invalid or unsafe URL")
	}

	methodMatches := methodPattern.FindStringSubmatch(curlCmd)
	if len(methodMatches) > 1 {
		req.Method = strings.ToUpper(methodMatches[1])
	}

	headerMatches := headerPattern.FindAllStringSubmatch(curlCmd, -1)
	for _, match := range headerMatches {
		if len(match) > 2 {
			req.Headers[strings.TrimSpace(match[1])] = strings.TrimSpace(match[2])
		}
	}

	body := extractDataBody(curlCmd)
	if body != "" {
		req.Body = body
		if req.Method == "GET" {
			req.Method = "POST"
		}
	}

	return req, nil
}

func extractDataBody(curlCmd string) string {
	idx := strings.Index(curlCmd, " -d ")
	if idx == -1 {
		idx = strings.Index(curlCmd, " --data ")
		if idx == -1 {
			return ""
		}
		idx += 8
	}
	if idx != -1 && strings.Index(curlCmd, " -d ") != -1 {
		idx += 4
	}

	remaining := strings.TrimSpace(curlCmd[idx:])
	if len(remaining) == 0 {
		return ""
	}

	var body string
	quote := remaining[0]
	if quote == '\'' || quote == '"' {
		end := findMatchingQuote(remaining[1:], quote)
		if end != -1 {
			body = remaining[1 : end+1]
		}
		if body == "" {
			body = remaining[1:]
		}
	}
	if body == "" {
		spaceIdx := strings.Index(remaining, " ")
		if spaceIdx == -1 {
			body = remaining
		}
		if body == "" {
			body = remaining[:spaceIdx]
		}
	}

	return unescapeBody(body)
}

func unescapeBody(s string) string {
	result := strings.ReplaceAll(s, `\"`, `"`)
	result = strings.ReplaceAll(result, `\\`, `\`)
	return result
}

func findMatchingQuote(s string, quote byte) int {
	escaped := false
	for i := 0; i < len(s); i++ {
		if escaped {
			escaped = false
			continue
		}
		if s[i] == '\\' {
			escaped = true
			continue
		}
		if s[i] == quote {
			return i
		}
	}
	return -1
}
