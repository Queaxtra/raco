package command

import (
	"context"
	"raco/protocol"
	"raco/ui/notification"

	tea "github.com/charmbracelet/bubbletea"
)

type StreamConnectedMsg struct {
	Success bool
	Error   string
}

type StreamMessageReceivedMsg struct {
	Message protocol.Message
}

type StreamDisconnectedMsg struct {
	Error string
}

type StreamClosedMsg struct {
	Error string
}

func ConnectStream(client protocol.StreamHandler) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return StreamConnectedMsg{Success: false, Error: "No stream client initialized"}
		}

		err := client.Connect(context.Background())
		if err != nil {
			return StreamConnectedMsg{Success: false, Error: err.Error()}
		}

		return StreamConnectedMsg{Success: true}
	}
}

func ListenStream(client protocol.StreamHandler) tea.Cmd {
	return func() tea.Msg {
		msgChan, err := client.Receive()
		if err != nil {
			return StreamDisconnectedMsg{Error: err.Error()}
		}

		msg, ok := <-msgChan
		if !ok {
			return StreamDisconnectedMsg{}
		}

		return StreamMessageReceivedMsg{Message: msg}
	}
}

func DisconnectStream(client protocol.StreamHandler) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return StreamClosedMsg{}
		}

		if err := client.Close(); err != nil {
			return StreamClosedMsg{Error: err.Error()}
		}

		return StreamClosedMsg{}
	}
}

func SendStreamMessage(client protocol.StreamHandler, data string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return notification.ShowCmd("Not connected")()
		}

		err := client.Send(data)
		if err != nil {
			return notification.ShowCmd("Send failed: " + err.Error())()
		}

		return nil
	}
}
