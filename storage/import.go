package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"raco/model"
	"raco/util"
	"time"
)

type PostmanCollection struct {
	Info struct {
		Name string `json:"name"`
	} `json:"info"`
	Item []PostmanItem `json:"item"`
}

type PostmanItem struct {
	Name    string              `json:"name"`
	Request *PostmanRequest     `json:"request,omitempty"`
	Item    []PostmanItem       `json:"item,omitempty"`
}

type PostmanRequest struct {
	Method string              `json:"method"`
	Header []PostmanHeader     `json:"header"`
	Body   *PostmanBody        `json:"body,omitempty"`
	URL    interface{}         `json:"url"`
}

type PostmanHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type PostmanBody struct {
	Mode string `json:"mode"`
	Raw  string `json:"raw"`
}

func ImportPostmanCollection(filePath string) (*model.Collection, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path is empty")
	}

	cleanPath := filepath.Clean(filePath)
	info, err := os.Stat(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access file: %w", err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file")
	}

	if info.Size() > 10*1024*1024 {
		return nil, fmt.Errorf("file too large (max 10MB)")
	}

	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	maxFileSize := int64(10 * 1024 * 1024)
	limitedReader := io.LimitReader(file, maxFileSize)

	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var postman PostmanCollection
	if err := json.Unmarshal(data, &postman); err != nil {
		return nil, fmt.Errorf("invalid postman format: %w", err)
	}

	collection := &model.Collection{
		ID:       util.GenerateID(),
		Name:     postman.Info.Name,
		Requests: make([]*model.Request, 0),
	}

	extractRequests(postman.Item, collection)

	return collection, nil
}

func extractRequests(items []PostmanItem, collection *model.Collection) {
	for _, item := range items {
		if item.Request != nil {
			req := convertPostmanRequest(item.Name, item.Request)
			if req != nil {
				collection.Requests = append(collection.Requests, req)
			}
		}

		if len(item.Item) > 0 {
			extractRequests(item.Item, collection)
		}
	}
}

func convertPostmanRequest(name string, pr *PostmanRequest) *model.Request {
	if pr == nil {
		return nil
	}

	req := &model.Request{
		ID:        util.GenerateID(),
		Name:      name,
		Method:    pr.Method,
		Headers:   make(map[string]string),
		CreatedAt: time.Now(),
	}

	if pr.Header != nil {
		for _, h := range pr.Header {
			req.Headers[h.Key] = h.Value
		}
	}

	if pr.URL != nil {
		switch url := pr.URL.(type) {
		case string:
			req.URL = url
		case map[string]interface{}:
			if raw, ok := url["raw"].(string); ok {
				req.URL = raw
			}
		}
	}

	if pr.Body != nil {
		if pr.Body.Mode == "raw" {
			req.Body = pr.Body.Raw
		}
	}

	return req
}
