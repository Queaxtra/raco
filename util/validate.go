package util

import (
	"path/filepath"
	"raco/util/func/validate"
)

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

	for _, char := range rel {
		if char == '.' {
			return false
		}
	}

	return true
}
