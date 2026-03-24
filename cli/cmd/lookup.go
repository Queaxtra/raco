package cmd

import (
	"fmt"
	"raco/model"
	"strconv"
)

// resolveRequestRef resolves a request by index, exact ID, or exact name.
// This keeps lookup rules consistent across collection preview and CRUD commands.
func resolveRequestRef(col *model.Collection, ref string) (int, *model.Request, error) {
	if col == nil {
		return -1, nil, fmt.Errorf("collection is nil")
	}
	if idx, err := strconv.Atoi(ref); err == nil {
		if idx < 0 || idx >= len(col.Requests) {
			return -1, nil, fmt.Errorf("request index out of range: %d", idx)
		}
		return idx, col.Requests[idx], nil
	}

	for idx, req := range col.Requests {
		if req != nil && req.ID == ref {
			return idx, req, nil
		}
	}

	matchIndex := -1
	var match *model.Request
	for idx, req := range col.Requests {
		if req != nil && req.Name == ref {
			if match != nil {
				return -1, nil, fmt.Errorf("request reference is ambiguous: %s", ref)
			}
			matchIndex = idx
			match = req
		}
	}

	if match == nil {
		return -1, nil, fmt.Errorf("request not found: %s", ref)
	}
	return matchIndex, match, nil
}
