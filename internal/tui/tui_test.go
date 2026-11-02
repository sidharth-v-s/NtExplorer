package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"ntexplorer/data"
	"ntexplorer/internal/app"
	"ntexplorer/internal/catalog"
	"ntexplorer/internal/config"
)

func newTestModel(t *testing.T) Model {
	t.Helper()
	t.Setenv("NTEXPLORER_CONFIG_DIR", t.TempDir())
	cat, _, err := catalog.Load(data.FS, ".")
	if err != nil {
		t.Fatalf("load embedded catalog: %v", err)
	}
	a := app.New(cat, config.Default(), nil, nil)
	m := New(a)
	m.width, m.height = 140, 45
	m.resize()
	m.ensureValidFocus()
	return m
}

func send(m Model, msg tea.Msg) Model {
	next, _ := m.Update(msg)
	return next.(Model)
}

func runes(text string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text)}
}

func press(m Model, keys ...tea.KeyMsg) Model {
	for _, k := range keys {
		m = send(m, k)
	}
	return m
}

func searchFor(t *testing.T, m Model, query string) Model {
	t.Helper()
	m = send(m, runes("/"))
	m = send(m, runes(query))
	if m.mode != listSearch {
		t.Fatalf("expected search mode, got %s", m.ModeName())
	}
	return m
}

func selectStructures(t *testing.T, m Model) Model {
	t.Helper()
	for i, item := range m.sidebar {
		if item.Kind == sideStructures {
			m.sidebarIndex = i
			m.loadSidebarContent()
			return m
		}
	}
	t.Fatal("structures sidebar entry not found")
	return m
}

func gotoStructures(t *testing.T, m Model) Model {
	t.Helper()
	for m.sidebarItem().Kind != sideStructures {
		before := m.sidebarIndex
		m = send(m, runes("j"))
		if m.sidebarIndex == before {
			t.Fatal("structures sidebar entry not found")
		}
	}
	if m.mode != listStructures {
		t.Fatalf("expected structure list, got %s", m.ModeName())
	}
	return m
}

func TestStructuresViewAndSearch(t *testing.T) {
	m := newTestModel(t)
	m = gotoStructures(t, m)
	if len(m.structures) < 10 {
		t.Fatalf("expected the bundled structures, got %d", len(m.structures))
	}
	if len(m.currentStructure.Name) == 0 {
		t.Fatal("structure list should preview the first structure")
	}
	view := m.View()
	for _, want := range []string{"STRUCTURES", "STARTUPINFOW", "STRUCTURE"} {
		if !strings.Contains(view, want) {
			t.Errorf("structure view missing %q:\n%s", want, view)
		}
	}

	m = send(m, runes("/"))
	m = send(m, runes("hProcess"))
	if m.mode != listStructureSearch {
		t.Fatalf("expected structure search mode, got %s", m.ModeName())
	}
	if len(m.structureResults) == 0 || m.structureResults[0].Structure.Name != "PROCESS_INFORMATION" {
		t.Fatalf("unexpected structure results: %+v", m.structureResults)
	}
	if !strings.Contains(m.View(), "STRUCTURE SEARCH") {
		t.Fatalf("structure search heading missing:\n%s", m.View())
	}

	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.hasStructure || m.currentStructure.Name != "PROCESS_INFORMATION" {
		t.Fatalf("Enter did not open the structure: %+v", m.currentStructure)
	}
	if m.hasCurrent {
		t.Fatal("API and structure detail state must be mutually exclusive")
	}
	detail := m.renderStructureDetail(m.currentStructure)
	for _, want := range []string{"DEFINITION", "MEMBERS", "hProcess", "RELATED APIS", "CreateProcessW"} {
		if !strings.Contains(detail, want) {
			t.Errorf("structure detail missing %q:\n%s", want, detail)
		}
	}
}

func TestStructureRelatedOverlayAndReverseLinks(t *testing.T) {
	m := newTestModel(t)
	m = gotoStructures(t, m)
	m = send(m, runes("/"))
	m = send(m, runes("STARTUPINFOW"))
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.currentStructure.Name != "STARTUPINFOW" {
		t.Fatalf("wrong structure opened: %s", m.currentStructure.Name)
	}

	m = send(m, runes("g"))
	if m.overlay != overlayRelated {
		t.Fatalf("expected related overlay, got %v", m.overlay)
	}
	if !strings.Contains(m.View(), "CreateProcessW") {
		t.Fatalf("reverse API links missing:\n%s", m.View())
	}
}

func TestStructureKeyFromAPIDetail(t *testing.T) {
	m := newTestModel(t)
	m = searchFor(t, m, "CreateProcessW")
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = send(m, runes("s"))
	if m.mode != listStructures {
		t.Fatalf("s should open structures, got %s", m.ModeName())
	}
	if !m.hasStructure || m.currentStructure.Name != "STARTUPINFOW" {
		t.Fatalf("first related structure not opened: %+v", m.currentStructure)
	}
	if m.hasCurrent {
		t.Fatal("API detail state should be cleared when opening a structure")
	}
}

func TestStructureFavoriteIsDisabled(t *testing.T) {
	m := newTestModel(t)
	m = gotoStructures(t, m)
	before := len(m.app.State.Favorites)
	m = send(m, runes("f"))
	if got := len(m.app.State.Favorites); got != before {
		t.Fatalf("favorite count changed from %d to %d", before, got)
	}
	if !strings.Contains(m.status, "API entries only") {
		t.Fatalf("unexpected status: %q", m.status)
	}
}

func TestStructureSearchNoResults(t *testing.T) {
	m := newTestModel(t)
	m = gotoStructures(t, m)
	m = send(m, runes("/"))
	m = send(m, runes("zzzznotastructure"))
	if len(m.structureResults) != 0 {
		t.Fatalf("unexpected structure results: %d", len(m.structureResults))
	}
	if !strings.Contains(m.View(), "No matches") {
		t.Fatalf("empty structure state not rendered:\n%s", m.View())
	}
}

func TestInitialView(t *testing.T) {
	m := newTestModel(t)
	view := m.View()
	for _, want := range []string{"NtExplorer", "CATEGORIES", "API FUNCTIONS", "API DETAILS", "Memory Management"} {
		if !strings.Contains(view, want) {
			t.Errorf("initial view missing %q", want)
		}
	}
	if m.focus != panelCategories {
		t.Fatalf("initial focus should be categories, got %v", m.focus)
	}
}

func TestSearchOpenAndScroll(t *testing.T) {
	m := newTestModel(t)
	m = searchFor(t, m, "VirtualAllocEx")
	view := m.View()
	if !strings.Contains(view, "SEARCH RESULTS") {
		t.Fatalf("search results panel not shown:\n%s", view)
	}
	if !strings.Contains(view, "VirtualAllocEx") {
		t.Fatalf("expected VirtualAllocEx in results:\n%s", view)
	}

	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.focus != panelDetail {
		t.Fatalf("Enter should focus detail, got %v", m.focus)
	}
	if !m.hasCurrent || m.current.Name != "VirtualAllocEx" {
		t.Fatalf("wrong current entry: %+v", m.current.Name)
	}
	detail := m.renderDetail(m.current)
	if !strings.Contains(detail, "PARAMETERS") || !strings.Contains(detail, "PROTOTYPE") {
		t.Fatalf("detail missing sections:\n%s", detail)
	}

	before := m.detail.YOffset
	for i := 0; i < 12; i++ {
		m = send(m, runes("j"))
	}
	if m.detail.YOffset <= before {
		t.Fatalf("detail did not scroll: before %d after %d", before, m.detail.YOffset)
	}
}

func TestEscapeClearsSearch(t *testing.T) {
	m := newTestModel(t)
	m = searchFor(t, m, "VirtualAlloc")
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = send(m, tea.KeyMsg{Type: tea.KeyEsc}) // detail -> list
	if m.query == "" {
		t.Fatal("first Esc should keep the query")
	}
	m = send(m, tea.KeyMsg{Type: tea.KeyEsc}) // clear search
	if m.query != "" || m.mode != listBrowse {
		t.Fatalf("search not cleared: query=%q mode=%s", m.query, m.ModeName())
	}
}

func TestSearchNoResults(t *testing.T) {
	m := newTestModel(t)
	m = searchFor(t, m, "zzzzzznotanapi")
	if len(m.results) != 0 {
		t.Fatalf("unexpected results: %d", len(m.results))
	}
	if !strings.Contains(m.View(), "No matches") {
		t.Fatalf("empty state not rendered:\n%s", m.View())
	}
}

func TestTabCyclesVisiblePanels(t *testing.T) {
	m := newTestModel(t)
	if m.focus != panelCategories {
		t.Fatal("expected to start in categories")
	}
	m = send(m, tea.KeyMsg{Type: tea.KeyTab})
	if m.focus != panelList {
		t.Fatalf("Tab should focus list, got %v", m.focus)
	}
	m = send(m, tea.KeyMsg{Type: tea.KeyTab})
	if m.focus != panelDetail {
		t.Fatalf("Tab should focus detail, got %v", m.focus)
	}
	m = send(m, tea.KeyMsg{Type: tea.KeyTab})
	if m.focus != panelCategories {
		t.Fatalf("Tab should wrap to categories, got %v", m.focus)
	}

	// Narrow layout hides the categories panel, so focus must skip it.
	m2 := newTestModel(t)
	m2.width, m2.height = 60, 30
	m2.resize()
	m2.ensureValidFocus()
	m2.focus = panelList
	m2 = send(m2, tea.KeyMsg{Type: tea.KeyShiftTab})
	if m2.focus != panelDetail {
		t.Fatalf("narrow Shift+Tab should go to detail, got %v", m2.focus)
	}
}

func TestHelpOverlay(t *testing.T) {
	m := newTestModel(t)
	m = send(m, runes("?"))
	if m.overlay != overlayHelp {
		t.Fatalf("expected help overlay, got %v", m.overlay)
	}
	if !strings.Contains(m.View(), "HELP") {
		t.Fatalf("help content missing:\n%s", m.View())
	}
	m = send(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.overlay != overlayNone {
		t.Fatal("Esc should close help")
	}
}

func TestRelatedAndReferencesOverlays(t *testing.T) {
	m := newTestModel(t)
	m = searchFor(t, m, "VirtualAlloc")
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})

	m = send(m, runes("g"))
	if m.overlay != overlayRelated {
		t.Fatalf("expected related overlay, got %v", m.overlay)
	}
	if !strings.Contains(m.View(), "RELATED") {
		t.Fatalf("related overlay missing:\n%s", m.View())
	}
	m = send(m, tea.KeyMsg{Type: tea.KeyEsc})

	m = send(m, runes("r"))
	if m.overlay != overlayReferences {
		t.Fatalf("expected references overlay, got %v", m.overlay)
	}
	if !strings.Contains(m.View(), "REFERENCES") || !strings.Contains(m.View(), "learn.microsoft.com") {
		t.Fatalf("references overlay missing:\n%s", m.View())
	}
}

func TestFavoriteToggle(t *testing.T) {
	m := newTestModel(t)
	m = searchFor(t, m, "VirtualAlloc")
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	id := m.current.ID

	m = send(m, runes("f"))
	if !m.app.IsFavorite(id) {
		t.Fatal("favorite was not added")
	}
	m = send(m, runes("f"))
	if m.app.IsFavorite(id) {
		t.Fatal("favorite was not removed")
	}
}

func TestCategoryNavigationSkipsHeaders(t *testing.T) {
	m := newTestModel(t)
	if m.sidebarItem().Header {
		t.Fatalf("initial sidebar selection is a header row: %+v", m.sidebarItem())
	}
	// All APIs -> Favorites -> Recently Viewed -> Process Management.
	m = press(m, runes("j"), runes("j"), runes("j"))
	if m.category != "Process Management" {
		t.Fatalf("expected Process Management, got %q (sidebar index %d)", m.category, m.sidebarIndex)
	}
	if len(m.apiList) == 0 || m.apiList[0].Category != "Process Management" {
		t.Fatalf("category list not loaded")
	}
}

func TestTagsViewAndFilter(t *testing.T) {
	m := newTestModel(t)
	for m.sidebarItem().Kind != sideTags {
		m = send(m, runes("j"))
		if m.sidebarIndex >= len(m.sidebar)-1 {
			t.Fatal("tags entry not found")
		}
	}
	// Enter in the sidebar moves focus to the tag list; a second Enter
	// applies the highlighted tag.
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.focus != panelList || m.mode != listTags {
		t.Fatalf("expected tag list focus, got focus=%v mode=%s", m.focus, m.ModeName())
	}
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != listBrowse || m.tagFilter == "" {
		t.Fatalf("tag filter not applied: mode=%s tag=%q", m.ModeName(), m.tagFilter)
	}
	if len(m.apiList) == 0 {
		t.Fatal("tag filter returned no APIs")
	}
	for _, api := range m.apiList {
		found := false
		for _, tag := range api.Tags {
			if tag == m.tagFilter {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s does not carry %q", api.Name, m.tagFilter)
		}
	}
	m = send(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != listTags || m.tagFilter != "" {
		t.Fatalf("Esc should return to the tag list, got mode=%s tag=%q", m.ModeName(), m.tagFilter)
	}
}

func TestLayoutFillsTerminal(t *testing.T) {
	for _, size := range []struct{ w, h int }{
		{140, 45}, {100, 30}, {80, 24}, {60, 20}, {40, 16},
	} {
		m := newTestModel(t)
		m.width, m.height = size.w, size.h
		m.resize()
		m.ensureValidFocus()
		lines := strings.Split(m.View(), "\n")
		if len(lines) != size.h {
			t.Errorf("%dx%d: rendered %d lines, want %d", size.w, size.h, len(lines), size.h)
		}
		for i, line := range lines {
			if got := lipgloss.Width(line); got > size.w {
				t.Errorf("%dx%d: line %d is %d columns wide: %q", size.w, size.h, i, got, line)
			}
		}
	}
}

func TestStructureLayoutFillsTerminal(t *testing.T) {
	for _, size := range []struct{ w, h int }{
		{140, 45}, {100, 30}, {80, 24}, {60, 20}, {40, 16},
	} {
		m := newTestModel(t)
		m.width, m.height = size.w, size.h
		m.resize()
		m.ensureValidFocus()
		m = selectStructures(t, m)
		lines := strings.Split(m.View(), "\n")
		if len(lines) != size.h {
			t.Errorf("%dx%d: rendered %d lines, want %d", size.w, size.h, len(lines), size.h)
		}
		for i, line := range lines {
			if got := lipgloss.Width(line); got > size.w {
				t.Errorf("%dx%d: line %d is %d columns wide: %q", size.w, size.h, i, got, line)
			}
		}
	}
}

func TestNarrowTerminalsDoNotPanic(t *testing.T) {
	for _, size := range []struct{ w, h int }{
		{120, 40}, {99, 30}, {80, 24}, {60, 20}, {40, 16}, {30, 12}, {24, 8}, {20, 6},
	} {
		m := newTestModel(t)
		m.width, m.height = size.w, size.h
		m.resize()
		view := m.View()
		if view == "" {
			t.Fatalf("empty view at %dx%d", size.w, size.h)
		}
	}
}

func TestCopyDetailSection(t *testing.T) {
	m := newTestModel(t)
	m = searchFor(t, m, "NtClose")
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.hasCurrent || m.current.Name != "NtClose" {
		t.Fatalf("wrong API opened: %+v", m.current.Name)
	}

	var copied string
	m.copyWriter = func(text string) error {
		copied = text
		return nil
	}
	m = send(m, runes("y"))
	if m.overlay != overlayCopy {
		t.Fatalf("expected copy overlay, got %v", m.overlay)
	}
	if len(m.copyTargets) < 2 || m.copyTargets[1].Label != "Description" {
		t.Fatalf("description target missing: %+v", m.copyTargets)
	}
	m = send(m, tea.KeyMsg{Type: tea.KeyDown})
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.overlay != overlayNone {
		t.Fatalf("copy did not close the overlay: %v", m.overlay)
	}
	if !strings.HasPrefix(copied, "DESCRIPTION\n\n") {
		t.Fatalf("copied the wrong section:\n%s", copied)
	}
	if !strings.Contains(copied, "NtClose closes a handle") {
		t.Fatalf("description text missing:\n%s", copied)
	}
	for _, forbidden := range []string{"CATEGORIES", "API FUNCTIONS", "API DETAILS", "│", "─"} {
		if strings.Contains(copied, forbidden) {
			t.Fatalf("copied text contains another panel or a border (%q):\n%s", forbidden, copied)
		}
	}
	if !strings.Contains(m.status, "Copied Description") {
		t.Fatalf("unexpected copy status: %q", m.status)
	}
}

func TestCopyListAndSidebarSelection(t *testing.T) {
	m := newTestModel(t)
	var copied string
	m.copyWriter = func(text string) error {
		copied = text
		return nil
	}

	// The categories panel is focused initially.
	want := m.sidebarItem().Label
	m = send(m, runes("y"))
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	if copied != want {
		t.Fatalf("sidebar copy = %q, want %q", copied, want)
	}

	m.focus = panelList
	want = m.apiList[m.listIndex].Name
	m = send(m, runes("y"))
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	if copied != want {
		t.Fatalf("list copy = %q, want %q", copied, want)
	}
}

func TestCopyStructureAndConstantSections(t *testing.T) {
	m := newTestModel(t)
	m = selectStructures(t, m)
	m.focus = panelDetail
	m = send(m, runes("y"))
	labels := make(map[string]bool)
	for _, target := range m.copyTargets {
		labels[target.Label] = true
	}
	for _, want := range []string{"Entire entry (Markdown)", "Definition", "Members", "References"} {
		if !labels[want] {
			t.Errorf("structure copy target %q missing: %+v", want, labels)
		}
	}
	m = send(m, tea.KeyMsg{Type: tea.KeyEsc})

	m = selectConstants(t, m)
	m.focus = panelDetail
	m = send(m, runes("y"))
	labels = make(map[string]bool)
	for _, target := range m.copyTargets {
		labels[target.Label] = true
	}
	for _, want := range []string{"Entire entry (Markdown)", "Value", "Description", "References"} {
		if !labels[want] {
			t.Errorf("constant copy target %q missing: %+v", want, labels)
		}
	}
}

func TestCopyFailureIsReported(t *testing.T) {
	m := newTestModel(t)
	m.copyWriter = func(string) error { return errors.New("no terminal") }
	m = send(m, runes("y"))
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.overlay != overlayNone {
		t.Fatalf("copy failure should close the overlay: %v", m.overlay)
	}
	if !m.statusErr || !strings.Contains(m.status, "Copy failed: no terminal") {
		t.Fatalf("unexpected copy failure status: %q", m.status)
	}
}

func TestCopyMenuEscapeDoesNotWrite(t *testing.T) {
	m := newTestModel(t)
	called := false
	m.copyWriter = func(string) error {
		called = true
		return nil
	}
	m = send(m, runes("y"))
	m = send(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.overlay != overlayNone {
		t.Fatalf("Esc should close the copy menu: %v", m.overlay)
	}
	if called {
		t.Fatal("Esc should not write to the clipboard")
	}
}

func TestExportWritesFile(t *testing.T) {
	t.Chdir(t.TempDir())
	m := newTestModel(t)
	m = searchFor(t, m, "VirtualAlloc")
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})

	m = send(m, runes("e"))
	if m.overlay != overlayExport {
		t.Fatalf("expected export overlay, got %v", m.overlay)
	}
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.overlay != overlayNone {
		t.Fatalf("export did not complete: %v (%s)", m.overlay, m.status)
	}
	path := filepath.Join("exports", "VirtualAlloc.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("export file missing: %v", err)
	}
}

func TestStructureExportWritesFile(t *testing.T) {
	t.Chdir(t.TempDir())
	m := newTestModel(t)
	m = gotoStructures(t, m)
	m = send(m, runes("/"))
	m = send(m, runes("STARTUPINFOW"))
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})

	m = send(m, runes("e"))
	if m.overlay != overlayExport {
		t.Fatalf("expected export overlay, got %v", m.overlay)
	}
	if !strings.Contains(m.View(), "Structure record as JSON") {
		t.Fatalf("structure export menu missing:\n%s", m.View())
	}
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.overlay != overlayNone {
		t.Fatalf("structure export did not complete: %v (%s)", m.overlay, m.status)
	}
	path := filepath.Join("exports", "STARTUPINFOW.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("structure export file missing: %v", err)
	}
	if !strings.Contains(string(data), `"structure"`) || !strings.Contains(string(data), `"id": "win32.startupinfow"`) {
		t.Fatalf("unexpected structure export:\n%s", data)
	}
}

func TestDetailRendersExampleAndHidesIt(t *testing.T) {
	m := newTestModel(t)
	m = searchFor(t, m, "CreateProcessW")
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	shown := m.renderDetail(m.current)
	if !strings.Contains(shown, "EXAMPLE") || !strings.Contains(shown, "CreateProcessW") {
		t.Fatalf("example missing:\n%s", shown)
	}
	m = send(m, runes("x"))
	hidden := m.renderDetail(m.current)
	if strings.Contains(hidden, "notepad.exe") {
		t.Fatalf("example still visible after toggle:\n%s", hidden)
	}
}

func selectConstants(t *testing.T, m Model) Model {
	t.Helper()
	for i, item := range m.sidebar {
		if item.Kind == sideConstants {
			m.sidebarIndex = i
			m.loadSidebarContent()
			return m
		}
	}
	t.Fatal("constants sidebar entry not found")
	return m
}

func gotoConstants(t *testing.T, m Model) Model {
	t.Helper()
	for m.sidebarItem().Kind != sideConstants {
		before := m.sidebarIndex
		m = send(m, runes("j"))
		if m.sidebarIndex == before {
			t.Fatal("constants sidebar entry not found")
		}
	}
	if m.mode != listConstants {
		t.Fatalf("expected constants list, got %s", m.ModeName())
	}
	return m
}

func TestConstantsViewAndSearch(t *testing.T) {
	m := newTestModel(t)
	m = gotoConstants(t, m)
	if len(m.constants) < 50 {
		t.Fatalf("expected the bundled constants, got %d", len(m.constants))
	}
	if len(m.currentConstant.Name) == 0 {
		t.Fatal("constant list should preview the first constant")
	}
	view := m.View()
	for _, want := range []string{"CONSTANTS", "MEM_COMMIT", "CONSTANT"} {
		if !strings.Contains(view, want) {
			t.Errorf("constants view missing %q:\n%s", want, view)
		}
	}

	m = send(m, runes("/"))
	m = send(m, runes("MEM_COMMIT"))
	if m.mode != listConstantSearch {
		t.Fatalf("expected constant search mode, got %s", m.ModeName())
	}
	if len(m.constantResults) == 0 || m.constantResults[0].Constant.Name != "MEM_COMMIT" {
		t.Fatalf("unexpected constant results: %+v", m.constantResults)
	}
	if !strings.Contains(m.View(), "CONSTANT SEARCH") {
		t.Fatalf("constant search heading missing:\n%s", m.View())
	}

	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.hasConstant || m.currentConstant.Name != "MEM_COMMIT" {
		t.Fatalf("Enter did not open the constant: %+v", m.currentConstant)
	}
	if m.hasCurrent || m.hasStructure {
		t.Fatal("API, structure, and constant detail state must be mutually exclusive")
	}
	detail := m.renderConstantDetail(m.currentConstant)
	for _, want := range []string{"0x00001000", "DESCRIPTION", "RELATED APIS", "VirtualAlloc", "REFERENCES"} {
		if !strings.Contains(detail, want) {
			t.Errorf("constant detail missing %q:\n%s", want, detail)
		}
	}
}

func TestConstantValueSearch(t *testing.T) {
	m := newTestModel(t)
	m = gotoConstants(t, m)
	m = send(m, runes("/"))
	m = send(m, runes("0x1000"))
	if len(m.constantResults) == 0 || m.constantResults[0].Constant.Name != "MEM_COMMIT" {
		t.Fatalf("value search did not match MEM_COMMIT: %+v", m.constantResults)
	}
}

func TestConstantRelatedOverlayAndReverseLinks(t *testing.T) {
	m := newTestModel(t)
	m = gotoConstants(t, m)
	m = send(m, runes("/"))
	m = send(m, runes("MEM_COMMIT"))
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.currentConstant.Name != "MEM_COMMIT" {
		t.Fatalf("wrong constant opened: %s", m.currentConstant.Name)
	}

	m = send(m, runes("g"))
	if m.overlay != overlayRelated {
		t.Fatalf("expected related overlay, got %v", m.overlay)
	}
	if !strings.Contains(m.View(), "VirtualAlloc") {
		t.Fatalf("reverse API links missing:\n%s", m.View())
	}
}

func TestConstantKeyFromAPIDetail(t *testing.T) {
	m := newTestModel(t)
	m = searchFor(t, m, "VirtualAlloc")
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = send(m, runes("c"))
	if m.mode != listConstants {
		t.Fatalf("c should open constants, got %s", m.ModeName())
	}
	if !m.hasConstant || m.currentConstant.Name != "MEM_COMMIT" {
		t.Fatalf("first related constant not opened: %+v", m.currentConstant)
	}
	if m.hasCurrent {
		t.Fatal("API detail state should be cleared when opening a constant")
	}
}

func TestConstantFavoriteIsDisabled(t *testing.T) {
	m := newTestModel(t)
	m = gotoConstants(t, m)
	before := len(m.app.State.Favorites)
	m = send(m, runes("f"))
	if got := len(m.app.State.Favorites); got != before {
		t.Fatalf("favorite count changed from %d to %d", before, got)
	}
	if !strings.Contains(m.status, "API entries only") {
		t.Fatalf("unexpected status: %q", m.status)
	}
}

func TestConstantSearchNoResults(t *testing.T) {
	m := newTestModel(t)
	m = gotoConstants(t, m)
	m = send(m, runes("/"))
	m = send(m, runes("zzzznotaconstant"))
	if len(m.constantResults) != 0 {
		t.Fatalf("unexpected constant results: %d", len(m.constantResults))
	}
	if !strings.Contains(m.View(), "No matches") {
		t.Fatalf("empty constant state not rendered:\n%s", m.View())
	}
}

func TestConstantExportWritesFile(t *testing.T) {
	t.Chdir(t.TempDir())
	m := newTestModel(t)
	m = gotoConstants(t, m)
	m = send(m, runes("/"))
	m = send(m, runes("MEM_COMMIT"))
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})

	m = send(m, runes("e"))
	if m.overlay != overlayExport {
		t.Fatalf("expected export overlay, got %v", m.overlay)
	}
	if !strings.Contains(m.View(), "Constant record as JSON") {
		t.Fatalf("constant export menu missing:\n%s", m.View())
	}
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.overlay != overlayNone {
		t.Fatalf("constant export did not complete: %v (%s)", m.overlay, m.status)
	}
	path := filepath.Join("exports", "MEM_COMMIT.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("constant export file missing: %v", err)
	}
	if !strings.Contains(string(data), `"constant"`) || !strings.Contains(string(data), `"id": "win32.mem_commit"`) {
		t.Fatalf("unexpected constant export:\n%s", data)
	}
}

func TestConstantLayoutFillsTerminal(t *testing.T) {
	for _, size := range []struct{ w, h int }{
		{140, 45}, {100, 30}, {80, 24}, {60, 20}, {40, 16},
	} {
		m := newTestModel(t)
		m.width, m.height = size.w, size.h
		m.resize()
		m.ensureValidFocus()
		m = selectConstants(t, m)
		lines := strings.Split(m.View(), "\n")
		if len(lines) != size.h {
			t.Errorf("%dx%d: rendered %d lines, want %d", size.w, size.h, len(lines), size.h)
		}
		for i, line := range lines {
			if got := lipgloss.Width(line); got > size.w {
				t.Errorf("%dx%d: line %d is %d columns wide: %q", size.w, size.h, i, got, line)
			}
		}
	}
}
