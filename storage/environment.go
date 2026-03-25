package storage

import (
	"raco/model"
	"raco/secretstore"
	"raco/storage/func/environment"
)

func (s *Storage) SaveEnvironment(env *model.Environment) error {
	return environment.Save(s.basePath, env)
}

func (s *Storage) LoadEnvironment(name string) (*model.Environment, error) {
	return environment.Load(s.basePath, name)
}

func (s *Storage) LoadMergedEnvironment(name string) (*model.Environment, error) {
	return environment.LoadMerged(s.basePath, name)
}

func (s *Storage) ResolveEnvironment(name string, store secretstore.Store) (*model.ResolvedEnvironment, error) {
	return environment.Resolve(s.basePath, name, store)
}
