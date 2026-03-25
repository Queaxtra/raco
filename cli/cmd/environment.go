package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"raco/model"
	"raco/secretstore"
	"raco/storage"
	"raco/util"
	"sort"
	"strings"

	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

// RunEnvironment owns both plain and secret-backed environment workflows.
// Secret values live outside YAML so environment files stay commit-friendly.
func RunEnvironment(ctx *Context, args []string) int {
	if len(args) == 0 {
		printEnvironmentUsage()
		return 1
	}

	store := ctx.Storage()
	action := args[0]
	subArgs := args[1:]

	switch action {
	case "list", "ls":
		return environmentList(ctx.StoragePath)
	case "show":
		return environmentShow(store, subArgs)
	case "get":
		return environmentGet(store, subArgs)
	case "create", "new":
		return environmentCreate(store, subArgs)
	case "delete", "rm":
		return environmentDelete(ctx.StoragePath, subArgs)
	case "set":
		return environmentSet(store, subArgs)
	case "set-secret":
		return environmentSetSecret(store, subArgs)
	case "set-parent":
		return environmentSetParent(store, subArgs)
	case "health":
		return environmentHealth(ctx, store, subArgs)
	case "rotate-secret":
		return environmentRotateSecret(store, subArgs)
	case "unset":
		return environmentUnset(store, subArgs)
	case "list-secrets":
		return environmentListSecrets(store, subArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown action: %s\n", action)
		printEnvironmentUsage()
		return 1
	}
}

func printEnvironmentUsage() {
	fmt.Println(`Usage: raco env <action> [options]

Actions:
  list, ls                List all environments
  show <name>             Show environment metadata
  get <name> <key>        Get variable or masked secret
  create, new <name>      Create new environment
  delete, rm <name>       Delete environment
  set <name> <key=val>    Set plain variable in environment
  set-secret <name> <key> Set secret value in environment
  set-parent <n> <p>      Set parent environment inheritance
  health [name]           Show environment health details
  rotate-secret <n> <k>   Rotate a secret value in secure storage
  unset <name> <key>      Remove variable from environment
  list-secrets <name>     List secret keys in environment

Examples:
  raco env list
  raco env create production
  raco env set production API_URL=https://api.prod.example.org
  raco env set-secret production API_TOKEN
  raco env get production API_TOKEN
  raco env get production API_TOKEN --reveal
  raco env unset production API_TOKEN`)
}

func environmentList(storagePath string) int {
	envPath := storagePath + "/environments"
	entries, err := os.ReadDir(envPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No environments found")
			return 0
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if len(entries) == 0 {
		fmt.Println("No environments found")
		return 0
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			name := strings.TrimSuffix(entry.Name(), ".yaml")
			fmt.Println(name)
		}
	}

	return 0
}

// environmentShow prints stored metadata only and does not resolve secret values.
func environmentShow(store *storage.Storage, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: environment name is required")
		return 1
	}

	env, err := store.LoadEnvironment(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	data, _ := yaml.Marshal(env)
	fmt.Print(string(data))
	return 0
}

func environmentCreate(store *storage.Storage, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: environment name is required")
		return 1
	}

	name := args[0]
	env := &model.Environment{
		Name:      name,
		Variables: make(map[string]model.EnvironmentVariable),
	}

	if err := store.SaveEnvironment(env); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Printf("Created environment: %s\n", name)
	return 0
}

func environmentDelete(storagePath string, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: environment name is required")
		return 1
	}

	name := args[0]
	if !isValidEnvName(name) {
		fmt.Fprintln(os.Stderr, "Error: invalid environment name format")
		return 1
	}

	path := filepath.Join(storagePath, "environments", name+".yaml")

	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		resolvedPath = path
	}

	expectedDir := filepath.Join(storagePath, "environments")
	resolvedDir, dirErr := filepath.EvalSymlinks(expectedDir)
	if dirErr == nil {
		expectedDir = resolvedDir
	}
	if !util.IsPathContained(resolvedPath, expectedDir) {
		fmt.Fprintln(os.Stderr, "Error: invalid path")
		return 1
	}

	if err := os.Remove(resolvedPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if secretBackend, err := secretstore.NewDefault(); err == nil {
		if secrets, listErr := secretBackend.List(name); listErr == nil {
			for _, key := range secrets {
				_ = secretBackend.Delete(name, key)
			}
		}
	}

	fmt.Printf("Deleted environment: %s\n", name)
	return 0
}

func isValidEnvName(name string) bool {
	if len(name) == 0 || len(name) > 64 {
		return false
	}

	for i, r := range name {
		isLower := r >= 'a' && r <= 'z'
		isUpper := r >= 'A' && r <= 'Z'
		isDigit := r >= '0' && r <= '9'
		isSpecial := r == '_' || r == '-'

		if !isLower && !isUpper && !isDigit && !isSpecial {
			return false
		}

		if i == 0 && !isLower && !isUpper && !isDigit {
			return false
		}
	}

	return true
}

// environmentSet updates only plaintext variables.
// Secret values must go through set-secret so they never reach the YAML payload.
func environmentSet(store *storage.Storage, args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: raco env set <name> <key=value>")
		return 1
	}

	name := args[0]
	env, err := store.LoadEnvironment(name)
	if err != nil {
		if os.IsNotExist(err) {
			env = &model.Environment{
				Name:      name,
				Variables: make(map[string]model.EnvironmentVariable),
			}
		}
		if env == nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
	}

	for _, pair := range args[1:] {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "Invalid format: %s (use key=value)\n", pair)
			return 1
		}
		if !util.ValidateEnvironmentKey(parts[0]) {
			fmt.Fprintf(os.Stderr, "Invalid environment key: %s\n", parts[0])
			return 1
		}
		env.Variables[parts[0]] = model.EnvironmentVariable{
			Kind:  model.EnvironmentVariablePlain,
			Value: parts[1],
		}
	}

	if err := store.SaveEnvironment(env); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Printf("Updated environment: %s\n", name)
	return 0
}

// environmentSetSecret requires a TTY and fails closed if secure storage is unavailable.
func environmentSetSecret(store *storage.Storage, args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: raco env set-secret <name> <key>")
		return 1
	}

	name := args[0]
	key := args[1]
	if !util.ValidateEnvironmentKey(key) {
		fmt.Fprintf(os.Stderr, "Invalid environment key: %s\n", key)
		return 1
	}

	secretBackend, err := secretstore.NewDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	info, err := os.Stdin.Stat()
	if err != nil || (info.Mode()&os.ModeCharDevice) == 0 {
		fmt.Fprintln(os.Stderr, "Error: set-secret requires a TTY")
		return 1
	}

	fmt.Printf("Enter secret for %s/%s: ", name, key)
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

	env, err := store.LoadEnvironment(name)
	if err != nil {
		if os.IsNotExist(err) {
			env = &model.Environment{Name: name, Variables: make(map[string]model.EnvironmentVariable)}
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
	}

	if err := secretBackend.Set(name, key, secretValue); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	env.Variables[key] = model.EnvironmentVariable{
		Kind: model.EnvironmentVariableSecret,
		Ref:  "raco/" + name + "/" + key,
	}

	if err := store.SaveEnvironment(env); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Printf("Stored secret: %s/%s\n", name, key)
	return 0
}

// environmentGet hides secret values unless the caller explicitly opts in with --reveal.
func environmentGet(store *storage.Storage, args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: raco env get <name> <key> [--reveal]")
		return 1
	}

	name := args[0]
	key := args[1]
	reveal := len(args) > 2 && args[2] == "--reveal"

	env, err := store.LoadEnvironment(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	variable, ok := env.Variables[key]
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: key not found: %s\n", key)
		return 1
	}

	if !variable.IsSecret() {
		fmt.Println(variable.Value)
		return 0
	}

	if !reveal {
		fmt.Println("[REDACTED]")
		return 0
	}

	secretBackend, err := secretstore.NewDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	value, err := secretBackend.Get(name, key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Println(value)
	return 0
}

// environmentUnset removes both the metadata entry and any backing secret value.
func environmentUnset(store *storage.Storage, args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: raco env unset <name> <key>")
		return 1
	}

	name := args[0]
	env, err := store.LoadEnvironment(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	secretBackend, _ := secretstore.NewDefault()
	for _, key := range args[1:] {
		if variable, ok := env.Variables[key]; ok && variable.IsSecret() && secretBackend != nil {
			_ = secretBackend.Delete(name, key)
		}
		delete(env.Variables, key)
	}

	if err := store.SaveEnvironment(env); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Printf("Updated environment: %s\n", name)
	return 0
}

func environmentListSecrets(store *storage.Storage, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: raco env list-secrets <name>")
		return 1
	}

	env, err := store.LoadEnvironment(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	keys := make([]string, 0)
	for key, variable := range env.Variables {
		if variable.IsSecret() {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Println(key)
	}
	return 0
}
