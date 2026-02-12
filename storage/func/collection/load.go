package collection

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"raco/model"
	"regexp"
)

var validIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

func Load(basePath string, id string) (*model.Collection, error) {
	if !validIDPattern.MatchString(id) {
		return nil, errors.New("invalid collection ID format")
	}

	path := filepath.Join(basePath, "collections", id+".json")

	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		resolvedPath = path
	}

	expectedDir := filepath.Join(basePath, "collections")
	if !isPathContained(resolvedPath, expectedDir) {
		return nil, errors.New("path traversal detected")
	}

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, err
	}

	var col model.Collection
	if err := json.Unmarshal(data, &col); err != nil {
		return nil, err
	}

	return &col, nil
}

func isPathContained(path, base string) bool {
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
