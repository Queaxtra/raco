package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"raco/model"
	"raco/secretstore"
	"raco/storage"
	"strings"

	"golang.org/x/term"
)

func environmentSetParent(store *storage.Storage, args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: raco env set-parent <name> <parent>")
		return 1
	}

	name := args[0]
	parent := strings.TrimSpace(args[1])
	if name == parent {
		fmt.Fprintln(os.Stderr, "Error: environment cannot inherit from itself")
		return 1
	}

	env, err := store.LoadEnvironment(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	if _, err := store.LoadEnvironment(parent); err != nil {
		fmt.Fprintf(os.Stderr, "Error: parent environment not found: %v\n", err)
		return 1
	}

	env.Parent = parent
	if err := store.SaveEnvironment(env); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	fmt.Printf("Set parent of %s to %s\n", name, parent)
	return 0
}

func environmentHealth(ctx *Context, store *storage.Storage, args []string) int {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}

	secretStatus := "unsupported"
	if _, err := secretstore.NewDefault(); err == nil {
		secretStatus = "available"
	}

	fmt.Printf("storage=%s\n", ctx.StoragePath)
	fmt.Printf("environments=%s\n", filepath.Join(ctx.StoragePath, "environments"))
	fmt.Printf("secret_backend=%s\n", secretStatus)
	if name == "" {
		return 0
	}

	env, err := store.LoadMergedEnvironment(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	fmt.Printf("name=%s\n", env.Name)
	fmt.Printf("parent=%s\n", env.Parent)
	fmt.Printf("variables=%d\n", len(env.Variables))
	return 0
}

func environmentRotateSecret(store *storage.Storage, args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: raco env rotate-secret <name> <key>")
		return 1
	}

	name := args[0]
	key := args[1]
	env, err := store.LoadEnvironment(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	variable, ok := env.Variables[key]
	if !ok || !variable.IsSecret() {
		fmt.Fprintf(os.Stderr, "Error: secret key not found: %s\n", key)
		return 1
	}

	secretBackend, err := secretstore.NewDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	info, err := os.Stdin.Stat()
	if err != nil || (info.Mode()&os.ModeCharDevice) == 0 {
		fmt.Fprintln(os.Stderr, "Error: rotate-secret requires a TTY")
		return 1
	}

	fmt.Printf("Enter new secret for %s/%s: ", name, key)
	secretBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	secretValue := strings.TrimSpace(string(secretBytes))
	if secretValue == "" {
		fmt.Fprintln(os.Stderr, "Error: secret value is required")
		return 1
	}

	if err := secretBackend.Set(name, key, secretValue); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	fmt.Printf("Rotated secret: %s/%s\n", name, key)
	return 0
}

func mergedEnvironmentModel(resolved *model.ResolvedEnvironment) *model.Environment {
	if resolved == nil {
		return nil
	}
	env := &model.Environment{
		Name:      resolved.Name,
		Variables: make(map[string]model.EnvironmentVariable, len(resolved.Variables)),
	}
	for key, value := range resolved.Variables {
		env.Variables[key] = model.EnvironmentVariable{
			Kind:  model.EnvironmentVariablePlain,
			Value: value,
		}
	}
	return env
}
