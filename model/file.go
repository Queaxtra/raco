package model

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileUpload struct {
	FieldName   string `json:"field_name" yaml:"field_name"`
	FilePath    string `json:"file_path" yaml:"file_path"`
	FileName    string `json:"file_name" yaml:"file_name"`
	ContentType string `json:"content_type" yaml:"content_type"`
	Size        int64  `json:"size" yaml:"size"`
}

// Validate converts a user-supplied file reference into a canonical, bounded,
// workspace-contained upload target before any bytes are read.
func (f *FileUpload) Validate() error {
	if f.FieldName == "" {
		return fmt.Errorf("field name is required")
	}

	if f.FilePath == "" {
		return fmt.Errorf("file path is required")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot resolve working directory: %w", err)
	}

	// workspace containment prevents persisted collections from exfiltrating arbitrary local files.
	resolvedPath, err := resolveContainedFilePath(cwd, f.FilePath)
	if err != nil {
		return fmt.Errorf("cannot resolve file path: %w", err)
	}
	f.FilePath = resolvedPath

	info, err := os.Stat(f.FilePath)
	if err != nil {
		return fmt.Errorf("cannot access file: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file")
	}

	const maxFileSize = 100 * 1024 * 1024 // 100MB limit
	if info.Size() > maxFileSize {
		return fmt.Errorf("file too large (max 100MB)")
	}

	f.Size = info.Size()

	if f.FileName == "" {
		f.FileName = filepath.Base(f.FilePath)
	}

	return nil
}

// ReadData intentionally assumes Validate already ran so callers can keep their
// happy path small while still benefiting from earlier safety checks.
func (f *FileUpload) ReadData() ([]byte, error) {
	return os.ReadFile(f.FilePath)
}

// resolveContainedFilePath mirrors the stronger containment rules used by the
// CLI helpers without introducing a package cycle from model to util.
func resolveContainedFilePath(basePath string, targetPath string) (string, error) {
	resolvedBase, err := filepath.EvalSymlinks(filepath.Clean(basePath))
	if err != nil {
		return "", err
	}
	finalPath := filepath.Clean(targetPath)
	if !filepath.IsAbs(finalPath) {
		finalPath = filepath.Join(resolvedBase, finalPath)
	}
	resolvedDir, err := resolveExistingFileDir(filepath.Dir(finalPath))
	if err != nil {
		return "", err
	}
	finalPath = filepath.Join(resolvedDir, filepath.Base(finalPath))
	rel, err := filepath.Rel(resolvedBase, finalPath)
	if err != nil {
		return "", err
	}
	cleaned := filepath.Clean(rel)
	if cleaned == ".." {
		return "", fmt.Errorf("path escapes base directory")
	}
	if strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes base directory")
	}
	return finalPath, nil
}

// resolveExistingFileDir canonicalizes the nearest existing parent directory so
// we can safely reason about not-yet-created descendants.
func resolveExistingFileDir(dir string) (string, error) {
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

type FileDownload struct {
	FilePath     string `json:"file_path" yaml:"file_path"`
	OriginalName string `json:"original_name" yaml:"original_name"`
	ContentType  string `json:"content_type" yaml:"content_type"`
	Size         int64  `json:"size" yaml:"size"`
}
