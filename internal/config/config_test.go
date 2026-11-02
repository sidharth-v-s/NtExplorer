package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultsWithoutFile(t *testing.T) {
	t.Setenv("NTEXPLORER_CONFIG_DIR", t.TempDir())
	cfg, warnings, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if cfg.Theme != "dark" || cfg.MaxResults != defaultMaxResult {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadInvalidConfigFallsBack(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NTEXPLORER_CONFIG_DIR", dir)
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, warnings, err := Load()
	if err != nil {
		t.Fatalf("invalid config should not be fatal: %v", err)
	}
	if cfg.MaxResults != defaultMaxResult {
		t.Fatalf("expected defaults, got %+v", cfg)
	}
	if len(warnings) == 0 {
		t.Fatal("expected a warning for invalid config")
	}
}

func TestLoadClampsAndWarns(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NTEXPLORER_CONFIG_DIR", dir)
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"theme":"neon","max_results":1000}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, warnings, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme != "dark" {
		t.Fatalf("unknown theme should fall back to dark, got %q", cfg.Theme)
	}
	if cfg.MaxResults != 200 {
		t.Fatalf("max results not clamped: %d", cfg.MaxResults)
	}
	if len(warnings) != 2 {
		t.Fatalf("expected two warnings, got %v", warnings)
	}
}

func TestStateRoundTrip(t *testing.T) {
	t.Setenv("NTEXPLORER_CONFIG_DIR", t.TempDir())
	state := State{
		Favorites:    []string{"win32.virtualalloc", "win32.virtualalloc"},
		Recent:       []string{"win32.openprocess"},
		LastCategory: "Memory Management",
	}
	if err := SaveState(state); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	loaded, warnings := LoadState()
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if len(loaded.Favorites) != 1 || loaded.Favorites[0] != "win32.virtualalloc" {
		t.Fatalf("favorites not deduplicated: %v", loaded.Favorites)
	}
	if loaded.LastCategory != "Memory Management" {
		t.Fatalf("last category lost: %+v", loaded)
	}
}

func TestLoadInvalidStateStartsFresh(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NTEXPLORER_CONFIG_DIR", dir)
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte("nonsense"), 0o600); err != nil {
		t.Fatal(err)
	}
	state, warnings := LoadState()
	if len(state.Favorites) != 0 || len(state.Recent) != 0 {
		t.Fatalf("expected empty state, got %+v", state)
	}
	if len(warnings) == 0 {
		t.Fatal("expected a warning")
	}
}

func TestAddRecentDeduplicatesAndBounds(t *testing.T) {
	var recent []string
	for i := 0; i < 25; i++ {
		recent = AddRecent(recent, string(rune('a'+i)))
	}
	if len(recent) != maxRecentItems {
		t.Fatalf("recent not bounded: %d", len(recent))
	}
	recent = AddRecent(recent, "a")
	if recent[0] != "a" {
		t.Fatalf("most recent should be first: %v", recent)
	}
	count := 0
	for _, id := range recent {
		if id == "a" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("duplicate id retained: %v", recent)
	}
}
