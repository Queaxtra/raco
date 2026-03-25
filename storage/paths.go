package storage

import "path/filepath"

func CollectionsPath(basePath string) string {
	return filepath.Join(basePath, "collections")
}

func EnvironmentsPath(basePath string) string {
	return filepath.Join(basePath, "environments")
}

func HistoryPath(basePath string) string {
	return filepath.Join(basePath, "history")
}

func HistoryCollectionPath(basePath string, collectionID string) string {
	return filepath.Join(HistoryPath(basePath), collectionID)
}

func SnapshotsPath(basePath string) string {
	return filepath.Join(basePath, "snapshots")
}

func SnapshotCollectionPath(basePath string, collectionID string) string {
	return filepath.Join(SnapshotsPath(basePath), collectionID)
}

func CookiesPath(basePath string) string {
	return filepath.Join(basePath, "cookies")
}

func ScriptsPath(basePath string) string {
	return filepath.Join(basePath, "scripts")
}
