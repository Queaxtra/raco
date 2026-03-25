package runner

import (
	"os"
	"path/filepath"
	"raco/model"
	"raco/util"
)

// snapshotEnabled lets collection-level flags opt requests into snapshot checks
// without mutating the underlying request definition.
func snapshotEnabled(req *model.Request, cfg *Config) bool {
	if req != nil && req.Snapshot.Enabled {
		return true
	}
	if cfg == nil {
		return false
	}
	return cfg.SnapshotDir != ""
}

// applySnapshot compares runtime output against an explicitly managed baseline.
// Missing snapshots fail closed so accidental first runs do not bless bad output.
func applySnapshot(collectionID string, req *model.Request, resp *model.Response, cfg *Config) AssertionResult {
	if req == nil || resp == nil || cfg == nil {
		return AssertionResult{Type: "snapshot", Passed: true, Message: "snapshot skipped"}
	}

	path, err := resolveSnapshotPath(collectionID, req, cfg)
	if err != nil {
		return AssertionResult{Type: "snapshot", Passed: false, Message: err.Error()}
	}

	if cfg.SnapshotUpdate {
		if err := writeSnapshot(path, resp.Body); err != nil {
			return AssertionResult{Type: "snapshot", Passed: false, Message: err.Error()}
		}
		return AssertionResult{Type: "snapshot", Passed: true, Message: "snapshot updated"}
	}

	data, err := util.ReadFileBounded(path, 5*1024*1024)
	if err != nil {
		if os.IsNotExist(err) {
			return AssertionResult{Type: "snapshot", Passed: false, Message: "snapshot missing"}
		}
		return AssertionResult{Type: "snapshot", Passed: false, Message: err.Error()}
	}
	if string(data) == resp.Body {
		return AssertionResult{Type: "snapshot", Passed: true, Message: "snapshot matched"}
	}
	return AssertionResult{Type: "snapshot", Passed: false, Message: "snapshot mismatch"}
}

// writeSnapshot uses atomic replacement to avoid truncating a good snapshot when
// a run crashes or the machine loses power mid-write.
func writeSnapshot(path string, body string) error {
	cleanPath := filepath.Clean(path)
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0755); err != nil {
		return err
	}
	tempPath := cleanPath + ".tmp"
	if err := os.WriteFile(tempPath, []byte(body), 0600); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	if err := os.Rename(tempPath, cleanPath); err != nil {
		return err
	}
	return nil
}

// resolveSnapshotPath keeps snapshots scoped to the configured snapshot root or
// the per-collection default, even when a request provides a custom file name.
func resolveSnapshotPath(collectionID string, req *model.Request, cfg *Config) (string, error) {
	baseDir := cfg.SnapshotDir
	if baseDir == "" {
		baseDir = filepath.Join(".raco", "snapshots", collectionID)
	}
	target := req.ID + ".json"
	if req.Snapshot.FilePath != "" {
		target = req.Snapshot.FilePath
	}
	return util.ResolveContainedPath(baseDir, target)
}
