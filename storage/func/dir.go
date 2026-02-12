package storagefunc

import (
	"os"
	"path/filepath"
)

func EnsureDir(basePath string) error {
	dirs := []string{
		filepath.Join(basePath, "collections"),
		filepath.Join(basePath, "environments"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}
