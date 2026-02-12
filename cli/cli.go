package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"raco/cli/cmd"
)

func Run(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 1
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	ctx := &cmd.Context{
		StoragePath: filepath.Join(homeDir, ".raco"),
	}

	command := args[0]
	subArgs := args[1:]

	switch command {
	case "request", "req":
		return cmd.RunRequest(ctx, subArgs)
	case "ws", "websocket":
		return cmd.RunWebSocket(ctx, subArgs)
	case "grpc":
		return cmd.RunGRPC(ctx, subArgs)
	case "collection", "col":
		return cmd.RunCollection(ctx, subArgs)
	case "env", "environment":
		return cmd.RunEnvironment(ctx, subArgs)
	case "import":
		return cmd.RunImport(ctx, subArgs)
	case "curl":
		return cmd.RunCurl(ctx, subArgs)
	case "run":
		return cmd.RunRunner(ctx, subArgs)
	case "stats":
		return cmd.RunStats(ctx, subArgs)
	case "help", "-h", "--help":
		printUsage()
		return 0
	case "version", "-v", "--version":
		fmt.Println("raco v1.0.0")
		return 0
	default:
		if len(args) > 0 && (args[0] == "-r" || args[0] == "-m") {
			return cmd.RunRequest(ctx, args)
		}
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		return 1
	}
}

func printUsage() {
	fmt.Println(`Usage: raco <command> [options]

Commands:
  request, req     Make HTTP request
  ws, websocket    Connect to WebSocket server
  grpc             Connect to gRPC server
  collection, col  Manage collections
  env, environment Manage environments
  import           Import Postman collection
  curl             Parse/convert cURL commands
  run              Run collection with assertions
  stats            Show request statistics
  help             Show this help
  version          Show version

Examples:
  raco req -m GET -r https://api.example.org
  raco req -m POST -r https://api.example.org -d '{"key":"value"}'
  raco ws -r wss://echo.websocket.org
  raco grpc -r localhost:50051
  raco col list
  raco env list
  raco import postman collection.json
  raco curl parse 'curl -X GET https://api.example.org'
  raco run my-collection -e production
  raco stats`)
}
