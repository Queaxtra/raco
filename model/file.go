package model

import (
	"fmt"
	"os"
	"path/filepath"
)

type FileUpload struct {
	FieldName string `json:"field_name" yaml:"field_name"`
	FilePath  string `json:"file_path" yaml:"file_path"`
	FileName  string `json:"file_name" yaml:"file_name"`
	ContentType string `json:"content_type" yaml:"content_type"`
	Size      int64  `json:"size" yaml:"size"`
}

func (f *FileUpload) Validate() error {
	if f.FieldName == "" {
		return fmt.Errorf("field name is required")
	}

	if f.FilePath == "" {
		return fmt.Errorf("file path is required")
	}

	info, err := os.Stat(f.FilePath)
	if err != nil {
		return fmt.Errorf("cannot access file: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file")
	}

	const maxFileSize = 100 * 1024 * 1024 // 100MB limit
	if info.Size() > maxFileSize {
		return fmt.Errorf("file too large (max 100MB)")
	}

	f.Size = info.Size()

	if f.FileName == "" {
		f.FileName = filepath.Base(f.FilePath)
	}

	return nil
}

func (f *FileUpload) ReadData() ([]byte, error) {
	return os.ReadFile(f.FilePath)
}

type FileDownload struct {
	FilePath     string `json:"file_path" yaml:"file_path"`
	OriginalName string `json:"original_name" yaml:"original_name"`
	ContentType  string `json:"content_type" yaml:"content_type"`
	Size         int64  `json:"size" yaml:"size"`
}
