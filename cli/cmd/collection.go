package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"raco/model"
	"raco/storage"
	"raco/util"
	"strings"
	"time"

	"github.com/google/uuid"
)

func RunCollection(ctx *Context, args []string) int {
	if len(args) == 0 {
		printCollectionUsage()
		return 1
	}

	store := ctx.Storage()
	action := args[0]
	subArgs := args[1:]

	switch action {
	case "list", "ls":
		return collectionList(store)
	case "show", "get":
		return collectionShow(store, subArgs)
	case "create", "new":
		return collectionCreate(store, subArgs)
	case "delete", "rm":
		return collectionDelete(ctx.StoragePath, subArgs)
	case "add-request", "add":
		return collectionAddRequest(ctx, store, subArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown action: %s\n", action)
		printCollectionUsage()
		return 1
	}
}

func printCollectionUsage() {
	fmt.Println(`Usage: raco collection <action> [options]

Actions:
  list, ls              List all collections
  show, get <id>        Show collection details
  create, new <name>    Create new collection
  delete, rm <id>       Delete collection
  add, add-request      Add request to collection

Examples:
  raco col list
  raco col create "My API Tests"
  raco col show my-api-tests
  raco col add my-api-tests -n "Get Users" -m GET -r https://api.example.org/users`)
}

func collectionList(store *storage.Storage) int {
	collections, err := store.ListCollections()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if len(collections) == 0 {
		fmt.Println("No collections found")
		return 0
	}

	for _, col := range collections {
		fmt.Printf("%s  %s  (%d requests)\n", col.ID, col.Name, len(col.Requests))
	}

	return 0
}

func collectionShow(store *storage.Storage, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: collection ID is required")
		return 1
	}

	col, err := store.LoadCollection(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	data, _ := json.MarshalIndent(col, "", "  ")
	fmt.Println(string(data))
	return 0
}

func collectionCreate(store *storage.Storage, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: collection name is required")
		return 1
	}

	name := args[0]
	id := generateSlug(name)

	col := &model.Collection{
		ID:       id,
		Name:     name,
		Requests: []*model.Request{},
	}

	if err := store.SaveCollection(col); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Printf("Created collection: %s (%s)\n", name, id)
	return 0
}

func collectionDelete(storagePath string, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: collection ID is required")
		return 1
	}

	id := args[0]
	if !isValidCollectionID(id) {
		fmt.Fprintln(os.Stderr, "Error: invalid collection ID format")
		return 1
	}

	path := filepath.Join(storagePath, "collections", id+".json")

	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		resolvedPath = path
	}

	expectedDir := filepath.Join(storagePath, "collections")
	if !util.IsPathContained(resolvedPath, expectedDir) {
		fmt.Fprintln(os.Stderr, "Error: invalid path")
		return 1
	}

	if err := os.Remove(resolvedPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Printf("Deleted collection: %s\n", id)
	return 0
}

func isValidCollectionID(id string) bool {
	if len(id) == 0 || len(id) > 64 {
		return false
	}

	for i, r := range id {
		isLower := r >= 'a' && r <= 'z'
		isUpper := r >= 'A' && r <= 'Z'
		isDigit := r >= '0' && r <= '9'
		isSpecial := r == '_' || r == '-'

		if !isLower && !isUpper && !isDigit && !isSpecial {
			return false
		}

		if i == 0 && !isLower && !isUpper && !isDigit {
			return false
		}
	}

	return true
}

func collectionAddRequest(ctx *Context, store *storage.Storage, args []string) int {
	if len(args) < 5 {
		fmt.Fprintln(os.Stderr, "Usage: raco col add <collection-id> -n <name> -m <method> -r <url> [-d body] [-H headers]")
		return 1
	}

	colID := args[0]
	col, err := store.LoadCollection(colID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading collection: %v\n", err)
		return 1
	}

	method, url, body, headers, err := ParseRequestArgsPublic(args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	var name string
	for i, arg := range args {
		if arg == "-n" && i+1 < len(args) {
			name = args[i+1]
			break
		}
	}

	if name == "" {
		name = method + " " + url
	}

	req := &model.Request{
		ID:           uuid.New().String(),
		Name:         name,
		Method:       strings.ToUpper(method),
		URL:          url,
		Headers:      headers,
		Body:         body,
		CreatedAt:    time.Now(),
		CollectionID: colID,
	}

	col.Requests = append(col.Requests, req)

	if err := store.SaveCollection(col); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving collection: %v\n", err)
		return 1
	}

	fmt.Printf("Added request '%s' to collection '%s'\n", name, col.Name)
	return 0
}

func generateSlug(name string) string {
	slug := ""
	for _, r := range name {
		if r >= 'a' && r <= 'z' {
			slug += string(r)
			continue
		}
		if r >= 'A' && r <= 'Z' {
			slug += string(r + 32)
			continue
		}
		if r >= '0' && r <= '9' {
			slug += string(r)
			continue
		}
		if r == ' ' || r == '-' || r == '_' {
			if len(slug) > 0 && slug[len(slug)-1] != '-' {
				slug += "-"
			}
		}
	}

	if len(slug) > 0 && slug[len(slug)-1] == '-' {
		slug = slug[:len(slug)-1]
	}

	return slug
}
