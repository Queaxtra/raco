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
	AssertExists     AssertionType = "exists"
	AssertNotExists  AssertionType = "not_exists"
	AssertStartsWith AssertionType = "starts_with"
	AssertEndsWith   AssertionType = "ends_with"
	AssertGreater    AssertionType = "greater_than"
	AssertLess       AssertionType = "less_than"
	AssertJSONSchema AssertionType = "json_schema"
	AssertLatencyMS  AssertionType = "latency_ms"
	AssertSnapshot   AssertionType = "snapshot"
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
	if assertion.Type == AssertExists {
		return validateExists(assertion, response)
	}
	if assertion.Type == AssertNotExists {
		return validateNotExists(assertion, response)
	}
	if assertion.Type == AssertStartsWith {
		return validateStringAssertion(assertion, response, "starts_with")
	}
	if assertion.Type == AssertEndsWith {
		return validateStringAssertion(assertion, response, "ends_with")
	}
	if assertion.Type == AssertGreater {
		return validateNumericAssertion(assertion, response, "greater_than")
	}
	if assertion.Type == AssertLess {
		return validateNumericAssertion(assertion, response, "less_than")
	}
	if assertion.Type == AssertJSONSchema {
		return validateJSONSchema(assertion, response)
	}
	if assertion.Type == AssertLatencyMS {
		return validateLatency(assertion, response)
	}
	if assertion.Type == AssertSnapshot {
		return validateSnapshot(assertion, response)
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
	if assertion.Operator == "greater_than" || assertion.Operator == "less_than" {
		return compareNumericResult(assertion, int64(response.StatusCode), assertion.Operator, assertion.Value, "Status code")
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
	if assertion.Operator == "starts_with" {
		return compareStringResult(assertion, valueStr, "starts_with", assertion.Value, fmt.Sprintf("Value at %s", assertion.Field))
	}
	if assertion.Operator == "ends_with" {
		return compareStringResult(assertion, valueStr, "ends_with", assertion.Value, fmt.Sprintf("Value at %s", assertion.Field))
	}
	if assertion.Operator == "greater_than" || assertion.Operator == "less_than" {
		return compareNumericResult(assertion, parseNumericValue(valueStr), assertion.Operator, assertion.Value, fmt.Sprintf("Value at %s", assertion.Field))
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
	if assertion.Operator == "starts_with" {
		return compareStringResult(assertion, value, "starts_with", assertion.Value, fmt.Sprintf("Header %s", assertion.Field))
	}
	if assertion.Operator == "ends_with" {
		return compareStringResult(assertion, value, "ends_with", assertion.Value, fmt.Sprintf("Header %s", assertion.Field))
	}
	if assertion.Operator == "greater_than" || assertion.Operator == "less_than" {
		return compareNumericResult(assertion, parseNumericValue(value), assertion.Operator, assertion.Value, fmt.Sprintf("Header %s", assertion.Field))
	}

	return AssertionResult{
		Assertion: assertion,
		Passed:    false,
		Message:   "Invalid operator for header",
	}
}

func validateExists(assertion Assertion, response *Response) AssertionResult {
	if assertion.Field == "" {
		return AssertionResult{Assertion: assertion, Passed: false, Message: "exists requires field"}
	}
	if response.Headers != nil {
		if _, ok := response.Headers[assertion.Field]; ok {
			return AssertionResult{Assertion: assertion, Passed: true, Message: fmt.Sprintf("Field %s exists", assertion.Field)}
		}
	}
	if response.Body != "" {
		var data interface{}
		if err := json.Unmarshal([]byte(response.Body), &data); err == nil {
			if extractJSONPath(data, assertion.Field) != nil {
				return AssertionResult{Assertion: assertion, Passed: true, Message: fmt.Sprintf("Field %s exists", assertion.Field)}
			}
		}
	}
	return AssertionResult{Assertion: assertion, Passed: false, Message: fmt.Sprintf("Field %s does not exist", assertion.Field)}
}

func validateNotExists(assertion Assertion, response *Response) AssertionResult {
	result := validateExists(assertion, response)
	result.Passed = !result.Passed
	if result.Passed {
		result.Message = fmt.Sprintf("Field %s does not exist", assertion.Field)
	}
	if !result.Passed {
		result.Message = fmt.Sprintf("Field %s exists", assertion.Field)
	}
	return result
}

func validateStringAssertion(assertion Assertion, response *Response, mode string) AssertionResult {
	if assertion.Field == "" {
		return AssertionResult{Assertion: assertion, Passed: false, Message: "string assertion requires field"}
	}
	headerResult := validateHeader(Assertion{Type: AssertHeader, Field: assertion.Field, Operator: mode, Value: assertion.Value}, response)
	if headerResult.Passed {
		headerResult.Assertion = assertion
		return headerResult
	}
	return validateJSONPath(Assertion{Type: AssertJSONPath, Field: assertion.Field, Operator: mode, Value: assertion.Value}, response)
}

func validateNumericAssertion(assertion Assertion, response *Response, mode string) AssertionResult {
	if assertion.Field == "" {
		return AssertionResult{Assertion: assertion, Passed: false, Message: "numeric assertion requires field"}
	}
	headerResult := validateHeader(Assertion{Type: AssertHeader, Field: assertion.Field, Operator: mode, Value: assertion.Value}, response)
	if headerResult.Passed {
		headerResult.Assertion = assertion
		return headerResult
	}
	return validateJSONPath(Assertion{Type: AssertJSONPath, Field: assertion.Field, Operator: mode, Value: assertion.Value}, response)
}

func validateJSONSchema(assertion Assertion, response *Response) AssertionResult {
	if response.Body == "" {
		return AssertionResult{Assertion: assertion, Passed: false, Message: "Response body is empty"}
	}
	var body interface{}
	if err := json.Unmarshal([]byte(response.Body), &body); err != nil {
		return AssertionResult{Assertion: assertion, Passed: false, Message: "Response body is not valid JSON"}
	}
	var schema struct {
		Type       string                 `json:"type"`
		Required   []string               `json:"required"`
		Properties map[string]interface{} `json:"properties"`
	}
	if err := json.Unmarshal([]byte(assertion.Value), &schema); err != nil {
		return AssertionResult{Assertion: assertion, Passed: false, Message: "Invalid JSON schema"}
	}
	objectBody, ok := body.(map[string]interface{})
	if !ok {
		return AssertionResult{Assertion: assertion, Passed: false, Message: "Body is not a JSON object"}
	}
	for _, key := range schema.Required {
		if _, ok := objectBody[key]; !ok {
			return AssertionResult{Assertion: assertion, Passed: false, Message: fmt.Sprintf("Required key missing: %s", key)}
		}
	}
	return AssertionResult{Assertion: assertion, Passed: true, Message: "JSON schema matched"}
}

func validateLatency(assertion Assertion, response *Response) AssertionResult {
	return compareNumericResult(assertion, response.Duration.Milliseconds(), assertion.Operator, assertion.Value, "Latency")
}

func validateSnapshot(assertion Assertion, response *Response) AssertionResult {
	if response.Body == assertion.Value {
		return AssertionResult{Assertion: assertion, Passed: true, Message: "Snapshot matched"}
	}
	return AssertionResult{Assertion: assertion, Passed: false, Message: "Snapshot mismatch"}
}

func compareStringResult(assertion Assertion, actual string, operator string, expected string, label string) AssertionResult {
	if operator == "starts_with" {
		if strings.HasPrefix(actual, expected) {
			return AssertionResult{Assertion: assertion, Passed: true, Message: fmt.Sprintf("%s starts with %s", label, expected)}
		}
		return AssertionResult{Assertion: assertion, Passed: false, Message: fmt.Sprintf("%s does not start with %s", label, expected)}
	}
	if operator == "ends_with" {
		if strings.HasSuffix(actual, expected) {
			return AssertionResult{Assertion: assertion, Passed: true, Message: fmt.Sprintf("%s ends with %s", label, expected)}
		}
		return AssertionResult{Assertion: assertion, Passed: false, Message: fmt.Sprintf("%s does not end with %s", label, expected)}
	}
	return AssertionResult{Assertion: assertion, Passed: false, Message: "Invalid string operator"}
}

func compareNumericResult(assertion Assertion, actual int64, operator string, expected string, label string) AssertionResult {
	expectedNumber, err := strconv.ParseInt(strings.TrimSpace(expected), 10, 64)
	if err != nil {
		return AssertionResult{Assertion: assertion, Passed: false, Message: "Invalid numeric expectation"}
	}
	if operator == "greater_than" {
		if actual > expectedNumber {
			return AssertionResult{Assertion: assertion, Passed: true, Message: fmt.Sprintf("%s is greater than %d", label, expectedNumber)}
		}
		return AssertionResult{Assertion: assertion, Passed: false, Message: fmt.Sprintf("%s is not greater than %d", label, expectedNumber)}
	}
	if operator == "less_than" {
		if actual < expectedNumber {
			return AssertionResult{Assertion: assertion, Passed: true, Message: fmt.Sprintf("%s is less than %d", label, expectedNumber)}
		}
		return AssertionResult{Assertion: assertion, Passed: false, Message: fmt.Sprintf("%s is not less than %d", label, expectedNumber)}
	}
	return AssertionResult{Assertion: assertion, Passed: false, Message: "Invalid numeric operator"}
}

func parseNumericValue(value string) int64 {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0
	}
	return parsed
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
