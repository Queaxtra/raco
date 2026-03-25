package cmd

import (
	"fmt"
	"os"
)

func RunCompletion(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: raco completion bash|zsh|fish")
		return 1
	}

	shell := args[0]
	if shell == "bash" {
		fmt.Println(`_raco_complete() { COMPREPLY=( $(compgen -W "request req ws websocket grpc collection col env environment import export curl run stats config doctor completion update help version" -- "${COMP_WORDS[1]}") ); } && complete -F _raco_complete raco`)
		return 0
	}
	if shell == "zsh" {
		fmt.Println(`#compdef raco
_raco() {
  local -a commands
  commands=(
    'request:Make HTTP request'
    'req:Make HTTP request'
    'ws:Connect to WebSocket server'
    'websocket:Connect to WebSocket server'
    'grpc:Connect to gRPC server'
    'collection:Manage collections'
    'col:Manage collections'
    'env:Manage environments'
    'environment:Manage environments'
    'import:Import collections'
    'export:Export collections'
    'curl:Parse or convert curl commands'
    'run:Run a collection'
    'stats:Show stats'
    'config:Manage config'
    'doctor:Run diagnostics'
    'completion:Shell completions'
    'update:Update raco'
    'help:Show help'
    'version:Show version'
  )
  _describe 'command' commands
}
_raco`)
		return 0
	}
	if shell == "fish" {
		fmt.Println(`complete -c raco -f -a "request req ws websocket grpc collection col env environment import export curl run stats config doctor completion update help version"`)
		return 0
	}

	fmt.Fprintf(os.Stderr, "Unknown shell: %s\n", shell)
	return 1
}
