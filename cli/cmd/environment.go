package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"raco/model"
	"raco/storage"
	"raco/util"
	"strings"

	"gopkg.in/yaml.v3"
)

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
	case "show", "get":
		return environmentShow(store, subArgs)
	case "create", "new":
		return environmentCreate(store, subArgs)
	case "delete", "rm":
		return environmentDelete(ctx.StoragePath, subArgs)
	case "set":
		return environmentSet(store, subArgs)
	case "unset":
		return environmentUnset(store, subArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown action: %s\n", action)
		printEnvironmentUsage()
		return 1
	}
}

func printEnvironmentUsage() {
	fmt.Println(`Usage: raco env <action> [options]

Actions:
  list, ls              List all environments
  show, get <name>      Show environment details
  create, new <name>    Create new environment
  delete, rm <name>     Delete environment
  set <name> <key=val>  Set variable in environment
  unset <name> <key>    Remove variable from environment

Examples:
  raco env list
  raco env create production
  raco env set production API_URL=https://api.prod.example.org
  raco env set production API_KEY=secret123
  raco env show production
  raco env unset production API_KEY`)
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
		Variables: make(map[string]string),
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
	if !util.IsPathContained(resolvedPath, expectedDir) {
		fmt.Fprintln(os.Stderr, "Error: invalid path")
		return 1
	}

	if err := os.Remove(resolvedPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
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
				Variables: make(map[string]string),
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
		env.Variables[parts[0]] = parts[1]
	}

	if err := store.SaveEnvironment(env); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Printf("Updated environment: %s\n", name)
	return 0
}

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

	for _, key := range args[1:] {
		delete(env.Variables, key)
	}

	if err := store.SaveEnvironment(env); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Printf("Updated environment: %s\n", name)
	return 0
}
