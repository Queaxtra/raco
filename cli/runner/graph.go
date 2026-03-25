package runner

import (
	"fmt"
	"raco/model"
	"regexp"
	"sort"
)

var placeholderPattern = regexp.MustCompile(`\{\{([A-Z0-9_]+)\}\}`)

type ExecutionPlan struct {
	OrderedIndices []int
	Dependencies   map[int][]int
}

func BuildExecutionPlan(col *model.Collection, filter RequestFilter) (*ExecutionPlan, error) {
	if col == nil {
		return nil, fmt.Errorf("collection is nil")
	}

	selected := make(map[int]struct{})
	for idx, req := range col.Requests {
		if requestMatches(req, idx, filter, col.Tags) {
			selected[idx] = struct{}{}
		}
	}

	extractorOwners := make(map[string]int)
	for idx, req := range col.Requests {
		if _, ok := selected[idx]; !ok {
			continue
		}
		if req == nil {
			continue
		}
		for _, extractor := range req.Extractors {
			if extractor.Target == "" {
				continue
			}
			if _, exists := extractorOwners[extractor.Target]; exists {
				continue
			}
			extractorOwners[extractor.Target] = idx
		}
	}

	deps := make(map[int][]int, len(selected))
	indegree := make(map[int]int, len(selected))
	for idx := range selected {
		indegree[idx] = 0
	}

	for idx, req := range col.Requests {
		if _, ok := selected[idx]; !ok {
			continue
		}
		keys := requestPlaceholderKeys(req)
		seen := make(map[int]struct{})
		for _, key := range keys {
			owner, ok := extractorOwners[key]
			if !ok || owner == idx {
				continue
			}
			if _, exists := seen[owner]; exists {
				continue
			}
			seen[owner] = struct{}{}
			deps[idx] = append(deps[idx], owner)
		}
		sort.Ints(deps[idx])
		indegree[idx] = len(deps[idx])
	}

	ready := make([]int, 0, len(selected))
	for idx := range selected {
		if indegree[idx] == 0 {
			ready = append(ready, idx)
		}
	}
	sort.Ints(ready)

	order := make([]int, 0, len(selected))
	for len(ready) > 0 {
		current := ready[0]
		ready = ready[1:]
		order = append(order, current)
		for idx := range selected {
			if idx == current {
				continue
			}
			for _, dep := range deps[idx] {
				if dep != current {
					continue
				}
				indegree[idx]--
				if indegree[idx] == 0 {
					ready = append(ready, idx)
					sort.Ints(ready)
				}
				break
			}
		}
	}

	if len(order) != len(selected) {
		return nil, fmt.Errorf("request dependency cycle detected")
	}

	return &ExecutionPlan{
		OrderedIndices: order,
		Dependencies:   deps,
	}, nil
}

func requestPlaceholderKeys(req *model.Request) []string {
	if req == nil {
		return nil
	}

	seen := make(map[string]struct{})
	add := func(text string) {
		matches := placeholderPattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			seen[match[1]] = struct{}{}
		}
	}

	add(req.URL)
	add(req.Body)
	for key, value := range req.Query {
		add(key)
		add(value)
	}
	for key, value := range req.Headers {
		add(key)
		add(value)
	}
	if req.BodyFile != "" {
		add(req.BodyFile)
	}

	out := make([]string, 0, len(seen))
	for key := range seen {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func ResolveRequestRef(col *model.Collection, ref string) (int, *model.Request, error) {
	if col == nil {
		return -1, nil, fmt.Errorf("collection is nil")
	}
	for idx, req := range col.Requests {
		if fmt.Sprintf("%d", idx) == ref {
			return idx, req, nil
		}
	}
	for idx, req := range col.Requests {
		if req != nil && req.ID == ref {
			return idx, req, nil
		}
	}
	for idx, req := range col.Requests {
		if req != nil && req.Name == ref {
			return idx, req, nil
		}
	}
	return -1, nil, fmt.Errorf("request not found: %s", ref)
}

func GraphLines(col *model.Collection, plan *ExecutionPlan) []string {
	if col == nil || plan == nil {
		return nil
	}
	out := make([]string, 0, len(plan.OrderedIndices))
	for _, idx := range plan.OrderedIndices {
		req := col.Requests[idx]
		deps := plan.Dependencies[idx]
		names := make([]string, 0, len(deps))
		for _, dep := range deps {
			names = append(names, fmt.Sprintf("%d", dep))
		}
		out = append(out, fmt.Sprintf("%d %s %s deps=[%s]", idx, req.Method, req.Name, joinCSV(names)))
	}
	return out
}

func joinCSV(items []string) string {
	if len(items) == 0 {
		return ""
	}
	out := items[0]
	for i := 1; i < len(items); i++ {
		out += ","
		out += items[i]
	}
	return out
}
