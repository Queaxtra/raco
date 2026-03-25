package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// IsPathContainedOrEqual treats the base directory itself as valid while still
// rejecting any path that escapes above the base via traversal segments.
func IsPathContainedOrEqual(path, base string) bool {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return false
	}
	cleaned := filepath.Clean(rel)
	if cleaned == "." {
		return true
	}
	if cleaned == ".." {
		return false
	}
	return !strings.HasPrefix(cleaned, ".."+string(filepath.Separator))
}

// ResolveExistingDir walks upward until it finds an existing directory, then
// rebuilds the unresolved suffix on top of the canonical location. This lets us
// validate future output paths before the final file or subdirectory exists.
func ResolveExistingDir(dir string) (string, error) {
	current := filepath.Clean(dir)
	visited := make([]string, 0, 4)
	for {
		visited = append(visited, current)
		info, err := os.Stat(current)
		if err == nil && info.IsDir() {
			resolved, err := filepath.EvalSymlinks(current)
			if err != nil {
				return "", err
			}
			for i := len(visited) - 2; i >= 0; i-- {
				resolved = filepath.Join(resolved, filepath.Base(visited[i]))
			}
			return resolved, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("base directory not found")
		}
		current = parent
	}
}

// ResolveContainedPath canonicalizes a target path relative to a trusted base
// directory and fails closed if symlinks or traversal would move it outside.
func ResolveContainedPath(basePath string, targetPath string) (string, error) {
	if strings.TrimSpace(basePath) == "" {
		return "", fmt.Errorf("base path is required")
	}
	if strings.TrimSpace(targetPath) == "" {
		return "", fmt.Errorf("target path is required")
	}

	resolvedBase, err := filepath.EvalSymlinks(filepath.Clean(basePath))
	if err != nil {
		return "", err
	}

	finalPath := filepath.Clean(targetPath)
	if !filepath.IsAbs(finalPath) {
		finalPath = filepath.Join(resolvedBase, finalPath)
	}

	resolvedDir, err := ResolveExistingDir(filepath.Dir(finalPath))
	if err != nil {
		return "", err
	}

	finalPath = filepath.Join(resolvedDir, filepath.Base(finalPath))
	if !IsPathContainedOrEqual(finalPath, resolvedBase) {
		return "", fmt.Errorf("path escapes base directory")
	}
	return finalPath, nil
}

// ReadFileBounded keeps all read-side helpers on the same contract: files must
// exist, must not be directories, and must fit inside an explicit size budget.
func ReadFileBounded(path string, maxSize int64) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("file path is empty")
	}
	if maxSize <= 0 {
		return nil, fmt.Errorf("max size must be positive")
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory")
	}
	if info.Size() > maxSize {
		return nil, fmt.Errorf("file too large")
	}
	return os.ReadFile(path)
}
