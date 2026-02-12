package command

import (
	"raco/model"
	"raco/storage"

	tea "github.com/charmbracelet/bubbletea"
)

type CollectionsLoadedMsg struct {
	Collections []*model.Collection
}

func Load(storage *storage.Storage) tea.Cmd {
	return func() tea.Msg {
		collections, err := storage.ListCollections()
		if err != nil {
			collections = []*model.Collection{}
		}
		return CollectionsLoadedMsg{Collections: collections}
	}
}
