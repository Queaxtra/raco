package config

import (
	"fmt"
	"os"
	"path/filepath"
	storagefunc "raco/storage/func"
	"raco/util"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DefaultOutput       string `yaml:"default_output"`
	Notifications       bool   `yaml:"notifications"`
	DefaultEnvironment  string `yaml:"default_environment,omitempty"`
	SnapshotDir         string `yaml:"snapshot_dir,omitempty"`
	HistoryDir          string `yaml:"history_dir,omitempty"`
	CookieDir           string `yaml:"cookie_dir,omitempty"`
	ScriptDir           string `yaml:"script_dir,omitempty"`
	AllowPrivateTargets bool   `yaml:"allow_private_targets,omitempty"`
}

// Default returns the on-disk layout we consider safe and supportable when no
// user configuration exists yet.
func Default(basePath string) Config {
	return Config{
		DefaultOutput: "body",
		Notifications: true,
		SnapshotDir:   filepath.Join(basePath, "snapshots"),
		HistoryDir:    filepath.Join(basePath, "history"),
		CookieDir:     filepath.Join(basePath, "cookies"),
		ScriptDir:     filepath.Join(basePath, "scripts"),
	}
}

func Path(basePath string) string {
	return filepath.Join(basePath, "config.yaml")
}

// Normalize fills empty values with the storage-local defaults so call sites can
// work with a complete config without duplicating fallback logic.
func Normalize(cfg *Config, basePath string) {
	if cfg == nil {
		return
	}
	if strings.TrimSpace(cfg.DefaultOutput) == "" {
		cfg.DefaultOutput = "body"
	}
	if strings.TrimSpace(cfg.SnapshotDir) == "" {
		cfg.SnapshotDir = filepath.Join(basePath, "snapshots")
	}
	if strings.TrimSpace(cfg.HistoryDir) == "" {
		cfg.HistoryDir = filepath.Join(basePath, "history")
	}
	if strings.TrimSpace(cfg.CookieDir) == "" {
		cfg.CookieDir = filepath.Join(basePath, "cookies")
	}
	if strings.TrimSpace(cfg.ScriptDir) == "" {
		cfg.ScriptDir = filepath.Join(basePath, "scripts")
	}
}

// Validate ensures configurable directories never escape the managed Raco base.
// This keeps config flexible without allowing path redirection attacks.
func Validate(cfg Config, basePath string) error {
	for _, path := range []string{cfg.SnapshotDir, cfg.HistoryDir, cfg.CookieDir, cfg.ScriptDir} {
		if _, err := util.ResolveContainedPath(basePath, path); err != nil {
			return err
		}
	}
	return nil
}

// Load reads config from disk, then normalizes and validates it before exposing
// it to the rest of the application.
func Load(basePath string) (Config, error) {
	path := Path(basePath)
	cfg := Default(basePath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return Config{}, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	Normalize(&cfg, basePath)
	if err := Validate(cfg, basePath); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Save re-validates the config at write time so bad in-memory mutations never
// become persistent state.
func Save(basePath string, cfg Config) error {
	Normalize(&cfg, basePath)
	if err := Validate(cfg, basePath); err != nil {
		return err
	}
	if err := storagefunc.EnsureBaseDirs(basePath); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	path := Path(basePath)
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	return os.Rename(tempPath, path)
}

func Reset(basePath string) error {
	err := os.Remove(Path(basePath))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func Get(cfg Config, key string) (string, error) {
	if key == "default_output" {
		return cfg.DefaultOutput, nil
	}
	if key == "notifications" {
		if cfg.Notifications {
			return "true", nil
		}
		return "false", nil
	}
	if key == "default_environment" {
		return cfg.DefaultEnvironment, nil
	}
	if key == "snapshot_dir" {
		return cfg.SnapshotDir, nil
	}
	if key == "history_dir" {
		return cfg.HistoryDir, nil
	}
	if key == "cookie_dir" {
		return cfg.CookieDir, nil
	}
	if key == "script_dir" {
		return cfg.ScriptDir, nil
	}
	if key == "allow_private_targets" {
		if cfg.AllowPrivateTargets {
			return "true", nil
		}
		return "false", nil
	}
	return "", fmt.Errorf("unknown config key: %s", key)
}

func Set(cfg *Config, key string, value string, basePath string) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	// Every setter path re-runs validation so invalid directories fail before the
	// caller has a chance to persist or consume them.
	if key == "default_output" {
		cfg.DefaultOutput = strings.TrimSpace(value)
		Normalize(cfg, basePath)
		return Validate(*cfg, basePath)
	}
	if key == "notifications" {
		cfg.Notifications = strings.EqualFold(strings.TrimSpace(value), "true")
		Normalize(cfg, basePath)
		return Validate(*cfg, basePath)
	}
	if key == "default_environment" {
		cfg.DefaultEnvironment = strings.TrimSpace(value)
		Normalize(cfg, basePath)
		return Validate(*cfg, basePath)
	}
	if key == "snapshot_dir" {
		cfg.SnapshotDir = strings.TrimSpace(value)
		Normalize(cfg, basePath)
		return Validate(*cfg, basePath)
	}
	if key == "history_dir" {
		cfg.HistoryDir = strings.TrimSpace(value)
		Normalize(cfg, basePath)
		return Validate(*cfg, basePath)
	}
	if key == "cookie_dir" {
		cfg.CookieDir = strings.TrimSpace(value)
		Normalize(cfg, basePath)
		return Validate(*cfg, basePath)
	}
	if key == "script_dir" {
		cfg.ScriptDir = strings.TrimSpace(value)
		Normalize(cfg, basePath)
		return Validate(*cfg, basePath)
	}
	if key == "allow_private_targets" {
		cfg.AllowPrivateTargets = strings.EqualFold(strings.TrimSpace(value), "true")
		Normalize(cfg, basePath)
		return Validate(*cfg, basePath)
	}
	return fmt.Errorf("unknown config key: %s", key)
}

// List returns a flat view tailored for CLI output rather than a serialization
// format. Boolean values are stringified for stable command printing.
func List(cfg Config) map[string]string {
	values := map[string]string{
		"allow_private_targets": "false",
		"cookie_dir":            cfg.CookieDir,
		"default_environment":   cfg.DefaultEnvironment,
		"default_output":        cfg.DefaultOutput,
		"history_dir":           cfg.HistoryDir,
		"notifications":         "false",
		"script_dir":            cfg.ScriptDir,
		"snapshot_dir":          cfg.SnapshotDir,
	}
	if cfg.Notifications {
		values["notifications"] = "true"
	}
	if cfg.AllowPrivateTargets {
		values["allow_private_targets"] = "true"
	}
	return values
}

func SortedKeys(values map[string]string) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
