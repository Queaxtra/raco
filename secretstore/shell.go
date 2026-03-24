package secretstore

import (
	"bytes"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// commandSpec keeps shell execution explicit so secret store calls never rely on
// shell interpolation or concatenated command strings.
type commandSpec struct {
	Name  string
	Args  []string
	Stdin string
}

// commandStore adapts native OS secret tools behind the Store interface.
// It intentionally fails closed when the expected binary is unavailable.
type commandStore struct {
	getCmd    string
	listCmd   string
	setter    func(service string, account string, value string) commandSpec
	getter    func(service string, account string) commandSpec
	deleter   func(service string, account string) commandSpec
	lister    func(servicePrefix string) commandSpec
	parseList func(output string, servicePrefix string) []string
}

// secretService keeps secret records namespaced by environment to avoid collisions
// between multiple developer setups on the same machine.
func secretService(envName string) string {
	return "raco/" + envName
}

func secretAccount(key string) string {
	return key
}

// ensureAvailable checks the external dependency up front so callers get a clear,
// deterministic unsupported error instead of a partially executed flow.
func (s *commandStore) ensureAvailable(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return ErrUnsupported
	}
	return nil
}

// run executes the native secret command without invoking a shell.
// The returned error stays generic on purpose so secret values never leak through stderr.
func (s *commandStore) run(spec commandSpec) (string, error) {
	cmd := exec.Command(spec.Name, spec.Args...)
	if spec.Stdin != "" {
		cmd.Stdin = strings.NewReader(spec.Stdin)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("secret store command failed")
	}
	return strings.TrimSpace(stdout.String()), nil
}

// Set writes a secret to the native keychain implementation.
func (s *commandStore) Set(envName string, key string, value string) error {
	if err := s.ensureAvailable(s.getCmd); err != nil {
		return err
	}
	_, err := s.run(s.setter(secretService(envName), secretAccount(key), value))
	return err
}

// Get reads a secret from the native keychain implementation.
func (s *commandStore) Get(envName string, key string) (string, error) {
	if err := s.ensureAvailable(s.getCmd); err != nil {
		return "", err
	}
	return s.run(s.getter(secretService(envName), secretAccount(key)))
}

// Delete removes a secret from the native keychain implementation.
func (s *commandStore) Delete(envName string, key string) error {
	if err := s.ensureAvailable(s.getCmd); err != nil {
		return err
	}
	_, err := s.run(s.deleter(secretService(envName), secretAccount(key)))
	return err
}

// List returns known secret keys for an environment without exposing their values.
func (s *commandStore) List(envName string) ([]string, error) {
	if err := s.ensureAvailable(s.listCmd); err != nil {
		return nil, err
	}
	output, err := s.run(s.lister(secretService(envName)))
	if err != nil {
		return nil, err
	}
	items := s.parseList(output, secretService(envName))
	sort.Strings(items)
	return items, nil
}

// parseDarwinList extracts account names from `security dump-keychain` output.
func parseDarwinList(output string, servicePrefix string) []string {
	lines := strings.Split(output, "\n")
	results := make([]string, 0)
	var service string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "\"svce\"") {
			service = extractQuotedValue(line)
		}
		if service != servicePrefix {
			continue
		}
		if strings.Contains(line, "\"acct\"") {
			account := extractQuotedValue(line)
			if account != "" {
				results = append(results, account)
			}
		}
	}
	return dedupe(results)
}

// parseLinuxList extracts account names from `secret-tool search --all` output.
func parseLinuxList(output string, servicePrefix string) []string {
	lines := strings.Split(output, "\n")
	results := make([]string, 0)
	serviceMatches := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if serviceMatches {
				serviceMatches = false
			}
			continue
		}
		if strings.HasPrefix(line, "service = ") && strings.Contains(line, servicePrefix) {
			serviceMatches = true
			continue
		}
		if serviceMatches && strings.HasPrefix(line, "account = ") {
			value := strings.Trim(strings.TrimPrefix(line, "account = "), "\"")
			if value != "" {
				results = append(results, value)
			}
		}
	}
	return dedupe(results)
}

// extractQuotedValue parses the quoted field value from the native tool output.
func extractQuotedValue(line string) string {
	first := strings.IndexByte(line, '"')
	if first == -1 {
		return ""
	}
	second := strings.IndexByte(line[first+1:], '"')
	if second == -1 {
		return ""
	}
	rest := line[first+second+2:]
	valueStart := strings.IndexByte(rest, '"')
	if valueStart == -1 {
		return ""
	}
	valueRest := rest[valueStart+1:]
	valueEnd := strings.IndexByte(valueRest, '"')
	if valueEnd == -1 {
		return ""
	}
	return valueRest[:valueEnd]
}

// dedupe keeps list output stable when the underlying OS tool repeats entries.
func dedupe(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
