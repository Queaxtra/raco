package secretstore

import (
	"errors"
	"fmt"
	"os"
	"runtime"
)

var ErrUnsupported = errors.New("secret store is not supported on this system")

// Store is the minimum contract needed to keep secret values out of environment YAML files.
type Store interface {
	Set(envName string, key string, value string) error
	Get(envName string, key string) (string, error)
	Delete(envName string, key string) error
	List(envName string) ([]string, error)
}

// NewDefault picks the safest supported backend for the current machine.
// Callers must handle the explicit unsupported case instead of falling back to plaintext.
func NewDefault() (Store, error) {
	if backend := os.Getenv("RACO_SECRET_BACKEND"); backend != "" {
		switch backend {
		case "fake":
			return NewFakeStore(), nil
		default:
			return nil, fmt.Errorf("unsupported secret backend: %s", backend)
		}
	}

	switch runtime.GOOS {
	case "darwin":
		return &commandStore{
			getCmd:  "security",
			listCmd: "security",
			setter: func(service string, account string, value string) commandSpec {
				return commandSpec{Name: "security", Args: []string{"add-generic-password", "-U", "-s", service, "-a", account, "-w", value}}
			},
			getter: func(service string, account string) commandSpec {
				return commandSpec{Name: "security", Args: []string{"find-generic-password", "-s", service, "-a", account, "-w"}}
			},
			deleter: func(service string, account string) commandSpec {
				return commandSpec{Name: "security", Args: []string{"delete-generic-password", "-s", service, "-a", account}}
			},
			lister: func(servicePrefix string) commandSpec {
				return commandSpec{Name: "security", Args: []string{"dump-keychain"}}
			},
			parseList: parseDarwinList,
		}, nil
	case "linux":
		return &commandStore{
			getCmd:  "secret-tool",
			listCmd: "secret-tool",
			setter: func(service string, account string, value string) commandSpec {
				return commandSpec{Name: "secret-tool", Args: []string{"store", "--label=" + service, "service", service, "account", account}, Stdin: value}
			},
			getter: func(service string, account string) commandSpec {
				return commandSpec{Name: "secret-tool", Args: []string{"lookup", "service", service, "account", account}}
			},
			deleter: func(service string, account string) commandSpec {
				return commandSpec{Name: "secret-tool", Args: []string{"clear", "service", service, "account", account}}
			},
			lister: func(servicePrefix string) commandSpec {
				return commandSpec{Name: "secret-tool", Args: []string{"search", "--all", "service-prefix", servicePrefix}}
			},
			parseList: parseLinuxList,
		}, nil
	default:
		return nil, ErrUnsupported
	}
}
