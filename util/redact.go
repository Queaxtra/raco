package util

import (
	"regexp"
	"sort"
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

// RedactWithSecrets removes runtime secret values first and then applies generic masking.
// The two-step approach catches secrets that do not match the generic token heuristics.
func RedactWithSecrets(data string, secrets []string) string {
	if data == "" {
		return data
	}
	redacted := data
	ordered := make([]string, 0, len(secrets))
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		ordered = append(ordered, secret)
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		return len(ordered[i]) > len(ordered[j])
	})
	for _, secret := range ordered {
		redacted = strings.ReplaceAll(redacted, secret, "[REDACTED]")
	}
	return RedactSensitiveData(redacted)
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

// RedactHeadersWithSecrets combines header-name masking with value-based secret masking.
func RedactHeadersWithSecrets(headers map[string]string, secrets []string) map[string]string {
	redacted := RedactHeaders(headers)
	if redacted == nil {
		return nil
	}
	for key, value := range redacted {
		redacted[key] = RedactWithSecrets(value, secrets)
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
