package model

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type AssertionType string

const (
	AssertStatusCode AssertionType = "status_code"
	AssertJSONPath   AssertionType = "jsonpath"
	AssertRegex      AssertionType = "regex"
	AssertHeader     AssertionType = "header"
)

type Assertion struct {
	Type     AssertionType `json:"type" yaml:"type"`
	Field    string        `json:"field" yaml:"field"`
	Operator string        `json:"operator" yaml:"operator"`
	Value    string        `json:"value" yaml:"value"`
}

type AssertionResult struct {
	Assertion Assertion
	Passed    bool
	Message   string
}

func ValidateAssertion(assertion Assertion, response *Response) AssertionResult {
	if response == nil {
		return AssertionResult{
			Assertion: assertion,
			Passed:    false,
			Message:   "Response is nil",
		}
	}

	if assertion.Type == AssertStatusCode {
		return validateStatusCode(assertion, response)
	}

	if assertion.Type == AssertJSONPath {
		return validateJSONPath(assertion, response)
	}

	if assertion.Type == AssertRegex {
		return validateRegex(assertion, response)
	}

	if assertion.Type == AssertHeader {
		return validateHeader(assertion, response)
	}

	return AssertionResult{
		Assertion: assertion,
		Passed:    false,
		Message:   "Unknown assertion type",
	}
}

func validateStatusCode(assertion Assertion, response *Response) AssertionResult {
	expected := assertion.Value
	actual := strconv.Itoa(response.StatusCode)

	if assertion.Operator == "equals" {
		if actual == expected {
			return AssertionResult{
				Assertion: assertion,
				Passed:    true,
				Message:   fmt.Sprintf("Status code is %s", actual),
			}
		}
		return AssertionResult{
			Assertion: assertion,
			Passed:    false,
			Message:   fmt.Sprintf("Expected %s but got %s", expected, actual),
		}
	}

	if assertion.Operator == "not_equals" {
		if actual != expected {
			return AssertionResult{
				Assertion: assertion,
				Passed:    true,
				Message:   fmt.Sprintf("Status code is not %s", expected),
			}
		}
		return AssertionResult{
			Assertion: assertion,
			Passed:    false,
			Message:   fmt.Sprintf("Status code should not be %s", expected),
		}
	}

	return AssertionResult{
		Assertion: assertion,
		Passed:    false,
		Message:   "Invalid operator for status_code",
	}
}

func validateJSONPath(assertion Assertion, response *Response) AssertionResult {
	if response.Body == "" {
		return AssertionResult{
			Assertion: assertion,
			Passed:    false,
			Message:   "Response body is empty",
		}
	}

	var data interface{}
	if err := json.Unmarshal([]byte(response.Body), &data); err != nil {
		return AssertionResult{
			Assertion: assertion,
			Passed:    false,
			Message:   "Response body is not valid JSON",
		}
	}

	value := extractJSONPath(data, assertion.Field)
	if value == nil {
		return AssertionResult{
			Assertion: assertion,
			Passed:    false,
			Message:   fmt.Sprintf("Path %s not found", assertion.Field),
		}
	}

	valueStr := fmt.Sprintf("%v", value)

	if assertion.Operator == "equals" {
		if valueStr == assertion.Value {
			return AssertionResult{
				Assertion: assertion,
				Passed:    true,
				Message:   fmt.Sprintf("Value at %s is %s", assertion.Field, valueStr),
			}
		}
		return AssertionResult{
			Assertion: assertion,
			Passed:    false,
			Message:   fmt.Sprintf("Expected %s but got %s", assertion.Value, valueStr),
		}
	}

	if assertion.Operator == "contains" {
		if strings.Contains(valueStr, assertion.Value) {
			return AssertionResult{
				Assertion: assertion,
				Passed:    true,
				Message:   fmt.Sprintf("Value contains %s", assertion.Value),
			}
		}
		return AssertionResult{
			Assertion: assertion,
			Passed:    false,
			Message:   fmt.Sprintf("Value does not contain %s", assertion.Value),
		}
	}

	return AssertionResult{
		Assertion: assertion,
		Passed:    false,
		Message:   "Invalid operator for jsonpath",
	}
}

const maxRegexPatternLength = 4096

func validateRegex(assertion Assertion, response *Response) AssertionResult {
	pattern := assertion.Value

	if len(pattern) > maxRegexPatternLength {
		return AssertionResult{
			Assertion: assertion,
			Passed:    false,
			Message:   "Regex pattern too long (max 4KB)",
		}
	}

	if len(response.Body) > 1024*1024 {
		return AssertionResult{
			Assertion: assertion,
			Passed:    false,
			Message:   "Response body too large for regex matching (max 1MB)",
		}
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return AssertionResult{
			Assertion: assertion,
			Passed:    false,
			Message:   "Invalid regex pattern",
		}
	}

	if assertion.Operator == "matches" {
		if re.MatchString(response.Body) {
			return AssertionResult{
				Assertion: assertion,
				Passed:    true,
				Message:   "Body matches regex pattern",
			}
		}
		return AssertionResult{
			Assertion: assertion,
			Passed:    false,
			Message:   "Body does not match regex pattern",
		}
	}

	return AssertionResult{
		Assertion: assertion,
		Passed:    false,
		Message:   "Invalid operator for regex",
	}
}

func validateHeader(assertion Assertion, response *Response) AssertionResult {
	if response.Headers == nil {
		return AssertionResult{
			Assertion: assertion,
			Passed:    false,
			Message:   "Response has no headers",
		}
	}

	value, exists := response.Headers[assertion.Field]
	if !exists {
		return AssertionResult{
			Assertion: assertion,
			Passed:    false,
			Message:   fmt.Sprintf("Header %s not found", assertion.Field),
		}
	}

	if assertion.Operator == "equals" {
		if value == assertion.Value {
			return AssertionResult{
				Assertion: assertion,
				Passed:    true,
				Message:   fmt.Sprintf("Header %s is %s", assertion.Field, value),
			}
		}
		return AssertionResult{
			Assertion: assertion,
			Passed:    false,
			Message:   fmt.Sprintf("Expected %s but got %s", assertion.Value, value),
		}
	}

	if assertion.Operator == "contains" {
		if strings.Contains(value, assertion.Value) {
			return AssertionResult{
				Assertion: assertion,
				Passed:    true,
				Message:   fmt.Sprintf("Header contains %s", assertion.Value),
			}
		}
		return AssertionResult{
			Assertion: assertion,
			Passed:    false,
			Message:   fmt.Sprintf("Header does not contain %s", assertion.Value),
		}
	}

	return AssertionResult{
		Assertion: assertion,
		Passed:    false,
		Message:   "Invalid operator for header",
	}
}

func extractJSONPath(data interface{}, path string) interface{} {
	if path == "" {
		return nil
	}

	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		if current == nil {
			return nil
		}

		switch v := current.(type) {
		case map[string]interface{}:
			current = v[part]
		default:
			return nil
		}
	}

	return current
}
