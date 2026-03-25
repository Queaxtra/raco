package collection

import (
	"encoding/json"
	"os"
	"path/filepath"
	"raco/model"
	storagefunc "raco/storage/func"
	"raco/util"
)

func writeRevision(basePath string, col *model.Collection) error {
	if col == nil {
		return nil
	}

	if err := storagefunc.EnsureBaseDirs(basePath); err != nil {
		return err
	}

	historyDir := filepath.Join(basePath, "history", col.ID)
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		return err
	}

	revision := model.CollectionRevision{
		CollectionID: col.ID,
		Revision:     col.Revision,
		UpdatedAt:    col.UpdatedAt,
		Collection:   sanitizedCollection(*col),
	}
	data, err := json.MarshalIndent(revision, "", "  ")
	if err != nil {
		return err
	}

	name := col.UpdatedAt.UTC().Format("20060102T150405.000000000Z07:00") + ".json"
	path := filepath.Join(historyDir, name)
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	return os.Rename(tempPath, path)
}

func sanitizedCollection(col model.Collection) model.Collection {
	copyCol := col
	copyCol.Requests = make([]*model.Request, 0, len(col.Requests))
	for _, req := range col.Requests {
		if req == nil {
			continue
		}

		copyReq := *req
		copyReq.URL = util.RedactSensitiveData(copyReq.URL)
		copyReq.Body = util.RedactSensitiveData(copyReq.Body)
		copyReq.Headers = util.RedactHeaders(copyReq.Headers)
		copyCol.Requests = append(copyCol.Requests, &copyReq)
	}
	return copyCol
}
