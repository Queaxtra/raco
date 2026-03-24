package util

import (
	"path/filepath"
	"raco/util/func/validate"
	"regexp"
	"strings"
)

var envKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,63}$`)

func ValidateURL(rawURL string) bool {
	return validate.URL(rawURL)
}

func ValidateMethod(method string) bool {
	return validate.Method(method)
}

func IsPathContained(path, base string) bool {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return false
	}
	cleaned := filepath.Clean(rel)
	if cleaned == ".." {
		return false
	}
	if cleaned == "." {
		return false
	}
	if strings.HasPrefix(cleaned, "..") {
		return false
	}
	return true
}

func ValidateWebSocketURL(rawURL string) bool {
	return validate.WebSocketURL(rawURL)
}

func ValidateGRPCTarget(target string) bool {
	return validate.GRPCTarget(target)
}

func ValidateEnvironmentKey(key string) bool {
	return envKeyPattern.MatchString(key)
}
