package cmd

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"raco/protocol"
	"raco/util"
	"strings"
	"syscall"
	"time"
)

func RunGRPC(ctx *Context, args []string) int {
	if len(args) > 0 && args[0] == "reflect" {
		return runGRPCReflect(args[1:])
	}
	if len(args) > 0 && args[0] == "scaffold" {
		return runGRPCScaffold(args[1:])
	}

	fs := flag.NewFlagSet("grpc", flag.ContinueOnError)
	address := fs.String("r", "", "gRPC server address (host:port)")
	insecure := fs.Bool("insecure", false, "Use insecure connection (no TLS)")
	scriptPath := fs.String("script", "", "Protocol script path")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if *address == "" {
		fmt.Fprintln(os.Stderr, "Error: Address is required (-r)")
		printGRPCUsage()
		return 1
	}
	if !*insecure && !util.ValidateGRPCTarget(*address) {
		fmt.Fprintln(os.Stderr, "Error: invalid gRPC target")
		return 1
	}
	if *insecure && !isLoopbackGRPCTarget(*address) {
		fmt.Fprintln(os.Stderr, "Error: insecure gRPC is only allowed for loopback targets")
		return 1
	}

	client := protocol.NewGRPCClient(*address)
	if setInsecure, ok := client.(interface{ SetInsecure(bool) }); ok {
		setInsecure.SetInsecure(*insecure)
	}
	connCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := client.Connect(connCtx); err != nil {
		fmt.Fprintf(os.Stderr, "Connection failed: %v\n", err)
		return 1
	}
	defer client.Close()

	if *scriptPath != "" {
		if err := runProtocolScript(client, *scriptPath); err != nil {
			fmt.Fprintf(os.Stderr, "Script failed: %v\n", err)
			return 1
		}
		return 0
	}

	fmt.Printf("Connected to gRPC server at %s\n", *address)
	fmt.Println("Send JSON envelope: {\"service\":\"pkg.Service\",\"method\":\"Method\",\"payload\":{...},\"metadata\":{...}}")
	fmt.Println("Ctrl+C to exit.")

	msgCh, err := client.Receive()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for msg := range msgCh {
			if msg.Direction == "received" {
				fmt.Printf("\n< [%s] %s\n> ", msg.Timestamp.Format(time.RFC3339), msg.Data)
			}
			if msg.Direction == "system" {
				fmt.Printf("\n[system] %s\n> ", msg.Data)
			}
			if msg.Type == "error" {
				fmt.Printf("\n! Error: %s\n", msg.Data)
				cancel()
				return
			}
		}
	}()

	inputCh := make(chan string)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("> ")
		for scanner.Scan() {
			text := scanner.Text()
			if text != "" {
				inputCh <- text
			}
			fmt.Print("> ")
		}
	}()

	for {
		select {
		case <-sigCh:
			fmt.Println("\nDisconnecting...")
			return 0
		case <-connCtx.Done():
			return 1
		case text := <-inputCh:
			if err := client.Send(text); err != nil {
				fmt.Fprintf(os.Stderr, "Send error: %v\n", err)
			}
		}
	}
}

func isLoopbackGRPCTarget(target string) bool {
	host := target
	if strings.Contains(target, ":") {
		parts := strings.Split(target, ":")
		if len(parts) > 1 {
			host = strings.Join(parts[:len(parts)-1], ":")
		}
	}
	host = strings.Trim(host, "[]")
	lower := strings.ToLower(host)
	if lower == "localhost" || lower == "127.0.0.1" || lower == "::1" {
		return true
	}
	return false
}

func printGRPCUsage() {
	fmt.Println(`Usage: raco grpc [options]

Options:
  -r <address>   gRPC server address (host:port) (required)
  -insecure     Use insecure connection (no TLS, for localhost)
  --script      Run protocol script and exit

Send (stdin) JSON envelope per line, e.g.:
  {"service":"grpc.health.v1.Health","method":"Check","payload":{},"metadata":{}}

Examples:
  raco grpc -r localhost:50051
  raco grpc -r localhost:50051 -insecure
  raco grpc -r api.example.org:443
  raco grpc reflect -r api.example.org:443
  raco grpc scaffold --proto service.proto --service pkg.Service --method Check`)
}
