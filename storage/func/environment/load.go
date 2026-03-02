package environment

import (
	"errors"
	"os"
	"path/filepath"
	"raco/model"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var validEnvNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

func Load(basePath string, name string) (*model.Environment, error) {
	if !validEnvNamePattern.MatchString(name) {
		return nil, errors.New("invalid environment name format")
	}

	path := filepath.Join(basePath, "environments", name+".yaml")

	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		resolvedPath = path
	}

	expectedDir := filepath.Join(basePath, "environments")
	if !isPathContained(resolvedPath, expectedDir) {
		return nil, errors.New("path traversal detected")
	}

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, err
	}

	var env model.Environment
	if err := yaml.Unmarshal(data, &env); err != nil {
		return nil, err
	}

	return &env, nil
}

// isPathContained ensures path is under base (prevents ".." traversal), not that path has no dots (e.g. .yaml).
func isPathContained(path, base string) bool {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return false
	}
	cleaned := filepath.Clean(rel)
	return cleaned != ".." && cleaned != "." && !strings.HasPrefix(cleaned, "..")
}
