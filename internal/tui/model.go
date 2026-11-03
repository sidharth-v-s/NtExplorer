// Package tui implements the Bubble Tea terminal interface. It contains no
// catalog parsing or search logic; all data access goes through internal/app.
package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"

	"ntexplorer/internal/app"
	"ntexplorer/internal/catalog"
	"ntexplorer/internal/clipboard"
	"ntexplorer/internal/search"
)

// Version is displayed in the header, help screen and exports.
const Version = "0.5.0"

const (
	minWideLayout  = 100
	minThreePanels = 72
	sidebarWidth   = 26
	listWidth      = 34
	minimumDetail  = 30
)

type panel int

const (
	panelCategories panel = iota
	panelList
	panelDetail
)

type overlay int

const (
	overlayNone overlay = iota
	overlayHelp
	overlayRelated
	overlayReferences
	overlayExport
	overlayCopy
)

type listMode int

const (
	listBrowse listMode = iota
	listSearch
	listTags
	listStructures
	listStructureSearch
	listConstants
	listConstantSearch
	listStatuses
	listStatusSearch
)

type sidebarKind int

const (
	sideAll sidebarKind = iota
	sideFavorites
	sideRecent
	sideCategory
	sideTags
	sideStructures
	sideConstants
	sideStatuses
)

type sidebarItem struct {
	Label    string
	Kind     sidebarKind
	Category string
	Count    int
	Header   bool
}

type exportTarget int

const (
	exportJSON exportTarget = iota
	exportMarkdown
	exportSearch
)

// Model is the Bubble Tea application model.
type Model struct {
	app    *app.App
	keys   KeyMap
	styles Styles

	width  int
	height int

	focus   panel
	overlay overlay

	sidebar      []sidebarItem
	sidebarIndex int

	mode             listMode
	category         string
	tagFilter        string
	apiList          []catalog.API
	results          []search.Result
	structures       []catalog.Structure
	structureResults []search.StructureResult
	constants        []catalog.Constant
	constantResults  []search.ConstantResult
	statuses         []catalog.NTStatus
	statusResults    []search.StatusResult
	listIndex        int
	tags             []catalog.TagCount
	tagIndex         int

	searching        bool
	searchStructures bool
	searchConstants  bool
	searchStatuses   bool
	searchInput      textinput.Model
	query            string
	preSearch        *browseState

	current          catalog.API
	hasCurrent       bool
	currentStructure catalog.Structure
	hasStructure     bool
	currentConstant  catalog.Constant
	hasConstant      bool
	currentStatus    catalog.NTStatus
	hasStatus        bool
	detail           viewport.Model

	overlayIndex  int
	helpOffset    int
	refOffset     int
	exportTargets []exportTarget
	exportIndex   int
	exportConfirm bool
	pendingPath   string
	pendingData   []byte

	copyTargets []copyTarget
	copyIndex   int
	copyWriter  func(string) error

	status    string
	statusErr bool

	showExamples bool
}

// New creates the initial TUI model.
func New(a *app.App) Model {
	m := Model{
		app:          a,
		keys:         DefaultKeyMap(),
		styles:       NewStyles(),
		focus:        panelCategories,
		detail:       viewport.New(0, 0),
		showExamples: true,
		copyWriter:   clipboard.Copy,
	}
	if a != nil && a.Config.ShowExamples != nil {
		m.showExamples = *a.Config.ShowExamples
	}

	m.searchInput = textinput.New()
	m.searchInput.Placeholder = "name, tag, category, or description"
	m.searchInput.Prompt = "/ "
	m.searchInput.CharLimit = 100

	m.buildSidebar()
	m.selectInitialSidebar()
	if a != nil && len(a.Warnings) > 0 {
		m.status = strings.Join(a.Warnings, " · ")
	}
	return m
}

// App exposes the application layer (used by tests).
func (m Model) App() *app.App { return m.app }

// ModeName is a stable label for tests and debugging.
func (m Model) ModeName() string {
	switch m.mode {
	case listSearch:
		return "search"
	case listTags:
		return "tags"
	case listStructures:
		return "structures"
	case listStructureSearch:
		return "structure-search"
	case listConstants:
		return "constants"
	case listConstantSearch:
		return "constant-search"
	case listStatuses:
		return "statuses"
	case listStatusSearch:
		return "status-search"
	default:
		return "browse"
	}
}

func (m Model) sidebarItem() sidebarItem {
	if m.sidebarIndex < 0 || m.sidebarIndex >= len(m.sidebar) {
		return sidebarItem{Label: "All APIs", Kind: sideAll}
	}
	return m.sidebar[m.sidebarIndex]
}

func (m Model) selectedAPI() (catalog.API, bool) {
	switch m.mode {
	case listSearch:
		if m.listIndex >= 0 && m.listIndex < len(m.results) {
			return m.results[m.listIndex].API, true
		}
	case listBrowse:
		if m.listIndex >= 0 && m.listIndex < len(m.apiList) {
			return m.apiList[m.listIndex], true
		}
	}
	return catalog.API{}, false
}

func (m Model) selectedStructure() (catalog.Structure, bool) {
	switch m.mode {
	case listStructures:
		if m.listIndex >= 0 && m.listIndex < len(m.structures) {
			return m.structures[m.listIndex], true
		}
	case listStructureSearch:
		if m.listIndex >= 0 && m.listIndex < len(m.structureResults) {
			return m.structureResults[m.listIndex].Structure, true
		}
	}
	return catalog.Structure{}, false
}

func (m Model) selectedConstant() (catalog.Constant, bool) {
	switch m.mode {
	case listConstants:
		if m.listIndex >= 0 && m.listIndex < len(m.constants) {
			return m.constants[m.listIndex], true
		}
	case listConstantSearch:
		if m.listIndex >= 0 && m.listIndex < len(m.constantResults) {
			return m.constantResults[m.listIndex].Constant, true
		}
	}
	return catalog.Constant{}, false
}

func (m Model) selectedStatus() (catalog.NTStatus, bool) {
	switch m.mode {
	case listStatuses:
		if m.listIndex >= 0 && m.listIndex < len(m.statuses) {
			return m.statuses[m.listIndex], true
		}
	case listStatusSearch:
		if m.listIndex >= 0 && m.listIndex < len(m.statusResults) {
			return m.statusResults[m.listIndex].Status, true
		}
	}
	return catalog.NTStatus{}, false
}

func (m Model) selectedTag() (string, bool) {
	if m.mode == listTags && m.tagIndex >= 0 && m.tagIndex < len(m.tags) {
		return m.tags[m.tagIndex].Tag, true
	}
	return "", false
}

func (m Model) panelsVisible() []panel {
	panels := []panel{panelList, panelDetail}
	if m.width >= minThreePanels {
		panels = append([]panel{panelCategories}, panels...)
	}
	return panels
}

func (m Model) categoryLabel() string {
	switch {
	case m.tagFilter != "":
		return "tag: " + m.tagFilter
	case m.category != "":
		return m.category
	case m.mode == listSearch:
		return "search results"
	case m.mode == listStructures:
		return "structures"
	case m.mode == listStructureSearch:
		return "structure search"
	case m.mode == listConstants:
		return "constants"
	case m.mode == listConstantSearch:
		return "constant search"
	case m.mode == listStatuses:
		return "NTSTATUS"
	case m.mode == listStatusSearch:
		return "NTSTATUS search"
	case m.sidebarItem().Kind == sideFavorites:
		return "favorites"
	case m.sidebarItem().Kind == sideRecent:
		return "recently viewed"
	default:
		return "all APIs"
	}
}
