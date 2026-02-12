package model

import (
	"encoding/json"
	"fmt"
	"regexp"
)

type ExtractionType string

const (
	ExtractJSONPath ExtractionType = "jsonpath"
	ExtractRegex    ExtractionType = "regex"
	ExtractHeader   ExtractionType = "header"
)

type Extractor struct {
	Type     ExtractionType `json:"type" yaml:"type"`
	Source   string         `json:"source" yaml:"source"`
	Target   string         `json:"target" yaml:"target"`
	Pattern  string         `json:"pattern,omitempty" yaml:"pattern,omitempty"`
}

func ExtractValue(extractor Extractor, response *Response, env *Environment) error {
	if response == nil {
		return fmt.Errorf("response is nil")
	}

	if env == nil {
		return fmt.Errorf("environment is nil")
	}

	var value string
	var err error

	if extractor.Type == ExtractJSONPath {
		value, err = extractFromJSON(response.Body, extractor.Source)
		if err != nil {
			return err
		}
	}

	if extractor.Type == ExtractRegex {
		value, err = extractFromRegex(response.Body, extractor.Pattern)
		if err != nil {
			return err
		}
	}

	if extractor.Type == ExtractHeader {
		value, err = extractFromHeader(response.Headers, extractor.Source)
		if err != nil {
			return err
		}
	}

	if value == "" {
		return fmt.Errorf("extracted value is empty")
	}

	if env.Variables == nil {
		env.Variables = make(map[string]string)
	}

	env.Variables[extractor.Target] = value
	return nil
}

func extractFromJSON(body string, path string) (string, error) {
	if body == "" {
		return "", fmt.Errorf("body is empty")
	}

	var data interface{}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	value := extractJSONPath(data, path)
	if value == nil {
		return "", fmt.Errorf("path not found: %s", path)
	}

	return fmt.Sprintf("%v", value), nil
}

const maxRegexPatternLen = 4096
const maxBodySizeForRegex = 1024 * 1024

func extractFromRegex(body string, pattern string) (string, error) {
	if body == "" {
		return "", fmt.Errorf("body is empty")
	}

	if pattern == "" {
		return "", fmt.Errorf("pattern is empty")
	}

	if len(pattern) > maxRegexPatternLen {
		return "", fmt.Errorf("regex pattern too long (max 4KB)")
	}

	if len(body) > maxBodySizeForRegex {
		return "", fmt.Errorf("body too large for regex extraction (max 1MB)")
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex: %w", err)
	}

	matches := re.FindStringSubmatch(body)
	if len(matches) < 2 {
		return "", fmt.Errorf("no match found")
	}

	return matches[1], nil
}

func extractFromHeader(headers map[string]string, key string) (string, error) {
	if headers == nil {
		return "", fmt.Errorf("headers is nil")
	}

	value, exists := headers[key]
	if !exists {
		return "", fmt.Errorf("header not found: %s", key)
	}

	return value, nil
}
