package secretstore

import (
	"fmt"
	"strings"
)

// FakeStore is an in-memory Store implementation used by tests and local diagnostics.
type FakeStore struct {
	values map[string]string
}

// NewFakeStore returns an isolated test double with no OS dependencies.
func NewFakeStore() *FakeStore {
	return &FakeStore{values: make(map[string]string)}
}

func fakeKey(envName string, key string) string {
	return envName + ":" + key
}

// Set stores a value entirely in memory.
func (s *FakeStore) Set(envName string, key string, value string) error {
	s.values[fakeKey(envName, key)] = value
	return nil
}

// Get loads a value from memory and mirrors Store lookup semantics.
func (s *FakeStore) Get(envName string, key string) (string, error) {
	value, ok := s.values[fakeKey(envName, key)]
	if !ok {
		return "", fmt.Errorf("secret not found")
	}
	return value, nil
}

// Delete removes a value from memory.
func (s *FakeStore) Delete(envName string, key string) error {
	delete(s.values, fakeKey(envName, key))
	return nil
}

// List returns the in-memory keys for a given environment namespace.
func (s *FakeStore) List(envName string) ([]string, error) {
	results := make([]string, 0)
	prefix := envName + ":"
	for fullKey := range s.values {
		if strings.HasPrefix(fullKey, prefix) {
			results = append(results, strings.TrimPrefix(fullKey, prefix))
		}
	}
	return results, nil
}
