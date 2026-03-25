package storagefunc

import (
	"os"
	"path/filepath"
)

func EnsureDir(basePath string) error {
	return EnsureBaseDirs(basePath)
}

func EnsureBaseDirs(basePath string) error {
	dirs := []string{
		filepath.Join(basePath, "collections"),
		filepath.Join(basePath, "environments"),
		filepath.Join(basePath, "history"),
		filepath.Join(basePath, "snapshots"),
		filepath.Join(basePath, "cookies"),
		filepath.Join(basePath, "scripts"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}
