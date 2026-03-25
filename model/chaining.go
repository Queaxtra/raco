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
	ExtractCookie   ExtractionType = "cookie"
	ExtractStatus   ExtractionType = "status"
	ExtractBody     ExtractionType = "body"
	ExtractRegexKey ExtractionType = "regex_named"
)

type Extractor struct {
	Type    ExtractionType `json:"type" yaml:"type"`
	Source  string         `json:"source" yaml:"source"`
	Target  string         `json:"target" yaml:"target"`
	Pattern string         `json:"pattern,omitempty" yaml:"pattern,omitempty"`
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
	if extractor.Type == ExtractCookie {
		value, err = extractFromCookie(response.Headers, extractor.Source)
		if err != nil {
			return err
		}
	}
	if extractor.Type == ExtractStatus {
		value = fmt.Sprintf("%d", response.StatusCode)
	}
	if extractor.Type == ExtractBody {
		value = response.Body
	}
	if extractor.Type == ExtractRegexKey {
		value, err = extractFromNamedRegex(response.Body, extractor.Pattern, extractor.Source)
		if err != nil {
			return err
		}
	}

	if value == "" {
		return fmt.Errorf("extracted value is empty")
	}

	if env.Variables == nil {
		env.Variables = make(map[string]EnvironmentVariable)
	}

	env.Variables[extractor.Target] = EnvironmentVariable{
		Kind:  EnvironmentVariablePlain,
		Value: value,
	}
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

func extractFromCookie(headers map[string]string, key string) (string, error) {
	headerValue, err := extractFromHeader(headers, "Set-Cookie")
	if err != nil {
		return "", err
	}
	pairs := regexp.MustCompile(`;\s*`).Split(headerValue, -1)
	for _, pair := range pairs {
		nameValue := regexp.MustCompile(`=`).Split(pair, 2)
		if len(nameValue) != 2 {
			continue
		}
		if nameValue[0] == key {
			return nameValue[1], nil
		}
	}
	return "", fmt.Errorf("cookie not found: %s", key)
}

func extractFromNamedRegex(body string, pattern string, name string) (string, error) {
	if body == "" {
		return "", fmt.Errorf("body is empty")
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex: %w", err)
	}
	match := re.FindStringSubmatch(body)
	if len(match) == 0 {
		return "", fmt.Errorf("no match found")
	}
	index := -1
	for idx, item := range re.SubexpNames() {
		if item == name {
			index = idx
			break
		}
	}
	if index == -1 || index >= len(match) {
		return "", fmt.Errorf("named group not found: %s", name)
	}
	return match[index], nil
}
