package storage

import (
	"raco/model"
	"raco/storage/func/environment"
)

func (s *Storage) SaveEnvironment(env *model.Environment) error {
	return environment.Save(s.basePath, env)
}

func (s *Storage) LoadEnvironment(name string) (*model.Environment, error) {
	return environment.Load(s.basePath, name)
}
