package helper

import "raco/model"

func TotalSidebarItems(collections []*model.Collection, expandedIndex int, history []*model.HistoryEntry, historyExpanded bool) int {
	if collections == nil {
		collections = []*model.Collection{}
	}
	if history == nil {
		history = []*model.HistoryEntry{}
	}

	total := 0
	for _, col := range collections {
		if col == nil {
			continue
		}
		total++
		if expandedIndex >= 0 && expandedIndex < len(collections) {
			if collections[expandedIndex] != nil && collections[expandedIndex].ID == col.ID {
				total += len(col.Requests)
			}
		}
	}

	total++
	if historyExpanded {
		total += len(history)
	}

	return total
}
