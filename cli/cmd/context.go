package cmd

import (
	"fmt"
	"raco/model"
	"raco/secretstore"
	"raco/storage"
)

type Context struct {
	StoragePath string
}

func (c *Context) Storage() *storage.Storage {
	return storage.NewStorage(c.StoragePath)
}

// ResolveEnvironment loads an environment into the runtime-only representation used by
// request execution. Secret-backed environments fail closed if the native backend cannot
// be used, while plain-only environments still work without extra OS dependencies.
func (c *Context) ResolveEnvironment(name string) (*model.ResolvedEnvironment, error) {
	store := c.Storage()
	env, err := store.LoadMergedEnvironment(name)
	if err != nil {
		return nil, err
	}

	hasSecrets := false
	for _, variable := range env.Variables {
		if variable.IsSecret() {
			hasSecrets = true
			break
		}
	}
	if !hasSecrets {
		return &model.ResolvedEnvironment{
			Name:      env.Name,
			Variables: env.GetVariables(),
		}, nil
	}

	secretBackend, err := secretstore.NewDefault()
	if err != nil {
		return nil, fmt.Errorf("secure environment resolution failed: %w", err)
	}

	return store.ResolveEnvironment(name, secretBackend)
}
