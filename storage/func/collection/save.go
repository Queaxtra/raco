package collection

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"raco/model"
	"raco/storage/func"
)

func Save(basePath string, col *model.Collection) error {
	if col == nil {
		return errors.New("collection is nil")
	}

	model.NormalizeCollection(col)

	if !validIDPattern.MatchString(col.ID) {
		return errors.New("invalid collection ID format")
	}

	if err := storagefunc.EnsureBaseDirs(basePath); err != nil {
		return err
	}

	path := filepath.Join(basePath, "collections", col.ID+".json")
	expectedDir := filepath.Join(basePath, "collections")
	if resolvedExpectedDir, resolveErr := filepath.EvalSymlinks(expectedDir); resolveErr == nil {
		expectedDir = resolvedExpectedDir
	}

	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		resolvedPath = filepath.Join(expectedDir, filepath.Base(path))
	}

	if !isPathContained(resolvedPath, expectedDir) {
		return errors.New("path traversal detected")
	}

	tempPath := resolvedPath + ".tmp"
	col.Touch()
	data, err := json.MarshalIndent(col, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		os.Remove(tempPath)
		return err
	}

	if err := os.Rename(tempPath, resolvedPath); err != nil {
		return err
	}

	return writeRevision(basePath, col)
}
