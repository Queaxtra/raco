package runner

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"raco/util"
)

// WriteReport stores collection run output in a CI-friendly format.
// The target path is cleaned and absolutized before any file is created.
func WriteReport(result *Result, path string, format string) error {
	cleanPath, err := resolveReportPath(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0750); err != nil {
		return err
	}
	file, err := os.OpenFile(cleanPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()

	switch format {
	case "json":
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	case "junit":
		return writeJUnit(writer, result)
	case "markdown", "github":
		return writeMarkdown(writer, result)
	case "sarif":
		return writeSARIF(writer, result)
	default:
		var buf bytes.Buffer
		PrintResultTo(&buf, result, "text")
		_, err := writer.Write(buf.Bytes())
		return err
	}
}

// writeMarkdown produces the same data summary as the text renderer but in a
// layout that works well in CI summaries and code review comments.
func writeMarkdown(writer *bufio.Writer, result *Result) error {
	if _, err := fmt.Fprintf(writer, "# Collection Run\n\n- Collection: %s\n- Environment: %s\n- Passed: %d\n- Failed: %d\n- Skipped: %d\n\n", result.CollectionName, result.EnvironmentName, result.PassedCount, result.FailedCount, result.SkippedCount); err != nil {
		return err
	}
	if _, err := writer.WriteString("| Status | Method | Name | Duration |\n|---|---|---|---|\n"); err != nil {
		return err
	}
	for _, req := range result.RequestResults {
		status := "PASS"
		if req.Skipped {
			status = "SKIP"
		}
		if !req.Passed && !req.Skipped {
			status = "FAIL"
		}
		if _, err := fmt.Fprintf(writer, "| %s | %s | %s | %dms |\n", status, req.Method, req.Name, req.Duration.Milliseconds()); err != nil {
			return err
		}
	}
	return nil
}

type sarifReport struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name string `json:"name"`
}

type sarifResult struct {
	RuleID  string            `json:"ruleId"`
	Level   string            `json:"level"`
	Message sarifResultText   `json:"message"`
	Props   map[string]string `json:"properties,omitempty"`
}

type sarifResultText struct {
	Text string `json:"text"`
}

func writeSARIF(writer *bufio.Writer, result *Result) error {
	report := sarifReport{
		Version: "2.1.0",
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Runs: []sarifRun{
			{
				Tool:    sarifTool{Driver: sarifDriver{Name: "raco"}},
				Results: make([]sarifResult, 0),
			},
		},
	}
	for _, req := range result.RequestResults {
		if req.Passed || req.Skipped {
			continue
		}
		report.Runs[0].Results = append(report.Runs[0].Results, sarifResult{
			RuleID: req.ErrorCategory,
			Level:  "error",
			Message: sarifResultText{
				Text: req.Name + ": " + req.ErrorMessage,
			},
			Props: map[string]string{
				"method": req.Method,
				"url":    req.URL,
			},
		})
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// resolveReportPath keeps report output inside the current workspace so CI jobs
// cannot be tricked into overwriting unrelated files.
func resolveReportPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("report path is required")
	}
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("absolute report paths are not allowed")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	cleanPath := filepath.Clean(path)
	resolvedBase, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		return "", err
	}
	resolvedPath, err := resolveReportTarget(resolvedBase, cleanPath)
	if err != nil {
		return "", err
	}
	if !util.IsPathContained(resolvedPath, resolvedBase) {
		return "", fmt.Errorf("report path escapes the workspace")
	}
	return resolvedPath, nil
}

// resolveReportTarget canonicalizes the deepest existing parent before the file
// is created, which makes symlink escapes visible during validation.
func resolveReportTarget(base string, relative string) (string, error) {
	target := filepath.Join(base, relative)
	dir := filepath.Dir(target)
	resolvedDir, err := resolveExistingDir(dir)
	if err != nil {
		return "", err
	}
	return filepath.Join(resolvedDir, filepath.Base(target)), nil
}

func resolveExistingDir(dir string) (string, error) {
	return util.ResolveExistingDir(dir)
}

// junitSuite models the subset of JUnit XML we need for collection run reporting.
type junitSuite struct {
	XMLName  xml.Name    `xml:"testsuite"`
	Name     string      `xml:"name,attr"`
	Tests    int         `xml:"tests,attr"`
	Failures int         `xml:"failures,attr"`
	Errors   int         `xml:"errors,attr"`
	Time     string      `xml:"time,attr"`
	Cases    []junitCase `xml:"testcase"`
}

type junitCase struct {
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
	Error     *junitFailure `xml:"error,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Body    string `xml:",chardata"`
}

// writeJUnit converts request results into a single suite.
// Runtime errors are emitted as JUnit errors while assertion problems become failures.
func writeJUnit(writer *bufio.Writer, result *Result) error {
	suite := junitSuite{
		Name:  result.CollectionName,
		Tests: result.SelectedCount,
		Time:  result.Duration.String(),
		Cases: make([]junitCase, 0, len(result.RequestResults)),
	}
	for _, req := range result.RequestResults {
		// Filter-skipped requests are omitted so JUnit totals describe the executed
		// suite rather than the user's broader collection inventory.
		if req.Skipped && req.SkipReason == "filtered out" {
			continue
		}
		testcase := junitCase{
			Name:      req.Name,
			ClassName: result.CollectionName,
			Time:      req.Duration.String(),
		}
		if req.ErrorCategory == "transport" || req.ErrorCategory == "no_assertions" {
			suite.Errors++
			testcase.Error = &junitFailure{Message: req.ErrorMessage, Body: req.ErrorMessage}
		} else if !req.Passed {
			suite.Failures++
			testcase.Failure = &junitFailure{Message: req.ErrorMessage, Body: req.ErrorMessage}
		}
		suite.Cases = append(suite.Cases, testcase)
	}
	encoder := xml.NewEncoder(writer)
	encoder.Indent("", "  ")
	return encoder.Encode(suite)
}
