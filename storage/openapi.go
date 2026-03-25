package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"raco/model"
	"raco/util"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type openAPIDocument struct {
	OpenAPI string                                 `yaml:"openapi" json:"openapi"`
	Info    openAPIInfo                            `yaml:"info" json:"info"`
	Servers []openAPIServer                        `yaml:"servers" json:"servers"`
	Paths   map[string]map[string]openAPIOperation `yaml:"paths" json:"paths"`
}

type openAPIInfo struct {
	Title string `yaml:"title" json:"title"`
}

type openAPIServer struct {
	URL string `yaml:"url" json:"url"`
}

type openAPIOperation struct {
	OperationID string                 `yaml:"operationId" json:"operationId"`
	Summary     string                 `yaml:"summary" json:"summary"`
	Tags        []string               `yaml:"tags" json:"tags"`
	Parameters  []openAPIParameter     `yaml:"parameters" json:"parameters"`
	RequestBody *openAPIRequestBody    `yaml:"requestBody" json:"requestBody"`
	Extensions  map[string]interface{} `yaml:",inline" json:"-"`
}

type openAPIParameter struct {
	Name string `yaml:"name" json:"name"`
	In   string `yaml:"in" json:"in"`
}

type openAPIRequestBody struct {
	Content map[string]openAPIMediaType `yaml:"content" json:"content"`
}

type openAPIMediaType struct {
	Example interface{} `yaml:"example" json:"example"`
}

func ImportOpenAPICollection(filePath string) (*model.Collection, error) {
	data, err := readBoundedFile(filePath, 10*1024*1024)
	if err != nil {
		return nil, err
	}

	var doc openAPIDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("invalid openapi format: %w", err)
	}
	if len(doc.Paths) == 0 {
		return nil, fmt.Errorf("openapi document has no paths")
	}

	col := &model.Collection{
		ID:        util.GenerateID(),
		Name:      strings.TrimSpace(doc.Info.Title),
		Requests:  make([]*model.Request, 0),
		Tags:      make([]string, 0),
		Contracts: make([]model.ContractProfile, 0),
	}
	if col.Name == "" {
		col.Name = "OpenAPI Import"
	}

	baseURL := ""
	if len(doc.Servers) > 0 {
		baseURL = strings.TrimRight(strings.TrimSpace(doc.Servers[0].URL), "/")
	}

	pathKeys := make([]string, 0, len(doc.Paths))
	for path := range doc.Paths {
		pathKeys = append(pathKeys, path)
	}
	sort.Strings(pathKeys)

	for _, path := range pathKeys {
		ops := doc.Paths[path]
		methods := make([]string, 0, len(ops))
		for method := range ops {
			methods = append(methods, method)
		}
		sort.Strings(methods)
		for _, method := range methods {
			op := ops[method]
			name := strings.TrimSpace(op.Summary)
			if name == "" {
				name = strings.TrimSpace(op.OperationID)
			}
			if name == "" {
				name = strings.ToUpper(method) + " " + path
			}

			req := &model.Request{
				ID:           util.GenerateID(),
				Name:         name,
				Method:       strings.ToUpper(method),
				URL:          buildImportedURL(baseURL, path),
				Query:        make(map[string]string),
				Headers:      make(map[string]string),
				Files:        make([]model.FileUpload, 0),
				Assertions:   make([]model.Assertion, 0),
				Extractors:   make([]model.Extractor, 0),
				Tags:         append([]string(nil), op.Tags...),
				CreatedAt:    time.Now().UTC(),
				CollectionID: col.ID,
			}

			for _, parameter := range op.Parameters {
				if strings.EqualFold(parameter.In, "query") {
					req.Query[parameter.Name] = ""
				}
				if strings.EqualFold(parameter.In, "header") {
					req.Headers[parameter.Name] = ""
				}
			}

			if op.RequestBody != nil {
				if media, ok := op.RequestBody.Content["application/json"]; ok {
					req.Headers["Content-Type"] = "application/json"
					if media.Example != nil {
						body, marshalErr := yaml.Marshal(media.Example)
						if marshalErr == nil {
							req.Body = strings.TrimSpace(string(body))
						}
					}
				}
			}

			model.NormalizeRequest(req)
			col.Requests = append(col.Requests, req)
		}
	}

	model.NormalizeCollection(col)
	return col, nil
}

func ExportOpenAPICollection(col *model.Collection, filePath string) error {
	if col == nil {
		return fmt.Errorf("collection is nil")
	}

	model.NormalizeCollection(col)
	doc := openAPIDocument{
		OpenAPI: "3.1.0",
		Info: openAPIInfo{
			Title: col.Name,
		},
		Paths: make(map[string]map[string]openAPIOperation),
	}

	for _, req := range col.Requests {
		if req == nil {
			continue
		}

		path := exportedOpenAPIPath(req.URL)
		method := strings.ToLower(strings.TrimSpace(req.Method))
		if doc.Paths[path] == nil {
			doc.Paths[path] = make(map[string]openAPIOperation)
		}

		op := openAPIOperation{
			OperationID: req.ID,
			Summary:     req.Name,
			Tags:        append([]string(nil), req.Tags...),
			Parameters:  make([]openAPIParameter, 0, len(req.Query)+len(req.Headers)),
		}

		for key := range req.Query {
			op.Parameters = append(op.Parameters, openAPIParameter{Name: key, In: "query"})
		}
		for key := range req.Headers {
			op.Parameters = append(op.Parameters, openAPIParameter{Name: key, In: "header"})
		}

		if strings.TrimSpace(req.Body) != "" {
			op.RequestBody = &openAPIRequestBody{
				Content: map[string]openAPIMediaType{
					"application/json": {Example: req.Body},
				},
			}
		}

		doc.Paths[path][method] = op
	}

	return writeYAMLFile(filePath, doc)
}

func buildImportedURL(baseURL string, path string) string {
	if baseURL == "" {
		return "https://example.invalid" + path
	}
	if strings.HasPrefix(path, "/") {
		return baseURL + path
	}
	return baseURL + "/" + path
}

func exportedOpenAPIPath(rawURL string) string {
	clean := strings.TrimSpace(rawURL)
	if clean == "" {
		return "/"
	}
	if strings.HasPrefix(clean, "https://") {
		trimmed := strings.TrimPrefix(clean, "https://")
		slash := strings.IndexByte(trimmed, '/')
		if slash == -1 {
			return "/"
		}
		return trimmed[slash:]
	}
	return clean
}

func readBoundedFile(filePath string, maxSize int64) ([]byte, error) {
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
	if info.Size() > maxSize {
		return nil, fmt.Errorf("file too large (max %d bytes)", maxSize)
	}
	return os.ReadFile(cleanPath)
}

func writeYAMLFile(filePath string, value interface{}) error {
	if filePath == "" {
		return fmt.Errorf("file path is empty")
	}
	cleanPath := filepath.Clean(filePath)
	data, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
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
