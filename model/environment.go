package model

type Environment struct {
	Name      string            `json:"name" yaml:"name"`
	Variables map[string]string `json:"variables" yaml:"variables"`
}

func (e *Environment) GetVariable(key string) string {
	if e.Variables == nil {
		return ""
	}
	return e.Variables[key]
}

func (e *Environment) GetVariables() map[string]string {
	if e.Variables == nil {
		return make(map[string]string)
	}
	return e.Variables
}
