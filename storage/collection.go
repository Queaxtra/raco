package storage

import (
	"raco/model"
	"raco/storage/func/collection"
)

func (s *Storage) SaveCollection(col *model.Collection) error {
	return collection.Save(s.basePath, col)
}

func (s *Storage) LoadCollection(id string) (*model.Collection, error) {
	return collection.Load(s.basePath, id)
}

func (s *Storage) ListCollections() ([]*model.Collection, error) {
	return collection.List(s.basePath)
}
