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
	default:
		var buf bytes.Buffer
		PrintResultTo(&buf, result, "text")
		_, err := writer.Write(buf.Bytes())
		return err
	}
}

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
	current := dir
	visited := make([]string, 0, 4)
	for {
		visited = append(visited, current)
		if info, err := os.Stat(current); err == nil && info.IsDir() {
			resolved, err := filepath.EvalSymlinks(current)
			if err != nil {
				return "", err
			}
			for i := len(visited) - 2; i >= 0; i-- {
				resolved = filepath.Join(resolved, filepath.Base(visited[i]))
			}
			return resolved, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("report path base not found")
		}
		current = parent
	}
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
