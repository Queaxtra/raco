package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"raco/model"
	"raco/util"
	"strings"
	"time"
)

type harDocument struct {
	Log harLog `json:"log"`
}

type harLog struct {
	Version string     `json:"version"`
	Creator harCreator `json:"creator"`
	Entries []harEntry `json:"entries"`
}

type harCreator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type harEntry struct {
	StartedDateTime string      `json:"startedDateTime,omitempty"`
	Time            float64     `json:"time,omitempty"`
	Request         harRequest  `json:"request"`
	Response        harResponse `json:"response"`
}

type harRequest struct {
	Method      string       `json:"method"`
	URL         string       `json:"url"`
	Headers     []harKV      `json:"headers"`
	QueryString []harKV      `json:"queryString"`
	PostData    *harPostData `json:"postData,omitempty"`
}

type harResponse struct {
	Status  int     `json:"status"`
	Content harBody `json:"content"`
}

type harPostData struct {
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
}

type harBody struct {
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
}

type harKV struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func ImportHARCollection(filePath string) (*model.Collection, error) {
	data, err := readBoundedFile(filePath, 10*1024*1024)
	if err != nil {
		return nil, err
	}

	var doc harDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("invalid har format: %w", err)
	}
	if len(doc.Log.Entries) == 0 {
		return nil, fmt.Errorf("har document has no entries")
	}

	col := &model.Collection{
		ID:        util.GenerateID(),
		Name:      "HAR Import",
		Requests:  make([]*model.Request, 0, len(doc.Log.Entries)),
		Tags:      make([]string, 0),
		Contracts: make([]model.ContractProfile, 0),
	}

	for idx, entry := range doc.Log.Entries {
		req := &model.Request{
			ID:           util.GenerateID(),
			Name:         fmt.Sprintf("%s %s", entry.Request.Method, entry.Request.URL),
			Method:       strings.ToUpper(entry.Request.Method),
			URL:          strings.TrimSpace(entry.Request.URL),
			Query:        make(map[string]string),
			Headers:      make(map[string]string),
			Files:        make([]model.FileUpload, 0),
			Assertions:   make([]model.Assertion, 0),
			Extractors:   make([]model.Extractor, 0),
			CreatedAt:    time.Now().UTC().Add(time.Duration(idx) * time.Millisecond),
			CollectionID: col.ID,
		}

		for _, header := range entry.Request.Headers {
			req.Headers[header.Name] = header.Value
		}
		for _, item := range entry.Request.QueryString {
			req.Query[item.Name] = item.Value
		}
		if entry.Request.PostData != nil {
			req.Body = entry.Request.PostData.Text
			if entry.Request.PostData.MimeType != "" {
				req.Headers["Content-Type"] = entry.Request.PostData.MimeType
			}
		}

		model.NormalizeRequest(req)
		col.Requests = append(col.Requests, req)
	}

	model.NormalizeCollection(col)
	return col, nil
}

func ExportHARCollection(col *model.Collection, filePath string) error {
	if col == nil {
		return fmt.Errorf("collection is nil")
	}

	model.NormalizeCollection(col)
	doc := harDocument{
		Log: harLog{
			Version: "1.2",
			Creator: harCreator{Name: "raco", Version: "dev"},
			Entries: make([]harEntry, 0, len(col.Requests)),
		},
	}

	for _, req := range col.Requests {
		if req == nil {
			continue
		}

		entry := harEntry{
			StartedDateTime: req.CreatedAt.UTC().Format(time.RFC3339Nano),
			Request: harRequest{
				Method:      req.Method,
				URL:         req.URL,
				Headers:     make([]harKV, 0, len(req.Headers)),
				QueryString: make([]harKV, 0, len(req.Query)),
			},
			Response: harResponse{
				Status: 0,
			},
		}

		for key, value := range req.Headers {
			entry.Request.Headers = append(entry.Request.Headers, harKV{Name: key, Value: value})
		}
		for key, value := range req.Query {
			entry.Request.QueryString = append(entry.Request.QueryString, harKV{Name: key, Value: value})
		}
		if strings.TrimSpace(req.Body) != "" {
			entry.Request.PostData = &harPostData{
				MimeType: req.Headers["Content-Type"],
				Text:     req.Body,
			}
		}

		doc.Log.Entries = append(doc.Log.Entries, entry)
	}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}

	cleanPath := filepath.Clean(filePath)
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0755); err != nil {
		return err
	}
	tempPath := cleanPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	return os.Rename(tempPath, cleanPath)
}
