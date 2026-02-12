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

	if !validIDPattern.MatchString(col.ID) {
		return errors.New("invalid collection ID format")
	}

	if err := storagefunc.EnsureDir(filepath.Join(basePath, "collections")); err != nil {
		return err
	}

	path := filepath.Join(basePath, "collections", col.ID+".json")

	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		resolvedPath = path
	}

	expectedDir := filepath.Join(basePath, "collections")
	if !isPathContained(resolvedPath, expectedDir) {
		return errors.New("path traversal detected")
	}

	tempPath := resolvedPath + ".tmp"
	data, err := json.MarshalIndent(col, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		os.Remove(tempPath)
		return err
	}

	return os.Rename(tempPath, resolvedPath)
}
