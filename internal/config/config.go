// Package config loads optional user configuration and local user state.
// Both are optional: defaults are always usable.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	appDirName       = "NtExplorer"
	configFileName   = "config.json"
	stateFileName    = "state.json"
	defaultMaxResult = 25
	maxRecentItems   = 20
)

// Config holds optional user preferences. Zero values are replaced by
// defaults during Load.
type Config struct {
	Theme           string `json:"theme,omitempty"`
	InitialCategory string `json:"initial_category,omitempty"`
	ShowExamples    *bool  `json:"show_examples,omitempty"`
	CatalogDir      string `json:"catalog_dir,omitempty"`
	MaxResults      int    `json:"max_results,omitempty"`
}

// Default returns the configuration used when no config file is present.
func Default() Config {
	return Config{
		Theme:      "dark",
		MaxResults: defaultMaxResult,
	}
}

// State is local user state that is separate from the bundled catalog.
type State struct {
	Favorites    []string `json:"favorites,omitempty"`
	Recent       []string `json:"recent,omitempty"`
	LastCategory string   `json:"last_category,omitempty"`
}

// Dir returns the directory used for configuration and state. The
// NTEXPLORER_CONFIG_DIR environment variable overrides the platform default,
// which is useful for tests and portable installs.
func Dir() (string, error) {
	if override := strings.TrimSpace(os.Getenv("NTEXPLORER_CONFIG_DIR")); override != "" {
		return override, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	return filepath.Join(dir, appDirName), nil
}

// Load reads config.json. A missing file yields defaults without an error.
// Invalid values are replaced by defaults and reported as warnings.
func Load() (Config, []string, error) {
	cfg := Default()
	dir, err := Dir()
	if err != nil {
		return cfg, []string{err.Error()}, nil
	}
	raw, err := os.ReadFile(filepath.Join(dir, configFileName))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cfg, nil, nil
		}
		return cfg, []string{fmt.Sprintf("could not read config: %v", err)}, nil
	}

	var loaded Config
	if err := json.Unmarshal(raw, &loaded); err != nil {
		return cfg, []string{fmt.Sprintf("config.json is invalid, using defaults: %v", err)}, nil
	}
	warnings := applyConfig(&cfg, loaded)
	cfg.ShowExamples = loaded.ShowExamples
	return cfg, warnings, nil
}

func applyConfig(cfg *Config, loaded Config) []string {
	var warnings []string
	if loaded.Theme != "" {
		switch loaded.Theme {
		case "dark", "light":
			cfg.Theme = loaded.Theme
		default:
			warnings = append(warnings, fmt.Sprintf("unknown theme %q, using dark", loaded.Theme))
		}
	}
	cfg.InitialCategory = strings.TrimSpace(loaded.InitialCategory)
	cfg.CatalogDir = strings.TrimSpace(loaded.CatalogDir)
	cfg.MaxResults = loaded.MaxResults
	if cfg.MaxResults <= 0 {
		cfg.MaxResults = defaultMaxResult
	}
	if cfg.MaxResults > 200 {
		warnings = append(warnings, "max_results is capped at 200")
		cfg.MaxResults = 200
	}
	return warnings
}

// LoadState reads state.json. A missing or invalid file yields empty state.
func LoadState() (State, []string) {
	var state State
	dir, err := Dir()
	if err != nil {
		return state, []string{err.Error()}
	}
	raw, err := os.ReadFile(filepath.Join(dir, stateFileName))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return state, nil
		}
		return state, []string{fmt.Sprintf("could not read saved state: %v", err)}
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		return State{}, []string{fmt.Sprintf("state.json is invalid, starting fresh: %v", err)}
	}
	state.Favorites = dedupe(state.Favorites)
	state.Recent = dedupe(state.Recent)
	return state, nil
}

// SaveState writes state.json atomically, creating the directory if needed.
func SaveState(state State) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	state.Favorites = dedupe(state.Favorites)
	state.Recent = dedupe(state.Recent)
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')

	tmp, err := os.CreateTemp(dir, ".state-*.json")
	if err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, filepath.Join(dir, stateFileName))
}

// AddRecent returns recent IDs with id as the most recent entry, bounded by
// maxRecentItems.
func AddRecent(recent []string, id string) []string {
	if id == "" {
		return recent
	}
	out := make([]string, 0, len(recent)+1)
	out = append(out, id)
	for _, existing := range recent {
		if existing != id {
			out = append(out, existing)
		}
	}
	if len(out) > maxRecentItems {
		out = out[:maxRecentItems]
	}
	return out
}

func dedupe(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
