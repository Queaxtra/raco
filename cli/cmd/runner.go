package cmd

import (
	"flag"
	"fmt"
	"os"
	"raco/cli/runner"
)

func RunRunner(ctx *Context, args []string) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	env := fs.String("e", "", "Environment name")
	outputFmt := fs.String("o", "text", "Output format: text, json")
	stopOnFail := fs.Bool("stop-on-fail", false, "Stop on first failure")

	reorderedArgs := reorderArgs(args)

	if err := fs.Parse(reorderedArgs); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		printRunnerUsage()
		return 1
	}

	colID := remaining[0]

	store := ctx.Storage()
	col, err := store.LoadCollection(colID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading collection: %v\n", err)
		return 1
	}

	var environment runner.EnvironmentProvider
	if *env != "" {
		loadedEnv, err := store.LoadEnvironment(*env)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading environment: %v\n", err)
			return 1
		}
		environment = &envWrapper{env: loadedEnv}
	}

	cfg := &runner.Config{
		Collection:  col,
		Environment: environment,
		StopOnFail:  *stopOnFail,
		OutputFormat: *outputFmt,
	}

	result := runner.Execute(cfg)
	runner.PrintResult(result, *outputFmt)

	if result.FailedCount > 0 {
		return 1
	}

	return 0
}

func printRunnerUsage() {
	fmt.Println(`Usage: raco run <collection-id> [options]

Options:
  -e <env>         Environment name
  -o <format>      Output format: text, json
  --stop-on-fail   Stop on first failure

Examples:
  raco run my-api-tests
  raco run my-api-tests -e production
  raco run my-api-tests -e staging -o json
  raco run my-api-tests --stop-on-fail`)
}

type envWrapper struct {
	env interface {
		GetVariable(key string) string
		GetVariables() map[string]string
	}
}

func (e *envWrapper) GetVariable(key string) string {
	return e.env.GetVariable(key)
}

func (e *envWrapper) GetVariables() map[string]string {
	return e.env.GetVariables()
}

func reorderArgs(args []string) []string {
	var flags []string
	var positional []string

	skipNext := false
	for i, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}

		if len(arg) > 0 && arg[0] == '-' {
			flags = append(flags, arg)
			if arg == "-e" || arg == "-o" {
				if i+1 < len(args) {
					flags = append(flags, args[i+1])
					skipNext = true
				}
			}
			continue
		}

		positional = append(positional, arg)
	}

	return append(flags, positional...)
}
