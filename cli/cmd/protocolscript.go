package cmd

import (
	"bytes"
	"fmt"
	"os"
	"raco/model"
	"raco/protocol"
	"raco/util"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// loadProtocolScript validates both file location and YAML structure before the
// script is handed to a live stream connection.
func loadProtocolScript(path string) (*model.ProtocolScript, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	resolvedPath, err := util.ResolveContainedPath(cwd, path)
	if err != nil {
		return nil, err
	}
	data, err := util.ReadFileBounded(resolvedPath, 1024*1024)
	if err != nil {
		return nil, err
	}
	var script model.ProtocolScript
	// KnownFields prevents silent typo acceptance in production scripts.
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&script); err != nil {
		return nil, err
	}
	if len(script.Steps) == 0 {
		return nil, fmt.Errorf("script has no steps")
	}
	if len(script.Steps) > 256 {
		return nil, fmt.Errorf("script has too many steps")
	}
	return &script, nil
}

// runProtocolScript executes the script as a linear state machine. Keeping it
// sequential avoids ambiguity about which incoming message belongs to which step.
func runProtocolScript(client protocol.StreamHandler, path string) error {
	script, err := loadProtocolScript(path)
	if err != nil {
		return err
	}
	msgCh, err := client.Receive()
	if err != nil {
		return err
	}
	lastMessage := ""
	for _, step := range script.Steps {
		if step.Type == "send" {
			if err := client.Send(step.Send); err != nil {
				return err
			}
		}
		if step.Type == "wait" || step.Expect != "" || len(step.Assertions) > 0 {
			timeout := 5 * time.Second
			if step.TimeoutMS > 0 {
				timeout = time.Duration(step.TimeoutMS) * time.Millisecond
			}
			msg, err := awaitProtocolMessage(msgCh, timeout)
			if err != nil {
				return err
			}
			lastMessage = msg.Data
			if step.Expect != "" && !strings.Contains(lastMessage, step.Expect) {
				return fmt.Errorf("script expectation failed: %s", step.Expect)
			}
			for _, assertion := range step.Assertions {
				if assertion.Operator == "contains" && !strings.Contains(lastMessage, assertion.Value) {
					return fmt.Errorf("script assertion failed: %s", assertion.Value)
				}
				if assertion.Operator == "equals" && lastMessage != assertion.Value {
					return fmt.Errorf("script assertion failed: %s", assertion.Value)
				}
			}
		}
	}
	if lastMessage != "" {
		fmt.Println(lastMessage)
	}
	return nil
}

// awaitProtocolMessage keeps every waiting step bounded so broken peers cannot
// block the CLI forever.
func awaitProtocolMessage(msgCh <-chan protocol.Message, timeout time.Duration) (protocol.Message, error) {
	select {
	case msg, ok := <-msgCh:
		if !ok {
			return protocol.Message{}, fmt.Errorf("stream closed")
		}
		return msg, nil
	case <-time.After(timeout):
		return protocol.Message{}, fmt.Errorf("script timed out")
	}
}
