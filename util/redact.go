package util

import (
	"regexp"
	"strings"
)

var sensitivePatterns = []string{
	"password",
	"secret",
	"token",
	"api_key",
	"api-key",
	"api key",
	"apikey",
	"auth",
	"bearer",
	"credential",
	"private",
}

var emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
var tokenRegex = regexp.MustCompile(`[A-Za-z0-9_-]{32,}`)
var bearerRegex = regexp.MustCompile(`Bearer\s+[A-Za-z0-9._-]+`)

// sensitiveJSONPatterns is compiled once to avoid repeated regexp.Compile inside RedactJSON.
var sensitiveJSONPatterns []*regexp.Regexp

func init() {
	sensitiveJSONPatterns = make([]*regexp.Regexp, 0, len(sensitivePatterns))
	for _, pattern := range sensitivePatterns {
		re := regexp.MustCompile(`"` + regexp.QuoteMeta(pattern) + `"\s*:\s*"[^"]*"`)
		sensitiveJSONPatterns = append(sensitiveJSONPatterns, re)
	}
}

func RedactSensitiveData(data string) string {
	if data == "" {
		return data
	}

	redacted := data
	redacted = emailRegex.ReplaceAllString(redacted, "[REDACTED_EMAIL]")
	redacted = bearerRegex.ReplaceAllString(redacted, "Bearer [REDACTED_TOKEN]")
	redacted = tokenRegex.ReplaceAllString(redacted, "[REDACTED_TOKEN]")

	return redacted
}

func RedactHeaders(headers map[string]string) map[string]string {
	if headers == nil {
		return nil
	}

	redacted := make(map[string]string)
	for key, value := range headers {
		lowerKey := strings.ToLower(key)
		shouldRedact := false

		for _, pattern := range sensitivePatterns {
			if strings.Contains(lowerKey, pattern) {
				shouldRedact = true
				break
			}
		}

		if shouldRedact {
			redacted[key] = "[REDACTED]"
		}
		if !shouldRedact {
			redacted[key] = value
		}
	}

	return redacted
}

func RedactJSON(json string) string {
	if json == "" {
		return json
	}

	redacted := json

	for i, re := range sensitiveJSONPatterns {
		replacement := `"` + sensitivePatterns[i] + `": "[REDACTED]"`
		redacted = re.ReplaceAllString(redacted, replacement)
	}

	return redacted
}

func IsSensitiveKey(key string) bool {
	if key == "" {
		return false
	}

	lowerKey := strings.ToLower(key)

	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerKey, pattern) {
			return true
		}
	}

	return false
}
