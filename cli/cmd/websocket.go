package cmd

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"raco/protocol"
	"strings"
	"syscall"
	"time"
)

func RunWebSocket(ctx *Context, args []string) int {
	fs := flag.NewFlagSet("websocket", flag.ContinueOnError)
	url := fs.String("r", "", "WebSocket URL (ws:// or wss://)")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if *url == "" {
		fmt.Fprintln(os.Stderr, "Error: URL is required (-r)")
		return 1
	}

	if !strings.HasPrefix(*url, "ws://") && !strings.HasPrefix(*url, "wss://") {
		fmt.Fprintln(os.Stderr, "Error: URL must start with ws:// or wss://")
		return 1
	}

	client := protocol.NewWebSocketClient(*url)
	connCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := client.Connect(connCtx); err != nil {
		fmt.Fprintf(os.Stderr, "Connection failed: %v\n", err)
		return 1
	}
	defer client.Close()

	fmt.Printf("Connected to %s\n", *url)
	fmt.Println("Type messages and press Enter to send. Ctrl+C to exit.")

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
			if msg.Type == "error" && msg.Direction == "system" {
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
