package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"raco/model"
	"raco/storage"
	"raco/util"
	"sort"
	"strings"
)

func collectionHooks(store *storage.Storage, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: raco col hooks set|show|clear <collection-id>")
		return 1
	}

	action := args[0]
	if action == "show" {
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: raco col hooks show <collection-id>")
			return 1
		}
		col, err := store.LoadCollection(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		data, _ := json.MarshalIndent(col.Hooks, "", "  ")
		fmt.Println(string(data))
		return 0
	}

	if action == "clear" {
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: raco col hooks clear <collection-id>")
			return 1
		}
		col, err := store.LoadCollection(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		col.Hooks = model.CollectionHooks{}
		if err := store.SaveCollection(col); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		fmt.Println("Hooks cleared")
		return 0
	}

	if action == "set" {
		fs := flag.NewFlagSet("hooks-set", flag.ContinueOnError)
		var setups multiFlag
		var teardowns multiFlag
		fs.Var(&setups, "setup", "Setup request reference")
		fs.Var(&teardowns, "teardown", "Teardown request reference")
		if err := fs.Parse(args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		remaining := fs.Args()
		if len(remaining) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: raco col hooks set <collection-id> [--setup ref] [--teardown ref]")
			return 1
		}
		col, err := store.LoadCollection(remaining[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		col.Hooks = model.CollectionHooks{
			Setup:    append([]string(nil), setups...),
			Teardown: append([]string(nil), teardowns...),
		}
		if err := store.SaveCollection(col); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		fmt.Println("Hooks updated")
		return 0
	}

	fmt.Fprintf(os.Stderr, "Unknown hooks action: %s\n", action)
	return 1
}

func collectionTags(store *storage.Storage, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: raco col tag add|remove|list ...")
		return 1
	}

	action := args[0]
	fs := flag.NewFlagSet("tag", flag.ContinueOnError)
	requestRef := fs.String("request", "", "Optional request reference")
	if err := fs.Parse(args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: raco col tag add|remove|list <collection-id> [tag] [--request ref]")
		return 1
	}

	col, err := store.LoadCollection(remaining[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	targetTags := &col.Tags
	if *requestRef != "" {
		_, req, err := resolveRequestRef(col, *requestRef)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		targetTags = &req.Tags
	}

	if action == "list" {
		for _, tag := range *targetTags {
			fmt.Println(tag)
		}
		return 0
	}

	if len(remaining) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: raco col tag add|remove <collection-id> <tag> [--request ref]")
		return 1
	}
	tag := strings.TrimSpace(strings.ToLower(remaining[1]))
	if tag == "" {
		fmt.Fprintln(os.Stderr, "Error: tag is required")
		return 1
	}

	if action == "add" {
		*targetTags = append(*targetTags, tag)
		*targetTags = normalizedTags(*targetTags)
		if err := store.SaveCollection(col); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		fmt.Printf("Added tag: %s\n", tag)
		return 0
	}

	if action == "remove" {
		filtered := make([]string, 0, len(*targetTags))
		for _, item := range *targetTags {
			if item != tag {
				filtered = append(filtered, item)
			}
		}
		*targetTags = filtered
		if err := store.SaveCollection(col); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		fmt.Printf("Removed tag: %s\n", tag)
		return 0
	}

	fmt.Fprintf(os.Stderr, "Unknown tag action: %s\n", action)
	return 1
}

func collectionHistory(storagePath string, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: raco col history <collection-id>")
		return 1
	}

	// History lookups stay rooted under the managed history directory so user
	// input cannot be used to read arbitrary JSON files from disk.
	dir, err := util.ResolveContainedPath(filepath.Join(storagePath, "history"), args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Println(strings.TrimSuffix(name, ".json"))
	}
	return 0
}

func collectionDiff(storagePath string, args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: raco col diff <collection-id> <revision-a> <revision-b>")
		return 1
	}

	first, err := loadCollectionRevision(storagePath, args[0], args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	second, err := loadCollectionRevision(storagePath, args[0], args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Printf("collection=%s\n", args[0])
	fmt.Printf("from=%d to=%d\n", first.Revision, second.Revision)
	fmt.Printf("requests=%d -> %d\n", len(first.Collection.Requests), len(second.Collection.Requests))
	fmt.Printf("tags=%s -> %s\n", strings.Join(first.Collection.Tags, ","), strings.Join(second.Collection.Tags, ","))
	return 0
}

func loadCollectionRevision(storagePath string, collectionID string, revisionRef string) (*model.CollectionRevision, error) {
	// Revision files are bounded because history is user-controlled data once it
	// lives on disk, even if Raco wrote it originally.
	path, err := util.ResolveContainedPath(filepath.Join(storagePath, "history", collectionID), revisionRef+".json")
	if err != nil {
		return nil, err
	}
	data, err := util.ReadFileBounded(path, 2*1024*1024)
	if err != nil {
		return nil, err
	}
	var revision model.CollectionRevision
	if err := json.Unmarshal(data, &revision); err != nil {
		return nil, err
	}
	return &revision, nil
}

// normalizedTags keeps tag matching deterministic across CLI commands, storage,
// and runner filters by lowercasing, trimming, deduplicating, and sorting.
func normalizedTags(tags []string) []string {
	if len(tags) == 0 {
		return make([]string, 0)
	}

	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(strings.ToLower(tag))
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	sort.Strings(out)
	return out
}
