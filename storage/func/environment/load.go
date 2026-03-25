package environment

import (
	"errors"
	"os"
	"path/filepath"
	"raco/model"
	"raco/secretstore"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var validEnvNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

const maxEnvironmentInheritanceDepth = 8

// Load reads environment metadata and transparently upgrades the legacy YAML shape in memory.
func Load(basePath string, name string) (*model.Environment, error) {
	if !validEnvNamePattern.MatchString(name) {
		return nil, errors.New("invalid environment name format")
	}

	path := filepath.Join(basePath, "environments", name+".yaml")
	expectedDir := filepath.Join(basePath, "environments")
	if resolvedExpectedDir, resolveErr := filepath.EvalSymlinks(expectedDir); resolveErr == nil {
		expectedDir = resolvedExpectedDir
	}

	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		resolvedPath = filepath.Join(expectedDir, filepath.Base(path))
	}

	if !isPathContained(resolvedPath, expectedDir) {
		return nil, errors.New("path traversal detected")
	}

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, err
	}

	var env model.Environment
	if err := yaml.Unmarshal(data, &env); err != nil {
		legacy, legacyErr := loadLegacyEnvironment(data)
		if legacyErr != nil {
			return nil, err
		}
		legacy.Name = name
		model.NormalizeEnvironment(legacy)
		return legacy, nil
	}

	if env.Name == "" {
		env.Name = name
	}
	model.NormalizeEnvironment(&env)

	return &env, nil
}

// Resolve loads metadata and then resolves secret-backed values through the secret store.
// The function fails instead of silently degrading when a declared secret cannot be resolved.
func Resolve(basePath string, name string, store secretstore.Store) (*model.ResolvedEnvironment, error) {
	env, err := LoadMerged(basePath, name)
	if err != nil {
		return nil, err
	}

	resolved := &model.ResolvedEnvironment{
		Name:       env.Name,
		Variables:  make(map[string]string, len(env.Variables)),
		SecretKeys: make(map[string]struct{}),
	}

	for key, variable := range env.Variables {
		switch variable.Kind {
		case model.EnvironmentVariableSecret:
			if store == nil {
				return nil, errors.New("secret store is required to resolve secrets")
			}
			value, err := store.Get(env.Name, key)
			if err != nil {
				return nil, err
			}
			resolved.Variables[key] = value
			resolved.SecretKeys[key] = struct{}{}
			resolved.SecretPlain = append(resolved.SecretPlain, value)
		default:
			resolved.Variables[key] = variable.Value
		}
	}

	return resolved, nil
}

func LoadMerged(basePath string, name string) (*model.Environment, error) {
	visited := make(map[string]struct{})
	return loadMerged(basePath, name, visited, 0)
}

// loadLegacyEnvironment maps the older flat YAML format into the structured environment model.
func loadLegacyEnvironment(data []byte) (*model.Environment, error) {
	var raw struct {
		Name      string            `yaml:"name"`
		Variables map[string]string `yaml:"variables"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	env := &model.Environment{
		Name:      raw.Name,
		Variables: make(map[string]model.EnvironmentVariable, len(raw.Variables)),
	}
	for key, value := range raw.Variables {
		env.Variables[key] = model.EnvironmentVariable{
			Kind:  model.EnvironmentVariablePlain,
			Value: value,
		}
	}
	return env, nil
}

func loadMerged(basePath string, name string, visited map[string]struct{}, depth int) (*model.Environment, error) {
	if depth > maxEnvironmentInheritanceDepth {
		return nil, errors.New("environment inheritance depth exceeded")
	}
	if _, ok := visited[name]; ok {
		return nil, errors.New("environment inheritance cycle detected")
	}
	visited[name] = struct{}{}

	env, err := Load(basePath, name)
	if err != nil {
		return nil, err
	}
	if env.Parent == "" {
		return env, nil
	}

	parent, err := loadMerged(basePath, env.Parent, visited, depth+1)
	if err != nil {
		return nil, err
	}

	merged := &model.Environment{
		Name:      env.Name,
		Parent:    env.Parent,
		Variables: make(map[string]model.EnvironmentVariable, len(parent.Variables)+len(env.Variables)),
	}
	for key, variable := range parent.Variables {
		merged.Variables[key] = variable
	}
	for key, variable := range env.Variables {
		merged.Variables[key] = variable
	}
	model.NormalizeEnvironment(merged)
	return merged, nil
}

// isPathContained ensures path is under base (prevents ".." traversal), not that path has no dots (e.g. .yaml).
func isPathContained(path, base string) bool {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return false
	}
	cleaned := filepath.Clean(rel)
	return cleaned != ".." && cleaned != "." && !strings.HasPrefix(cleaned, "..")
}
