package cmd

import (
	"flag"
	"fmt"
	"os"
	"raco/cli/runner"
	"raco/model"
	"raco/util/osnotify"
	"strings"
)

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

// RunRunner wires collection execution to filters, reporting, and resolved environments.
// Execution intentionally stays sequential because extractors can feed later requests.
func RunRunner(ctx *Context, args []string) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	envName := fs.String("e", "", "Environment name")
	outputFmt := fs.String("o", "text", "Output format: text, json")
	stopOnFail := fs.Bool("stop-on-fail", false, "Stop on first failure")
	reportPath := fs.String("report", "", "Write report to path")
	reportFormat := fs.String("report-format", "text", "Report format: text, json, junit")
	failIfNoTests := fs.Bool("fail-if-no-tests", false, "Fail requests with no assertions")
	maxParallel := fs.Int("max-parallel", 1, "Maximum parallel requests (currently only 1)")

	var requestRefs multiFlag
	var exactNames multiFlag
	var nameContains multiFlag
	var methods multiFlag
	fs.Var(&requestRefs, "request", "Request reference filter")
	fs.Var(&exactNames, "name", "Exact request name filter")
	fs.Var(&nameContains, "name-contains", "Substring request name filter")
	fs.Var(&methods, "method", "HTTP method filter")

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

	if *maxParallel != 1 {
		fmt.Fprintln(os.Stderr, "Error: only --max-parallel 1 is supported in this release")
		return 1
	}

	colID := remaining[0]

	store := ctx.Storage()
	col, err := store.LoadCollection(colID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading collection: %v\n", err)
		return 1
	}

	var resolvedModelEnv *model.ResolvedEnvironment
	if *envName != "" {
		resolvedModelEnv, err = ctx.ResolveEnvironment(*envName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading environment: %v\n", err)
			return 1
		}
	}

	cfg := &runner.Config{
		Collection:      col,
		Environment:     resolvedModelEnv,
		EnvironmentName: *envName,
		StopOnFail:      *stopOnFail,
		OutputFormat:    *outputFmt,
		FailIfNoTests:   *failIfNoTests,
		MaxParallel:     *maxParallel,
		RequestFilter: runner.RequestFilter{
			Refs:         requestRefs,
			ExactNames:   exactNames,
			NameContains: nameContains,
			Methods:      methods,
		},
	}

	result := runner.Execute(cfg)
	runner.PrintResult(result, *outputFmt)

	if *reportPath != "" {
		if err := runner.WriteReport(result, *reportPath, *reportFormat); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing report: %v\n", err)
			return 1
		}
	}

	msg := fmt.Sprintf("%s: %d passed, %d failed", result.CollectionName, result.PassedCount, result.FailedCount)
	if result.FailedCount > 0 {
		osnotify.Send("Raco", msg)
		return 1
	}
	osnotify.Send("Raco", msg)
	return 0
}

func printRunnerUsage() {
	fmt.Println(`Usage: raco run <collection-id> [options]

Options:
  -e <env>              Environment name
  -o <format>           Output format: text, json
  --stop-on-fail        Stop on first failure
  --request <ref>       Request ref filter (repeatable)
  --name <name>         Exact request name filter (repeatable)
  --name-contains <s>   Substring request name filter (repeatable)
  --method <method>     Method filter (repeatable)
  --report <path>       Write report to path
  --report-format <f>   Report format: text, json, junit
  --fail-if-no-tests    Fail requests without assertions
  --max-parallel 1      Reserved for future parallel execution

Examples:
  raco run my-api-tests
  raco run my-api-tests -e production
  raco run my-api-tests --request 0 --report result.xml --report-format junit
  raco run my-api-tests --name-contains users --method GET`)
}

// reorderArgs keeps the standard flag package usable even when users place flags after the collection ID.
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
			if arg == "-e" || arg == "-o" || arg == "--report" || arg == "--report-format" || arg == "--request" || arg == "--name" || arg == "--name-contains" || arg == "--method" || arg == "--max-parallel" {
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
