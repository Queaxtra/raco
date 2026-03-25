package cmd

import (
	"fmt"
	"os"
	"raco/config"
)

func RunConfig(ctx *Context, args []string) int {
	if len(args) == 0 {
		printConfigUsage()
		return 1
	}

	action := args[0]
	cfg, err := config.Load(ctx.StoragePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if action == "list" {
		values := config.List(cfg)
		for _, key := range config.SortedKeys(values) {
			fmt.Printf("%s=%s\n", key, values[key])
		}
		return 0
	}

	if action == "get" {
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: raco config get <key>")
			return 1
		}
		value, err := config.Get(cfg, args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		fmt.Println(value)
		return 0
	}

	if action == "set" {
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: raco config set <key> <value>")
			return 1
		}
		if err := config.Set(&cfg, args[1], args[2], ctx.StoragePath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		if err := config.Save(ctx.StoragePath, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		fmt.Printf("Updated %s\n", args[1])
		return 0
	}

	if action == "reset" {
		if err := config.Reset(ctx.StoragePath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		fmt.Println("Config reset")
		return 0
	}

	fmt.Fprintf(os.Stderr, "Unknown action: %s\n", action)
	printConfigUsage()
	return 1
}

func printConfigUsage() {
	fmt.Println(`Usage: raco config <action> [options]

Actions:
  list                 List config values
  get <key>            Get a config value
  set <key> <value>    Set a config value
  reset                Reset config to defaults`)
}
