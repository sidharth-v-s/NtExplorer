package tui

import (
	"fmt"

	"ntexplorer/internal/catalog"
)

// buildSidebar rebuilds the sidebar entries and their counts.
func (m *Model) buildSidebar() {
	items := []sidebarItem{{Label: "APIS", Header: true}}
	allCount := 0
	categoryCounts := map[string]int{}
	tagCount := 0
	if m.app != nil && m.app.Catalog != nil {
		allCount = len(m.app.Catalog.APIs)
		for _, api := range m.app.Catalog.APIs {
			categoryCounts[api.Category]++
		}
		tagCount = len(m.app.Catalog.Tags)
	}
	favorites, recent := 0, 0
	if m.app != nil {
		favorites = len(m.app.State.Favorites)
		recent = len(m.app.State.Recent)
	}

	items = append(items,
		sidebarItem{Label: "All APIs", Kind: sideAll, Count: allCount},
		sidebarItem{Label: "Favorites", Kind: sideFavorites, Count: favorites},
		sidebarItem{Label: "Recently Viewed", Kind: sideRecent, Count: recent},
	)
	if m.app != nil && m.app.Catalog != nil {
		for _, category := range m.app.Catalog.Categories {
			items = append(items, sidebarItem{
				Label:    category.Name,
				Kind:     sideCategory,
				Category: category.Name,
				Count:    categoryCounts[category.Name],
			})
		}
	}
	structureCount := 0
	constantCount := 0
	if m.app != nil && m.app.Catalog != nil {
		structureCount = len(m.app.Catalog.Structures)
		constantCount = len(m.app.Catalog.Constants)
	}
	items = append(items,
		sidebarItem{Label: "LIBRARY", Header: true},
		sidebarItem{Label: "MalDev Tags", Kind: sideTags, Count: tagCount},
		sidebarItem{Label: "Structures", Kind: sideStructures, Count: structureCount},
		sidebarItem{Label: "Constants", Kind: sideConstants, Count: constantCount},
		sidebarItem{Label: "NTSTATUS", Kind: sideSoon, Count: 0},
	)
	m.sidebar = items
}

// selectInitialSidebar picks the configured category, then the last browsed
// category, then "All APIs".
func (m *Model) selectInitialSidebar() {
	target := ""
	if m.app != nil {
		target = m.app.Config.InitialCategory
		if target == "" {
			target = m.app.State.LastCategory
		}
	}
	index := -1
	for i, item := range m.sidebar {
		if !item.Header && item.Kind == sideCategory && item.Category == target {
			index = i
			break
		}
	}
	if index < 0 {
		for i, item := range m.sidebar {
			// Header rows use the zero value of sidebarKind, so they must be
			// excluded explicitly rather than compared by Kind alone.
			if !item.Header && item.Kind == sideAll {
				index = i
				break
			}
		}
	}
	if index < 0 {
		// Never start on a header row.
		for i, item := range m.sidebar {
			if !item.Header {
				index = i
				break
			}
		}
	}
	if index < 0 {
		index = 0
	}
	m.sidebarIndex = index
	m.focus = panelCategories
	m.loadSidebarContent()
}

// loadSidebarContent resets search and reloads the middle panel for the
// selected sidebar item.
func (m *Model) loadSidebarContent() {
	m.resetSearchState()
	m.reloadSidebarList()
}

func (m *Model) resetSearchState() {
	m.query = ""
	m.results = nil
	m.searching = false
	m.preSearch = nil
	m.searchInput.SetValue("")
	m.searchInput.Blur()
}

// reloadSidebarList fills the middle panel from the selected sidebar item
// without touching search input state.
func (m *Model) reloadSidebarList() {
	m.mode = listBrowse
	m.apiList = nil
	m.results = nil
	m.structures = nil
	m.structureResults = nil
	m.constants = nil
	m.constantResults = nil
	m.listIndex = 0
	m.tagFilter = ""
	m.category = ""
	m.soonLabel = ""

	item := m.sidebarItem()
	switch item.Kind {
	case sideAll:
		if m.app != nil {
			m.apiList = m.app.Browse("")
		}
	case sideCategory:
		m.category = item.Category
		if m.app != nil {
			m.apiList = m.app.Browse(item.Category)
			m.app.RememberCategory(item.Category)
		}
	case sideFavorites:
		if m.app != nil {
			m.apiList = m.app.FavoriteAPIs()
		}
	case sideRecent:
		if m.app != nil {
			m.apiList = m.app.RecentAPIs()
		}
	case sideTags:
		m.mode = listTags
		if m.app != nil {
			m.tags = m.app.Tags()
		}
		if m.tagIndex >= len(m.tags) {
			m.tagIndex = 0
		}
	case sideStructures:
		m.mode = listStructures
		if m.app != nil {
			m.structures = m.app.BrowseStructures()
		}
	case sideConstants:
		m.mode = listConstants
		if m.app != nil {
			m.constants = m.app.BrowseConstants()
		}
	case sideSoon:
		m.mode = listSoon
		m.soonLabel = item.Label
	}
	m.syncPreview()
}

// moveSidebar moves to the next selectable sidebar row.
func (m *Model) moveSidebar(delta int) {
	index := m.sidebarIndex
	for {
		index += delta
		if index < 0 || index >= len(m.sidebar) {
			return
		}
		if !m.sidebar[index].Header {
			break
		}
	}
	if index != m.sidebarIndex {
		m.sidebarIndex = index
		m.loadSidebarContent()
		m.status = ""
	}
}

func (m *Model) moveList(delta int) {
	if m.mode == listTags {
		m.moveTag(delta)
		return
	}
	if m.mode == listSoon {
		return
	}
	count := m.listLength()
	if count == 0 {
		return
	}
	index := clamp(m.listIndex+delta, 0, count-1)
	if index != m.listIndex {
		m.listIndex = index
		m.syncPreview()
		m.status = ""
	}
}

func (m *Model) jumpList(toEnd bool) {
	count := m.listLength()
	if count == 0 {
		return
	}
	index := 0
	if toEnd {
		index = count - 1
	}
	if index != m.listIndex {
		m.listIndex = index
		m.syncPreview()
		m.status = ""
	}
}

func (m *Model) listLength() int {
	switch m.mode {
	case listSearch:
		return len(m.results)
	case listBrowse:
		return len(m.apiList)
	case listTags:
		return len(m.tags)
	case listStructures:
		return len(m.structures)
	case listStructureSearch:
		return len(m.structureResults)
	case listConstants:
		return len(m.constants)
	case listConstantSearch:
		return len(m.constantResults)
	}
	return 0
}

func (m *Model) moveTag(delta int) {
	if len(m.tags) == 0 {
		return
	}
	index := clamp(m.tagIndex+delta, 0, len(m.tags)-1)
	if index != m.tagIndex {
		m.tagIndex = index
		m.status = ""
	}
}

// cycleFocus moves between the visible panels.
func (m *Model) cycleFocus(delta int) {
	panels := m.panelsVisible()
	current := 0
	for i, p := range panels {
		if p == m.focus {
			current = i
			break
		}
	}
	next := (current + delta + len(panels)) % len(panels)
	m.focus = panels[next]
}

// openSelected opens the highlighted API or applies the highlighted tag.
func (m *Model) openSelected() {
	switch m.mode {
	case listTags:
		m.applySelectedTag()
		return
	case listSoon:
		m.status = m.soonMessage()
		return
	case listStructures, listStructureSearch:
		structure, ok := m.selectedStructure()
		if !ok {
			m.status = "Nothing selected."
			return
		}
		m.focus = panelDetail
		m.setStructureDetail(structure)
		m.status = "Reading " + structure.Name
		return
	case listConstants, listConstantSearch:
		constant, ok := m.selectedConstant()
		if !ok {
			m.status = "Nothing selected."
			return
		}
		m.focus = panelDetail
		m.setConstantDetail(constant)
		m.status = "Reading " + constant.Name
		return
	}
	api, ok := m.selectedAPI()
	if !ok {
		m.status = "Nothing selected."
		return
	}
	m.focus = panelDetail
	m.setDetail(api)
	m.logView(api.ID)
	m.status = "Reading " + api.Name
}

func (m *Model) applySelectedTag() {
	tag, ok := m.selectedTag()
	if !ok {
		return
	}
	m.mode = listBrowse
	m.tagFilter = tag
	m.category = ""
	m.apiList = nil
	if m.app != nil {
		m.apiList = m.app.BrowseTag(tag)
	}
	m.listIndex = 0
	m.focus = panelList
	m.syncPreview()
	m.status = fmt.Sprintf("%d APIs tagged %q · Esc tags", len(m.apiList), tag)
}

func (m *Model) soonMessage() string {
	return "The " + m.soonLabel + " browser is planned for a later release. This release covers APIs, Native APIs, structures, and constants."
}

// setDetail updates the current API entry. Scroll position resets only when the
// entry itself changes.
func (m *Model) setDetail(api catalog.API) {
	if !m.hasCurrent || m.current.ID != api.ID {
		m.detail.GotoTop()
	}
	m.current = api
	m.hasCurrent = true
	m.currentStructure = catalog.Structure{}
	m.hasStructure = false
	m.currentConstant = catalog.Constant{}
	m.hasConstant = false
	m.refreshDetail()
}

// setStructureDetail updates the current structure entry.
func (m *Model) setStructureDetail(structure catalog.Structure) {
	if !m.hasStructure || m.currentStructure.ID != structure.ID {
		m.detail.GotoTop()
	}
	m.currentStructure = structure
	m.hasStructure = true
	m.current = catalog.API{}
	m.hasCurrent = false
	m.currentConstant = catalog.Constant{}
	m.hasConstant = false
	m.refreshDetail()
}

// setConstantDetail updates the current constant entry.
func (m *Model) setConstantDetail(constant catalog.Constant) {
	if !m.hasConstant || m.currentConstant.ID != constant.ID {
		m.detail.GotoTop()
	}
	m.currentConstant = constant
	m.hasConstant = true
	m.current = catalog.API{}
	m.hasCurrent = false
	m.currentStructure = catalog.Structure{}
	m.hasStructure = false
	m.refreshDetail()
}

// apiName resolves an ID for display, falling back to the ID itself.
func (m Model) apiName(id string) string {
	if m.app != nil {
		if api, ok := m.app.API(id); ok {
			return api.Name
		}
	}
	return id
}

func (m *Model) clearDetail() {
	m.hasCurrent = false
	m.current = catalog.API{}
	m.hasStructure = false
	m.currentStructure = catalog.Structure{}
	m.hasConstant = false
	m.currentConstant = catalog.Constant{}
	m.detail.SetContent("")
	m.detail.GotoTop()
}

func (m *Model) syncPreview() {
	if constant, ok := m.selectedConstant(); ok {
		m.setConstantDetail(constant)
		return
	}
	if structure, ok := m.selectedStructure(); ok {
		m.setStructureDetail(structure)
		return
	}
	if api, ok := m.selectedAPI(); ok {
		m.setDetail(api)
		return
	}
	m.clearDetail()
}

func (m *Model) refreshDetail() {
	if m.detail.Width <= 0 {
		return
	}
	switch {
	case m.hasConstant:
		m.detail.SetContent(m.renderConstantDetail(m.currentConstant))
	case m.hasStructure:
		m.detail.SetContent(m.renderStructureDetail(m.currentStructure))
	case m.hasCurrent:
		m.detail.SetContent(m.renderDetail(m.current))
	}
}

// structureActive reports whether the detail pane is showing a structure
// rather than an API or constant.
func (m Model) structureActive() bool {
	return m.hasStructure && !m.hasCurrent && !m.hasConstant
}

// constantActive reports whether the detail pane is showing a constant.
func (m Model) constantActive() bool {
	return m.hasConstant && !m.hasCurrent && !m.hasStructure
}

func (m *Model) logView(id string) {
	if m.app == nil {
		return
	}
	if err := m.app.RecordRecent(id); err != nil {
		m.status = "Could not save recent items: " + err.Error()
		m.statusErr = true
	}
	m.rebuildSidebarPreservingSelection()
}

// rebuildSidebarPreservingSelection refreshes counts without moving the
// user's sidebar selection.
func (m *Model) rebuildSidebarPreservingSelection() {
	selected := m.sidebarItem()
	m.buildSidebar()
	for i, item := range m.sidebar {
		if item.Kind == selected.Kind && item.Category == selected.Category && item.Label == selected.Label {
			m.sidebarIndex = i
			return
		}
	}
}

// navigateToAPI jumps to an API by ID, used by related-entry navigation.
func (m *Model) navigateToAPI(id string) {
	api, ok := m.app.API(id)
	if !ok {
		m.status = "Related API not found in this catalog."
		m.statusErr = true
		return
	}
	for i, item := range m.sidebar {
		if item.Kind == sideAll {
			m.sidebarIndex = i
			break
		}
	}
	m.loadSidebarContent()
	m.setDetail(*api)
	for i := range m.apiList {
		if m.apiList[i].ID == id {
			m.listIndex = i
			break
		}
	}
	m.focus = panelDetail
	m.overlay = overlayNone
	m.logView(id)
	m.status = "Opened " + api.Name
}

// openStructures switches to the Structures view. When an API is selected and
// documents related structures, the first one is opened directly.
func (m *Model) openStructures() {
	targetID := ""
	if api, ok := m.currentEntry(); ok && len(api.RelatedStructures) > 0 {
		targetID = api.RelatedStructures[0]
	}
	found := false
	for i, item := range m.sidebar {
		if item.Kind == sideStructures {
			m.sidebarIndex = i
			found = true
			break
		}
	}
	if !found {
		m.status = "The structure catalog is unavailable."
		m.statusErr = true
		return
	}
	m.loadSidebarContent()
	if targetID != "" {
		for i, structure := range m.structures {
			if structure.ID == targetID {
				m.listIndex = i
				m.syncPreview()
				m.focus = panelDetail
				m.status = "Reading " + structure.Name
				return
			}
		}
	}
	m.focus = panelList
	m.status = "Structures · Enter opens · / searches structures"
}

// openConstants switches to the Constants view. When an API is selected and
// documents related constants, the first one is opened directly.
func (m *Model) openConstants() {
	targetID := ""
	if api, ok := m.currentEntry(); ok && len(api.RelatedConstants) > 0 {
		targetID = api.RelatedConstants[0]
	}
	found := false
	for i, item := range m.sidebar {
		if item.Kind == sideConstants {
			m.sidebarIndex = i
			found = true
			break
		}
	}
	if !found {
		m.status = "The constant catalog is unavailable."
		m.statusErr = true
		return
	}
	m.loadSidebarContent()
	if targetID != "" {
		for i, constant := range m.constants {
			if constant.ID == targetID {
				m.listIndex = i
				m.syncPreview()
				m.focus = panelDetail
				m.status = "Reading " + constant.Name
				return
			}
		}
	}
	m.focus = panelList
	m.status = "Constants · Enter opens · / searches constants"
}

// resize recalculates panel and viewport sizes.
func (m *Model) resize() {
	l := m.computeLayout()
	m.detail.Width = maxInt(1, l.detailW-4)
	// Detail panel: border (2) + heading (1) leaves the viewport height.
	m.detail.Height = maxInt(1, l.detailH-3)
	m.searchInput.Width = maxInt(8, l.listW-7)
	m.refreshDetail()
}

type layout struct {
	bodyH          int
	sidebarVisible bool
	stacked        bool
	sidebarW       int
	listW          int
	detailW        int
	listH          int
	detailH        int
}

func (m Model) computeLayout() layout {
	bodyH := m.height - 2
	if bodyH < 3 {
		bodyH = 3
	}
	l := layout{bodyH: bodyH}
	width := m.width
	switch {
	case width >= minWideLayout:
		l.sidebarVisible = true
		l.sidebarW = sidebarWidth
		l.listW = listWidth
		l.detailW = width - sidebarWidth - listWidth
		l.listH = bodyH
		l.detailH = bodyH
	case width >= minThreePanels:
		l.listW = listWidth
		l.detailW = width - listWidth
		l.listH = bodyH
		l.detailH = bodyH
	default:
		l.stacked = true
		l.listW = width
		l.detailW = width
		l.listH = bodyH / 3
		if l.listH < 8 {
			l.listH = 8
		}
		l.detailH = bodyH - l.listH
		if l.detailH < 5 {
			l.detailH = 5
			l.listH = bodyH - l.detailH
		}
		if l.listH < 3 {
			l.listH = 3
		}
	}
	if l.detailW < 10 {
		l.detailW = 10
	}
	if l.listW < 10 {
		l.listW = 10
	}
	return l
}

// visiblePanelsChanged keeps focus valid after a resize.
func (m *Model) ensureValidFocus() {
	for _, p := range m.panelsVisible() {
		if p == m.focus {
			return
		}
	}
	m.focus = panelList
}

func clamp(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
