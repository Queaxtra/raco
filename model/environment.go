package model

type EnvironmentVariableKind string

const (
	EnvironmentVariablePlain  EnvironmentVariableKind = "plain"
	EnvironmentVariableSecret EnvironmentVariableKind = "secret"
)

type EnvironmentVariable struct {
	Kind  EnvironmentVariableKind `json:"kind" yaml:"kind"`
	Value string                  `json:"value,omitempty" yaml:"value,omitempty"`
	Ref   string                  `json:"ref,omitempty" yaml:"ref,omitempty"`
}

// IsSecret reports whether the variable must be resolved through the secret backend.
func (v EnvironmentVariable) IsSecret() bool {
	return v.Kind == EnvironmentVariableSecret
}

type Environment struct {
	Name      string                         `json:"name" yaml:"name"`
	Parent    string                         `json:"parent,omitempty" yaml:"parent,omitempty"`
	Variables map[string]EnvironmentVariable `json:"variables" yaml:"variables"`
}

// GetVariable returns the inline value stored in the environment metadata.
// Secret values are resolved only in ResolvedEnvironment.
func (e *Environment) GetVariable(key string) string {
	if e == nil || e.Variables == nil {
		return ""
	}
	variable, ok := e.Variables[key]
	if !ok {
		return ""
	}
	return variable.Value
}

// GetVariables exposes a copy of inline variables to keep callers from mutating stored state.
func (e *Environment) GetVariables() map[string]string {
	if e == nil || e.Variables == nil {
		return make(map[string]string)
	}

	out := make(map[string]string, len(e.Variables))
	for key, variable := range e.Variables {
		out[key] = variable.Value
	}
	return out
}

// ResolvedEnvironment is the runtime-only environment created after secret resolution.
type ResolvedEnvironment struct {
	Name        string
	Variables   map[string]string
	SecretKeys  map[string]struct{}
	SecretPlain []string
}

// GetVariable returns the fully resolved runtime value for a single key.
func (e *ResolvedEnvironment) GetVariable(key string) string {
	if e == nil || e.Variables == nil {
		return ""
	}
	return e.Variables[key]
}

// GetVariables returns a copy so runtime state cannot be mutated accidentally by callers.
func (e *ResolvedEnvironment) GetVariables() map[string]string {
	if e == nil || e.Variables == nil {
		return make(map[string]string)
	}
	out := make(map[string]string, len(e.Variables))
	for key, value := range e.Variables {
		out[key] = value
	}
	return out
}

// SecretValues returns a copy of raw secret values that can be used for redaction only.
func (e *ResolvedEnvironment) SecretValues() []string {
	if e == nil || len(e.SecretPlain) == 0 {
		return nil
	}
	out := make([]string, len(e.SecretPlain))
	copy(out, e.SecretPlain)
	return out
}

// IsSecretKey reports whether the key came from the secret backend.
func (e *ResolvedEnvironment) IsSecretKey(key string) bool {
	if e == nil || e.SecretKeys == nil {
		return false
	}
	_, ok := e.SecretKeys[key]
	return ok
}
