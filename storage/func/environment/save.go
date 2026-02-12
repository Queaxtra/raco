package environment

import (
	"errors"
	"os"
	"path/filepath"
	"raco/model"
	"raco/storage/func"

	"gopkg.in/yaml.v3"
)

func Save(basePath string, env *model.Environment) error {
	if env == nil {
		return errors.New("environment is nil")
	}

	if !validEnvNamePattern.MatchString(env.Name) {
		return errors.New("invalid environment name format")
	}

	if err := storagefunc.EnsureDir(filepath.Join(basePath, "environments")); err != nil {
		return err
	}

	path := filepath.Join(basePath, "environments", env.Name+".yaml")

	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		resolvedPath = path
	}

	expectedDir := filepath.Join(basePath, "environments")
	if !isPathContained(resolvedPath, expectedDir) {
		return errors.New("path traversal detected")
	}

	tempPath := resolvedPath + ".tmp"
	data, err := yaml.Marshal(env)
	if err != nil {
		return err
	}

	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		os.Remove(tempPath)
		return err
	}

	return os.Rename(tempPath, resolvedPath)
}
