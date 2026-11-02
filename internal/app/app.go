// Package app is the application layer. It combines the catalog with search,
// favorites and local state. The TUI only calls this package.
package app

import (
	"fmt"
	"sort"
	"strings"

	"ntexplorer/internal/catalog"
	"ntexplorer/internal/config"
	"ntexplorer/internal/search"
)

// App is the long-lived application state shared by the TUI.
type App struct {
	Catalog  *catalog.Catalog
	Config   config.Config
	State    config.State
	Issues   []catalog.Issue
	Warnings []string
	DataDir  string // "embedded" or a filesystem path
}

// New builds the application layer. State is loaded from the standard user
// configuration directory; failures become warnings, never fatal errors.
func New(cat *catalog.Catalog, cfg config.Config, configWarnings []string, issues []catalog.Issue) *App {
	a := &App{
		Catalog:  cat,
		Config:   cfg,
		Issues:   issues,
		Warnings: append([]string(nil), configWarnings...),
		DataDir:  "embedded",
	}
	state, stateWarnings := config.LoadState()
	a.State = state
	a.Warnings = append(a.Warnings, stateWarnings...)
	return a
}

// Search ranks all APIs for query.
func (a *App) Search(query string) []search.Result {
	if a == nil || a.Catalog == nil {
		return nil
	}
	return search.Query(a.Catalog.APIs, query, a.Config.MaxResults)
}

// Browse returns APIs for a category. An empty category means all APIs.
func (a *App) Browse(category string) []catalog.API {
	if a == nil || a.Catalog == nil {
		return nil
	}
	return a.Catalog.APIsForCategory(category)
}

// BrowseTag returns APIs carrying tag.
func (a *App) BrowseTag(tag string) []catalog.API {
	if a == nil || a.Catalog == nil {
		return nil
	}
	return a.Catalog.APIsWithTag(tag)
}

// BrowseStructures returns the structure catalog in sorted order.
func (a *App) BrowseStructures() []catalog.Structure {
	if a == nil || a.Catalog == nil {
		return nil
	}
	return a.Catalog.Structures
}

// Structure resolves a structure by ID.
func (a *App) Structure(id string) (*catalog.Structure, bool) {
	if a == nil || a.Catalog == nil {
		return nil, false
	}
	return a.Catalog.Structure(id)
}

// SearchStructures ranks structures for query.
func (a *App) SearchStructures(query string) []search.StructureResult {
	if a == nil || a.Catalog == nil {
		return nil
	}
	return search.QueryStructures(a.Catalog.Structures, query, a.Config.MaxResults)
}

// StructureRelatedAPIs returns APIs that name id in related_structures, in
// catalog order.
func (a *App) StructureRelatedAPIs(id string) []catalog.API {
	if a == nil || a.Catalog == nil {
		return nil
	}
	out := make([]catalog.API, 0, 8)
	for _, api := range a.Catalog.APIs {
		for _, related := range api.RelatedStructures {
			if related == id {
				out = append(out, api)
				break
			}
		}
	}
	return out
}

// BrowseConstants returns the constant catalog in sorted order.
func (a *App) BrowseConstants() []catalog.Constant {
	if a == nil || a.Catalog == nil {
		return nil
	}
	return a.Catalog.Constants
}

// Constant resolves a constant by ID.
func (a *App) Constant(id string) (*catalog.Constant, bool) {
	if a == nil || a.Catalog == nil {
		return nil, false
	}
	return a.Catalog.Constant(id)
}

// SearchConstants ranks constants for query.
func (a *App) SearchConstants(query string) []search.ConstantResult {
	if a == nil || a.Catalog == nil {
		return nil
	}
	return search.QueryConstants(a.Catalog.Constants, query, a.Config.MaxResults)
}

// ConstantRelatedAPIs returns APIs that link to id through related_constants
// or through a parameter that names the constant, in catalog order.
func (a *App) ConstantRelatedAPIs(id string) []catalog.API {
	if a == nil || a.Catalog == nil {
		return nil
	}
	constant, ok := a.Catalog.Constant(id)
	if !ok {
		return nil
	}
	out := make([]catalog.API, 0, 8)
	for _, api := range a.Catalog.APIs {
		if apiReferencesConstant(api, id, constant.Name) {
			out = append(out, api)
		}
	}
	return out
}

func apiReferencesConstant(api catalog.API, id, name string) bool {
	for _, related := range api.RelatedConstants {
		if related == id {
			return true
		}
	}
	for _, parameter := range api.Parameters {
		for _, symbol := range parameter.Constants {
			if symbol == name {
				return true
			}
		}
	}
	return false
}

// Tags returns the tag vocabulary with usage counts.
func (a *App) Tags() []catalog.TagCount {
	if a == nil || a.Catalog == nil {
		return nil
	}
	return a.Catalog.TagCounts()
}

// CategoryNames returns the primary categories in declared order.
func (a *App) CategoryNames() []string {
	if a == nil || a.Catalog == nil {
		return nil
	}
	names := make([]string, 0, len(a.Catalog.Categories))
	for _, category := range a.Catalog.Categories {
		names = append(names, category.Name)
	}
	return names
}

// API resolves an API by ID.
func (a *App) API(id string) (*catalog.API, bool) {
	if a == nil || a.Catalog == nil {
		return nil, false
	}
	return a.Catalog.API(id)
}

// IsFavorite reports whether id is in the favorites list.
func (a *App) IsFavorite(id string) bool {
	for _, favorite := range a.State.Favorites {
		if favorite == id {
			return true
		}
	}
	return false
}

// ToggleFavorite adds or removes an API ID from favorites and persists the
// change. It returns the new favorite state and any persistence error.
func (a *App) ToggleFavorite(id string) (bool, error) {
	if a.IsFavorite(id) {
		next := a.State.Favorites[:0]
		for _, favorite := range a.State.Favorites {
			if favorite != id {
				next = append(next, favorite)
			}
		}
		a.State.Favorites = next
		return false, config.SaveState(a.State)
	}
	a.State.Favorites = append(a.State.Favorites, id)
	return true, config.SaveState(a.State)
}

// FavoriteAPIs returns resolvable favorites in saved order.
func (a *App) FavoriteAPIs() []catalog.API {
	return a.resolveIDs(a.State.Favorites)
}

// RecordRecent marks an API as recently viewed and persists the change.
func (a *App) RecordRecent(id string) error {
	a.State.Recent = config.AddRecent(a.State.Recent, id)
	return config.SaveState(a.State)
}

// RecentAPIs returns resolvable recently viewed entries, newest first.
func (a *App) RecentAPIs() []catalog.API {
	return a.resolveIDs(a.State.Recent)
}

// RememberCategory persists the last browsed category.
func (a *App) RememberCategory(category string) {
	a.State.LastCategory = category
}

// SaveState persists favorites, recent items and the last category.
func (a *App) SaveState() error {
	return config.SaveState(a.State)
}

func (a *App) resolveIDs(ids []string) []catalog.API {
	out := make([]catalog.API, 0, len(ids))
	for _, id := range ids {
		if api, ok := a.API(id); ok {
			out = append(out, *api)
		}
	}
	return out
}

// ErrorIssues returns validation issues with error severity.
func (a *App) ErrorIssues() []catalog.Issue {
	var out []catalog.Issue
	for _, issue := range a.Issues {
		if issue.Severity == catalog.SeverityError {
			out = append(out, issue)
		}
	}
	return out
}

// WarningIssues returns validation issues with warning severity.
func (a *App) WarningIssues() []catalog.Issue {
	var out []catalog.Issue
	for _, issue := range a.Issues {
		if issue.Severity == catalog.SeverityWarning {
			out = append(out, issue)
		}
	}
	return out
}

// StatusLine summarizes the catalog for the TUI status bar.
func (a *App) StatusLine() string {
	if a == nil || a.Catalog == nil {
		return "catalog unavailable"
	}
	parts := []string{
		fmt.Sprintf("catalog %s", valueOr(a.Catalog.CatalogVersion, "unversioned")),
		fmt.Sprintf("%d APIs", len(a.Catalog.APIs)),
	}
	if len(a.Catalog.Structures) > 0 {
		parts = append(parts, fmt.Sprintf("%d structures", len(a.Catalog.Structures)))
	}
	if len(a.Catalog.Constants) > 0 {
		parts = append(parts, fmt.Sprintf("%d constants", len(a.Catalog.Constants)))
	}
	if a.DataDir != "" && a.DataDir != "embedded" {
		parts = append(parts, "external data")
	}
	return strings.Join(parts, " · ")
}

// WarningsText returns startup warnings as one readable paragraph.
func (a *App) WarningsText() string {
	if a == nil || len(a.Warnings) == 0 {
		return ""
	}
	return strings.Join(a.Warnings, "; ")
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// SortByName sorts APIs alphabetically in place.
func SortByName(apis []catalog.API) {
	sort.SliceStable(apis, func(i, j int) bool {
		return strings.ToLower(apis[i].Name) < strings.ToLower(apis[j].Name)
	})
}
