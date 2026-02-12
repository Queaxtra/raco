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

	collections := make([]*model.Collection, 0)
	for _, entry := range entries {
		isFile := !entry.IsDir()
		isJSON := filepath.Ext(entry.Name()) == ".json"
		if isFile && isJSON {
			id := entry.Name()[:len(entry.Name())-5]
			col, err := Load(basePath, id)
			if err == nil {
				collections = append(collections, col)
			}
		}
	}

	return collections, nil
}
