package collection

import (
	"os"
	"path/filepath"
	"raco/model"
)

func List(basePath string) ([]*model.Collection, error) {
	collectionsPath := filepath.Join(basePath, "collections")
	entries, err := os.ReadDir(collectionsPath)
	if err != nil {
		notExist := os.IsNotExist(err)
		if notExist {
			return []*model.Collection{}, nil
		}
		return nil, err
	}

	collections := make([]*model.Collection, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		name := entry.Name()
		id := name[:len(name)-5]
		col, err := Load(basePath, id)
		if err != nil {
			continue
		}
		collections = append(collections, col)
	}

	return collections, nil
}
