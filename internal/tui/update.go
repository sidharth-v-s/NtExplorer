package tui

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"ntexplorer/internal/catalog"
	"ntexplorer/internal/export"
	"ntexplorer/internal/search"
)

type browseState struct {
	mode             listMode
	category         string
	tagFilter        string
	apiList          []catalog.API
	structures       []catalog.Structure
	structureResults []search.StructureResult
	constants        []catalog.Constant
	constantResults  []search.ConstantResult
	statuses         []catalog.NTStatus
	statusResults    []search.StatusResult
	listIndex        int
	tagIndex         int
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update handles messages. Keys are dispatched by view so text input mode can
// never steal global shortcuts accidentally.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		m.ensureValidFocus()
		return m, nil
	case tea.KeyMsg:
		m.statusErr = false
		// Ctrl+C always quits, even while typing in search or an overlay.
		if msg.String() == "ctrl+c" {
			if m.app != nil {
				_ = m.app.SaveState()
			}
			return m, tea.Quit
		}
		if m.searching {
			return m.updateSearch(msg)
		}
		if m.overlay != overlayNone {
			return m.updateOverlay(msg)
		}
		return m.updateBrowse(msg)
	}
	return m, nil
}

func (m Model) updateBrowse(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	keys := m.keys

	switch {
	case key.Matches(msg, keys.Quit):
		if m.app != nil {
			_ = m.app.SaveState()
		}
		return m, tea.Quit

	case key.Matches(msg, keys.Help):
		m.overlay = overlayHelp
		m.helpOffset = 0
		return m, nil

	case key.Matches(msg, keys.Search):
		m.startSearch()
		return m, m.searchInput.Focus()

	case key.Matches(msg, keys.Tab):
		m.cycleFocus(1)
		return m, nil
	case key.Matches(msg, keys.ShiftTab):
		m.cycleFocus(-1)
		return m, nil

	case key.Matches(msg, keys.Escape):
		m.escape()
		return m, nil

	case key.Matches(msg, keys.Up):
		m.moveFocus(-1, false)
		return m, nil
	case key.Matches(msg, keys.Down):
		m.moveFocus(1, false)
		return m, nil
	case key.Matches(msg, keys.PageUp):
		m.moveFocus(-1, true)
		return m, nil
	case key.Matches(msg, keys.PageDown):
		m.moveFocus(1, true)
		return m, nil
	case key.Matches(msg, keys.Top):
		m.jumpFocus(false)
		return m, nil
	case key.Matches(msg, keys.Bottom):
		m.jumpFocus(true)
		return m, nil
	case key.Matches(msg, keys.HalfUp):
		if m.focus == panelDetail {
			m.detail.HalfPageUp()
		}
		return m, nil
	case key.Matches(msg, keys.HalfDown):
		if m.focus == panelDetail {
			m.detail.HalfPageDown()
		}
		return m, nil

	case key.Matches(msg, keys.Enter):
		switch m.focus {
		case panelCategories:
			m.focus = panelList
		case panelList:
			m.openSelected()
		}
		return m, nil

	case key.Matches(msg, keys.Related):
		m.openRelated()
		return m, nil
	case key.Matches(msg, keys.References):
		m.openReferences()
		return m, nil
	case key.Matches(msg, keys.Favorite):
		m.toggleFavorite()
		return m, nil
	case key.Matches(msg, keys.Export):
		m.openExport()
		return m, nil
	case key.Matches(msg, keys.Copy):
		m.openCopy()
		return m, nil
	case key.Matches(msg, keys.Examples):
		m.showExamples = !m.showExamples
		m.refreshDetail()
		if m.showExamples {
			m.status = "Examples shown."
		} else {
			m.status = "Examples hidden."
		}
		return m, nil
	case key.Matches(msg, keys.Structures):
		m.openStructures()
		return m, nil
	case key.Matches(msg, keys.Constants):
		m.openConstants()
		return m, nil
	}
	return m, nil
}

// moveFocus moves the selection or scrolls, depending on the focused panel.
func (m *Model) moveFocus(delta int, page bool) {
	switch m.focus {
	case panelCategories:
		m.moveSidebar(delta)
	case panelList:
		if page {
			m.moveList(delta * 5)
			return
		}
		m.moveList(delta)
	case panelDetail:
		if page {
			if delta > 0 {
				m.detail.PageDown()
			} else {
				m.detail.PageUp()
			}
			return
		}
		if delta > 0 {
			m.detail.ScrollDown(1)
		} else {
			m.detail.ScrollUp(1)
		}
	}
}

func (m *Model) jumpFocus(toEnd bool) {
	switch m.focus {
	case panelList, panelCategories:
		m.jumpList(toEnd)
	case panelDetail:
		if toEnd {
			m.detail.GotoBottom()
		} else {
			m.detail.GotoTop()
		}
	}
}

func (m *Model) escape() {
	switch {
	case m.focus == panelDetail && m.query != "":
		// First leave the detail pane, then clear the search on the next Esc.
		m.focus = panelList
	case m.focus == panelDetail:
		m.focus = panelList
	case m.query != "":
		m.cancelSearch()
	case m.mode == listBrowse && m.tagFilter != "":
		m.mode = listTags
		m.tagFilter = ""
		m.apiList = nil
		m.listIndex = 0
		m.syncPreview()
		m.status = "Tag filter cleared."
	case m.focus == panelList && m.width >= minThreePanels:
		m.focus = panelCategories
	default:
		m.status = ""
	}
}

func (m *Model) startSearch() {
	if m.preSearch == nil {
		m.preSearch = &browseState{
			mode:             m.mode,
			category:         m.category,
			tagFilter:        m.tagFilter,
			apiList:          append([]catalog.API(nil), m.apiList...),
			structures:       append([]catalog.Structure(nil), m.structures...),
			structureResults: append([]search.StructureResult(nil), m.structureResults...),
			constants:        append([]catalog.Constant(nil), m.constants...),
			constantResults:  append([]search.ConstantResult(nil), m.constantResults...),
			statuses:         append([]catalog.NTStatus(nil), m.statuses...),
			statusResults:    append([]search.StatusResult(nil), m.statusResults...),
			listIndex:        m.listIndex,
			tagIndex:         m.tagIndex,
		}
	}
	savedMode := m.preSearch.mode
	m.searchStructures = savedMode == listStructures || savedMode == listStructureSearch
	m.searchConstants = savedMode == listConstants || savedMode == listConstantSearch
	m.searchStatuses = savedMode == listStatuses || savedMode == listStatusSearch
	switch {
	case m.searchStructures:
		m.searchInput.Placeholder = "structure name, member, tag, or text"
	case m.searchConstants:
		m.searchInput.Placeholder = "constant name, value, tag, or text"
	case m.searchStatuses:
		m.searchInput.Placeholder = "status name, value, severity, or text"
	default:
		m.searchInput.Placeholder = "name, tag, category, or description"
	}
	m.query = ""
	m.searchInput.SetValue("")
	m.searching = true
	m.focus = panelList
}

func (m *Model) applySearch() {
	if strings.TrimSpace(m.query) == "" {
		m.results = nil
		m.structureResults = nil
		m.constantResults = nil
		m.statusResults = nil
		// Deleting the query restores the list the user was browsing, while
		// keeping the search field active.
		m.restoreSnapshot(false)
		return
	}
	m.results = nil
	m.structureResults = nil
	m.constantResults = nil
	m.statusResults = nil
	switch {
	case m.searchStructures:
		m.mode = listStructureSearch
		if m.app != nil {
			m.structureResults = m.app.SearchStructures(m.query)
		}
	case m.searchConstants:
		m.mode = listConstantSearch
		if m.app != nil {
			m.constantResults = m.app.SearchConstants(m.query)
		}
	case m.searchStatuses:
		m.mode = listStatusSearch
		if m.app != nil {
			m.statusResults = m.app.SearchStatuses(m.query)
		}
	default:
		m.mode = listSearch
		if m.app != nil {
			m.results = m.app.Search(m.query)
		}
	}
	m.listIndex = 0
	m.syncPreview()
}

func (m *Model) cancelSearch() {
	m.searching = false
	m.searchStructures = false
	m.searchConstants = false
	m.searchStatuses = false
	m.searchInput.Blur()
	m.searchInput.SetValue("")
	m.query = ""
	m.results = nil
	m.structureResults = nil
	m.constantResults = nil
	m.statusResults = nil
	m.restoreSnapshot(true)
	m.focus = panelList
	m.status = ""
}

// restoreSnapshot returns to the browse state saved when search started. When
// clear is false the snapshot is kept so it can be restored again.
func (m *Model) restoreSnapshot(clear bool) {
	saved := m.preSearch
	if saved == nil {
		m.reloadSidebarList()
		return
	}
	m.mode = saved.mode
	m.category = saved.category
	m.tagFilter = saved.tagFilter
	m.apiList = saved.apiList
	m.structures = saved.structures
	m.structureResults = saved.structureResults
	m.constants = saved.constants
	m.constantResults = saved.constantResults
	m.statuses = saved.statuses
	m.statusResults = saved.statusResults
	m.listIndex = saved.listIndex
	m.tagIndex = saved.tagIndex
	if clear {
		m.preSearch = nil
	}
	m.syncPreview()
}

func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	keys := m.keys

	switch {
	case key.Matches(msg, keys.Escape):
		m.cancelSearch()
		return m, nil
	case key.Matches(msg, keys.Enter):
		m.searching = false
		m.searchInput.Blur()
		if m.query == "" {
			m.cancelSearch()
			return m, nil
		}
		if m.listLength() == 0 {
			m.status = "No matches for " + `"` + m.query + `"` + "."
			return m, nil
		}
		m.openSelected()
		return m, nil
	case key.Matches(msg, keys.Up):
		m.moveList(-1)
		return m, nil
	case key.Matches(msg, keys.Down):
		m.moveList(1)
		return m, nil
	case key.Matches(msg, keys.PageUp):
		m.moveList(-5)
		return m, nil
	case key.Matches(msg, keys.PageDown):
		m.moveList(5)
		return m, nil
	}

	updated, cmd := m.searchInput.Update(msg)
	m.searchInput = updated
	value := m.searchInput.Value()
	if value != m.query {
		m.query = value
		m.applySearch()
	}
	return m, cmd
}

func (m *Model) openRelated() {
	if m.statusActive() {
		m.status = "Related API links are not recorded for NTSTATUS values."
		return
	}
	if m.constantActive() {
		if len(m.currentRelatedIDs()) == 0 {
			m.status = "No related APIs are documented for " + m.currentConstant.Name + "."
			return
		}
		m.overlay = overlayRelated
		m.overlayIndex = 0
		return
	}
	if m.structureActive() {
		if len(m.currentRelatedIDs()) == 0 {
			m.status = "No related APIs are documented for " + m.currentStructure.Name + "."
			return
		}
		m.overlay = overlayRelated
		m.overlayIndex = 0
		return
	}
	api, ok := m.currentEntry()
	if !ok {
		m.status = "Select an API, structure, or constant to see related entries."
		return
	}
	if len(api.RelatedAPIs) == 0 {
		m.status = "No related APIs are documented for " + api.Name + "."
		return
	}
	m.overlay = overlayRelated
	m.overlayIndex = 0
}

func (m *Model) openReferences() {
	name, references, ok := m.currentReferences()
	if !ok {
		m.status = "Select an entry to see references."
		return
	}
	if len(references) == 0 {
		m.status = "No references are recorded for " + name + "."
		return
	}
	m.overlay = overlayReferences
	m.refOffset = 0
}

// currentRelatedIDs returns the API IDs linked to the current entry. For a
// structure or constant this is the reverse link: APIs that name it.
func (m Model) currentRelatedIDs() []string {
	if m.constantActive() {
		if m.app == nil {
			return nil
		}
		apis := m.app.ConstantRelatedAPIs(m.currentConstant.ID)
		ids := make([]string, 0, len(apis))
		for _, api := range apis {
			ids = append(ids, api.ID)
		}
		return ids
	}
	if m.structureActive() {
		if m.app == nil {
			return nil
		}
		apis := m.app.StructureRelatedAPIs(m.currentStructure.ID)
		ids := make([]string, 0, len(apis))
		for _, api := range apis {
			ids = append(ids, api.ID)
		}
		return ids
	}
	if api, ok := m.currentEntry(); ok {
		return api.RelatedAPIs
	}
	return nil
}

// currentReferences returns the reference list and display name for whichever
// entry the detail pane is showing.
func (m Model) currentReferences() (string, []catalog.Reference, bool) {
	if m.statusActive() {
		return m.currentStatus.Name, m.currentStatus.References, true
	}
	if m.constantActive() {
		return m.currentConstant.Name, m.currentConstant.References, true
	}
	if m.structureActive() {
		return m.currentStructure.Name, m.currentStructure.References, true
	}
	if api, ok := m.currentEntry(); ok {
		return api.Name, api.References, true
	}
	return "", nil, false
}

func (m *Model) currentEntry() (catalog.API, bool) {
	if api, ok := m.selectedAPI(); ok {
		return api, true
	}
	if m.hasCurrent {
		return m.current, true
	}
	return catalog.API{}, false
}

func (m *Model) toggleFavorite() {
	if m.structureActive() || m.constantActive() || m.statusActive() {
		m.status = "Favorites apply to API entries only."
		return
	}
	api, ok := m.currentEntry()
	if !ok {
		m.status = "Select an API to favorite."
		return
	}
	if m.app == nil {
		return
	}
	favorite, err := m.app.ToggleFavorite(api.ID)
	if err != nil {
		m.status = "Could not save favorites: " + err.Error()
		m.statusErr = true
		return
	}
	m.rebuildSidebarPreservingSelection()
	if favorite {
		m.status = "Added " + api.Name + " to favorites."
	} else {
		m.status = "Removed " + api.Name + " from favorites."
	}
	// Removing an entry while browsing favorites should update the list.
	item := m.sidebarItem()
	if item.Kind == sideFavorites {
		m.reloadSidebarList()
		m.listIndex = clamp(m.listIndex, 0, maxInt(0, len(m.apiList)-1))
		m.syncPreview()
	}
}

func (m Model) updateOverlay(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	keys := m.keys
	switch m.overlay {
	case overlayHelp:
		switch {
		case key.Matches(msg, keys.Escape, keys.Help, keys.Quit):
			m.overlay = overlayNone
		case key.Matches(msg, keys.Up):
			if m.helpOffset > 0 {
				m.helpOffset--
			}
		case key.Matches(msg, keys.Down):
			m.helpOffset++
		case key.Matches(msg, keys.PageUp):
			m.helpOffset = maxInt(0, m.helpOffset-8)
		case key.Matches(msg, keys.PageDown):
			m.helpOffset += 8
		}
		return m, nil

	case overlayRelated:
		related := m.currentRelatedIDs()
		switch {
		case key.Matches(msg, keys.Escape, keys.Related, keys.Quit):
			m.overlay = overlayNone
		case key.Matches(msg, keys.Up):
			if m.overlayIndex > 0 {
				m.overlayIndex--
			}
		case key.Matches(msg, keys.Down):
			if m.overlayIndex < len(related)-1 {
				m.overlayIndex++
			}
		case key.Matches(msg, keys.Enter):
			if len(related) == 0 {
				m.overlay = overlayNone
				return m, nil
			}
			id := related[clamp(m.overlayIndex, 0, len(related)-1)]
			m.navigateToAPI(id)
		}
		return m, nil

	case overlayReferences:
		switch {
		case key.Matches(msg, keys.Escape, keys.References, keys.Quit):
			m.overlay = overlayNone
		case key.Matches(msg, keys.Up):
			if m.refOffset > 0 {
				m.refOffset--
			}
		case key.Matches(msg, keys.Down):
			m.refOffset++
		}
		return m, nil

	case overlayExport:
		return m.updateExportOverlay(msg)

	case overlayCopy:
		return m.updateCopyOverlay(msg)
	}
	return m, nil
}

func (m *Model) openExport() {
	name, ok := m.currentEntryName()
	if !ok {
		m.status = "Select an API, structure, constant, or NTSTATUS value to export."
		return
	}
	m.exportTargets = []exportTarget{exportJSON, exportMarkdown}
	if !m.structureActive() && !m.constantActive() && !m.statusActive() && m.query != "" && len(m.results) > 0 {
		m.exportTargets = append(m.exportTargets, exportSearch)
	}
	m.exportIndex = 0
	m.exportConfirm = false
	m.pendingPath = ""
	m.pendingData = nil
	m.overlay = overlayExport
	m.status = "Export " + name
}

// currentEntryName returns the display name of the active detail entry.
func (m Model) currentEntryName() (string, bool) {
	if m.statusActive() {
		return m.currentStatus.Name, true
	}
	if m.constantActive() {
		return m.currentConstant.Name, true
	}
	if m.structureActive() {
		return m.currentStructure.Name, true
	}
	if api, ok := m.currentEntry(); ok {
		return api.Name, true
	}
	return "", false
}

func (m Model) updateExportOverlay(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	keys := m.keys
	if m.exportConfirm {
		switch {
		case key.Matches(msg, keys.Confirm):
			if err := export.WriteReplace(m.pendingPath, m.pendingData); err != nil {
				m.failExport(err)
				return m, nil
			}
			path := m.pendingPath
			m.finishExport(path)
		case key.Matches(msg, keys.Cancel, keys.Escape, keys.Quit):
			m.exportConfirm = false
			m.pendingPath = ""
			m.pendingData = nil
			m.status = "Export cancelled; choose another format or Esc to close."
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, keys.Escape, keys.Quit):
		m.overlay = overlayNone
	case key.Matches(msg, keys.Up):
		if m.exportIndex > 0 {
			m.exportIndex--
		}
	case key.Matches(msg, keys.Down):
		if m.exportIndex < len(m.exportTargets)-1 {
			m.exportIndex++
		}
	case key.Matches(msg, keys.Enter):
		m.runExport()
	}
	return m, nil
}

func (m *Model) runExport() {
	if m.exportIndex < 0 || m.exportIndex >= len(m.exportTargets) {
		return
	}
	target := m.exportTargets[m.exportIndex]
	path, data, err := m.buildExport(target)
	if err != nil {
		m.failExport(err)
		return
	}
	if err := export.WriteNew(path, data); err != nil {
		if errors.Is(err, export.ErrExists) {
			m.exportConfirm = true
			m.pendingPath = path
			m.pendingData = data
			m.status = path + " already exists. Overwrite? (y/n)"
			return
		}
		m.failExport(err)
		return
	}
	m.finishExport(path)
}

func (m *Model) buildExport(target exportTarget) (string, []byte, error) {
	catalogVersion := ""
	if m.app != nil && m.app.Catalog != nil {
		catalogVersion = m.app.Catalog.CatalogVersion
	}
	if m.statusActive() {
		status := m.currentStatus
		switch target {
		case exportMarkdown:
			name := export.StatusFilename(status, "md")
			return filepath.Join("exports", name), []byte(export.StatusMarkdown(status)), nil
		case exportJSON:
			data, err := export.StatusJSON(status, catalogVersion)
			name := export.StatusFilename(status, "json")
			return filepath.Join("exports", name), data, err
		default:
			return "", nil, errors.New("search results are only exported from API search")
		}
	}
	if m.constantActive() {
		constant := m.currentConstant
		switch target {
		case exportMarkdown:
			name := export.ConstantFilename(constant, "md")
			return filepath.Join("exports", name), []byte(export.ConstantMarkdown(constant)), nil
		case exportJSON:
			data, err := export.ConstantJSON(constant, catalogVersion)
			name := export.ConstantFilename(constant, "json")
			return filepath.Join("exports", name), data, err
		default:
			return "", nil, errors.New("search results are only exported from API search")
		}
	}
	if m.structureActive() {
		structure := m.currentStructure
		switch target {
		case exportMarkdown:
			name := export.StructureFilename(structure, "md")
			return filepath.Join("exports", name), []byte(export.StructureMarkdown(structure)), nil
		case exportJSON:
			data, err := export.StructureJSON(structure, catalogVersion)
			name := export.StructureFilename(structure, "json")
			return filepath.Join("exports", name), data, err
		default:
			return "", nil, errors.New("search results are only exported from API search")
		}
	}
	api, ok := m.currentEntry()
	if !ok {
		return "", nil, errors.New("no API selected")
	}
	switch target {
	case exportMarkdown:
		name := export.DefaultFilename(api, "md")
		return filepath.Join("exports", name), []byte(export.Markdown(api)), nil
	case exportSearch:
		data, err := export.ResultsJSON(m.query, m.results)
		return filepath.Join("exports", "search-results.json"), data, err
	default:
		data, err := export.APIJSON(api, catalogVersion)
		name := export.DefaultFilename(api, "json")
		return filepath.Join("exports", name), data, err
	}
}

// exportName labels a target in the export menu.
func (m Model) exportName(target exportTarget) string {
	if m.statusActive() {
		switch target {
		case exportMarkdown:
			return "NTSTATUS reference page (.md)"
		default:
			return "NTSTATUS record as JSON (.json)"
		}
	}
	if m.constantActive() {
		switch target {
		case exportMarkdown:
			return "Constant reference page (.md)"
		default:
			return "Constant record as JSON (.json)"
		}
	}
	if m.structureActive() {
		switch target {
		case exportMarkdown:
			return "Structure reference page (.md)"
		default:
			return "Structure record as JSON (.json)"
		}
	}
	switch target {
	case exportMarkdown:
		return "Markdown reference page (.md)"
	case exportSearch:
		return "Search results list (.json)"
	default:
		return "API record as JSON (.json)"
	}
}

func (m Model) exportDestination(target exportTarget) string {
	if m.statusActive() {
		status := m.currentStatus
		switch target {
		case exportMarkdown:
			return filepath.Join("exports", export.StatusFilename(status, "md"))
		default:
			return filepath.Join("exports", export.StatusFilename(status, "json"))
		}
	}
	if m.constantActive() {
		constant := m.currentConstant
		switch target {
		case exportMarkdown:
			return filepath.Join("exports", export.ConstantFilename(constant, "md"))
		default:
			return filepath.Join("exports", export.ConstantFilename(constant, "json"))
		}
	}
	if m.structureActive() {
		structure := m.currentStructure
		switch target {
		case exportMarkdown:
			return filepath.Join("exports", export.StructureFilename(structure, "md"))
		default:
			return filepath.Join("exports", export.StructureFilename(structure, "json"))
		}
	}
	api, ok := m.currentEntry()
	if !ok {
		return "exports/"
	}
	switch target {
	case exportMarkdown:
		return filepath.Join("exports", export.DefaultFilename(api, "md"))
	case exportSearch:
		return filepath.Join("exports", "search-results.json")
	default:
		return filepath.Join("exports", export.DefaultFilename(api, "json"))
	}
}

func (m *Model) finishExport(path string) {
	m.overlay = overlayNone
	m.exportConfirm = false
	m.pendingPath = ""
	m.pendingData = nil
	m.status = "Exported to " + path
	m.statusErr = false
}

func (m *Model) failExport(err error) {
	m.overlay = overlayExport
	m.status = "Export failed: " + err.Error()
	m.statusErr = true
}
