package tui

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// View renders the whole screen.
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return "Loading NtExplorer…"
	}
	if m.width < 24 || m.height < 8 {
		return m.styles.Brand.Render("NtExplorer") + "\n" +
			m.styles.RowDim.Render("Terminal too small. Resize to at least 24 × 8.")
	}

	header := m.renderHeader()
	footer := m.renderFooter()
	bodyH := m.height - 2

	var body string
	if m.overlay != overlayNone {
		body = m.renderOverlay(bodyH)
	} else {
		body = m.renderBody(bodyH)
	}
	return header + "\n" + body + "\n" + footer
}

func (m Model) renderHeader() string {
	brand := m.styles.Brand.Render("NtExplorer")
	meta := m.styles.HeaderMeta.Render("Windows API Reference · v" + Version)
	gap := m.width - lipgloss.Width(brand) - lipgloss.Width(meta)
	if gap >= 2 {
		return brand + strings.Repeat(" ", gap) + meta
	}
	return brand
}

func (m Model) renderFooter() string {
	var left string
	switch {
	case m.status != "":
		if m.statusErr {
			left = m.styles.Error.Render(truncatePlain(m.status, m.width))
		} else {
			left = m.styles.Success.Render(truncatePlain(m.status, m.width))
		}
	case m.searching:
		left = m.styles.FooterHi.Render("SEARCH") + m.styles.Footer.Render("  type to filter · ↑/↓ results · Enter open · Esc cancel")
	default:
		hints := strings.Join(m.keys.Hints(), "  ")
		left = m.styles.Footer.Render(truncatePlain(hints, m.width))
	}

	right := ""
	if m.app != nil {
		right = m.app.StatusLine() + " · " + m.categoryLabel()
	}
	right = fmt.Sprintf("%s · %s", runtime.GOOS, right)
	styledRight := m.styles.Footer.Render(right)
	rightWidth := lipgloss.Width(styledRight)
	leftWidth := lipgloss.Width(left)
	if leftWidth+3+rightWidth > m.width {
		return padRight(left, m.width)
	}
	gap := m.width - leftWidth - rightWidth
	return left + strings.Repeat(" ", gap) + styledRight
}

func (m Model) renderBody(bodyH int) string {
	l := m.computeLayout()
	if l.stacked {
		top := m.renderListPanel(l.listW, l.listH)
		bottom := m.renderDetailPanel(l.detailW, l.detailH)
		return lipgloss.JoinVertical(lipgloss.Left, top, bottom)
	}
	panels := make([]string, 0, 3)
	if l.sidebarVisible {
		panels = append(panels, m.renderSidebarPanel(l.sidebarW, l.listH))
	}
	panels = append(panels,
		m.renderListPanel(l.listW, l.listH),
		m.renderDetailPanel(l.detailW, l.detailH),
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, panels...)
}

func (m Model) panel(title, subtitle, content string, focused bool, width, height int) string {
	// Positioning lipgloss style dimensions excludes borders but includes
	// padding: the border adds 2 columns and the panel has 1 column of
	// horizontal padding on each side.
	innerW := maxInt(1, width-4)
	innerH := maxInt(1, height-2)
	heading := m.panelHeading(title, subtitle, innerW)
	body := heading
	if content != "" && innerH > 1 {
		lines := strings.Split(content, "\n")
		available := innerH - 1
		if len(lines) > available {
			lines = lines[:available]
		}
		body += "\n" + strings.Join(lines, "\n")
	}
	style := m.styles.Panel
	if focused {
		style = m.styles.PanelFocused
	}
	return style.Width(maxInt(1, width-2)).Height(innerH).Render(body)
}

func (m Model) panelHeading(title, subtitle string, width int) string {
	title = truncatePlain(title, width)
	if subtitle == "" {
		return m.styles.PanelTitle.Render(title)
	}
	full := title + "  " + subtitle
	if lipgloss.Width(full) <= width {
		return m.styles.PanelTitle.Render(title) + "  " + m.styles.PanelSub.Render(subtitle)
	}
	return m.styles.PanelTitle.Render(truncatePlain(full, width))
}

// renderSidebarPanel draws the category / library navigation column.
func (m Model) renderSidebarPanel(width, height int) string {
	innerW := maxInt(1, width-4)
	innerH := maxInt(1, height-2)

	lines := make([]string, 0, len(m.sidebar))
	selectedLine := 0
	lineIndex := 0
	for i, item := range m.sidebar {
		if item.Header {
			lines = append(lines, m.styles.Group.Render(truncatePlain(item.Label, innerW)))
			lineIndex++
			continue
		}
		selected := i == m.sidebarIndex
		if selected {
			selectedLine = lineIndex
		}
		label := item.Label
		count := ""
		if item.Count > 0 {
			count = fmt.Sprintf("%d", item.Count)
		} else if item.Kind == sideSoon {
			count = "soon"
		}
		text := "  " + label
		if count != "" {
			gap := innerW - runewidth.StringWidth(text) - runewidth.StringWidth(count)
			if gap < 1 {
				text = truncatePlain(text, maxInt(1, innerW-runewidth.StringWidth(count)-1))
				gap = innerW - runewidth.StringWidth(text) - runewidth.StringWidth(count)
			}
			if gap < 1 {
				gap = 1
			}
			text += strings.Repeat(" ", gap) + count
		}
		text = truncatePlain(text, innerW)
		if selected {
			text = "▸" + strings.TrimPrefix(text, " ")
			if runewidth.StringWidth(text) > innerW {
				text = truncatePlain(text, innerW)
			}
			lines = append(lines, m.styles.Selected.Width(innerW).Render(text))
		} else {
			lines = append(lines, m.styles.Row.Render(text))
		}
		lineIndex++
	}

	visible := windowRows(lines, selectedLine, innerH-1)
	return m.panel("CATEGORIES", "", strings.Join(visible, "\n"), m.focus == panelCategories, width, height)
}

// renderListPanel draws API results, search results or the tag list.
func (m Model) renderListPanel(width, height int) string {
	innerW := maxInt(1, width-4)
	innerH := maxInt(1, height-2)

	title := "API FUNCTIONS"
	subtitle := ""
	var rows []string
	selected := m.listIndex

	switch m.mode {
	case listSearch:
		title = "SEARCH RESULTS"
		subtitle = fmt.Sprintf("%d for %q", len(m.results), m.query)
		for i, result := range m.results {
			line := result.API.Name + " · " + strings.Join(result.Reasons, ", ")
			rows = append(rows, m.listRow(line, i == selected, innerW))
		}
		if len(rows) == 0 {
			rows = m.emptySearchRows(innerW)
		}
	case listStructureSearch:
		title = "STRUCTURE SEARCH"
		subtitle = fmt.Sprintf("%d for %q", len(m.structureResults), m.query)
		for i, result := range m.structureResults {
			line := result.Structure.Name + " · " + strings.Join(result.Reasons, ", ")
			rows = append(rows, m.listRow(line, i == selected, innerW))
		}
		if len(rows) == 0 {
			rows = m.emptyStructureSearchRows(innerW)
		}
	case listStructures:
		title = "STRUCTURES"
		subtitle = fmt.Sprintf("%d structures", len(m.structures))
		for i, structure := range m.structures {
			line := structure.Name
			if len(structure.Members) > 0 {
				line += fmt.Sprintf(" · %d members", len(structure.Members))
			}
			rows = append(rows, m.listRow(line, i == selected, innerW))
		}
		if len(rows) == 0 {
			rows = wrapText("No structures are bundled in this catalog.", innerW)
		}
	case listConstantSearch:
		title = "CONSTANT SEARCH"
		subtitle = fmt.Sprintf("%d for %q", len(m.constantResults), m.query)
		for i, result := range m.constantResults {
			line := result.Constant.Name + "  " + result.Constant.Value + " · " + strings.Join(result.Reasons, ", ")
			rows = append(rows, m.listRow(line, i == selected, innerW))
		}
		if len(rows) == 0 {
			rows = m.emptyConstantSearchRows(innerW)
		}
	case listConstants:
		title = "CONSTANTS"
		subtitle = fmt.Sprintf("%d constants", len(m.constants))
		for i, constant := range m.constants {
			rows = append(rows, m.listRow(constant.Name+"  "+constant.Value, i == selected, innerW))
		}
		if len(rows) == 0 {
			rows = wrapText("No constants are bundled in this catalog.", innerW)
		}
	case listTags:
		title = "MALDEV TAGS"
		subtitle = fmt.Sprintf("%d tags", len(m.tags))
		for i, tag := range m.tags {
			line := fmt.Sprintf("%s (%d)", tag.Tag, tag.Count)
			rows = append(rows, m.listRow(line, i == m.tagIndex, innerW))
		}
		if len(rows) == 0 {
			rows = wrapText("No tags are declared in this catalog.", innerW)
		}
	case listSoon:
		title = strings.ToUpper(m.soonLabel)
		subtitle = "planned"
		rows = wrapText(m.soonMessage(), innerW)
	default:
		switch m.sidebarItem().Kind {
		case sideFavorites:
			title = "FAVORITES"
		case sideRecent:
			title = "RECENTLY VIEWED"
		}
		subtitle = fmt.Sprintf("%d · %s", len(m.apiList), m.categoryLabel())
		for i, api := range m.apiList {
			rows = append(rows, m.listRow(api.Name, i == selected, innerW))
		}
		if len(rows) == 0 {
			rows = m.emptyBrowseRows(innerW)
		}
	}

	if m.searching {
		// Keep the result heading once there is a query; the input row is
		// rendered above the results.
		if strings.TrimSpace(m.query) == "" {
			if m.searchStructures {
				title = "STRUCTURE SEARCH"
			} else if m.searchConstants {
				title = "CONSTANT SEARCH"
			} else {
				title = "SEARCH"
			}
			subtitle = "type to filter"
		}
		input := m.searchInput.View()
		rows = append([]string{input}, rows...)
		selected = m.listIndex + 1
	}

	contentHeight := innerH - 1
	if m.searching {
		contentHeight--
	}
	visible := windowRows(rows, selected, contentHeight)
	return m.panel(title, subtitle, strings.Join(visible, "\n"), m.focus == panelList, width, height)
}

func (m Model) listRow(line string, selected bool, width int) string {
	prefix := "  "
	if selected {
		prefix = "▸ "
	}
	text := truncatePlain(prefix+line, width)
	if selected {
		return m.styles.Selected.Width(width).Render(text)
	}
	return m.styles.Row.Render(text)
}

func (m Model) emptySearchRows(width int) []string {
	rows := []string{
		m.styles.Warning.Render(truncatePlain("No matches.", width)),
		"",
	}
	rows = append(rows, wrapText("Try a shorter query, an exact API name such as VirtualAllocEx, a tag such as remote-memory, or a category such as Memory Management.", width)...)
	return rows
}

func (m Model) emptyStructureSearchRows(width int) []string {
	rows := []string{
		m.styles.Warning.Render(truncatePlain("No matches.", width)),
		"",
	}
	rows = append(rows, wrapText("Try a structure name such as STARTUPINFOW, a member name such as hProcess, or a tag such as memory-management.", width)...)
	return rows
}

func (m Model) emptyConstantSearchRows(width int) []string {
	rows := []string{
		m.styles.Warning.Render(truncatePlain("No matches.", width)),
		"",
	}
	rows = append(rows, wrapText("Try a constant name such as MEM_COMMIT, a value such as 0x1000, or a tag such as memory-protection.", width)...)
	return rows
}

func (m Model) emptyBrowseRows(width int) []string {
	var text string
	switch m.sidebarItem().Kind {
	case sideFavorites:
		text = "No favorites yet. Open an API and press f to add it."
	case sideRecent:
		text = "No recently viewed APIs yet. Open an entry to populate this list."
	default:
		text = "No APIs in this section. Press / to search the whole catalog."
	}
	return wrapText(text, width)
}

// renderDetailPanel draws the scrollable API detail column.
func (m Model) renderDetailPanel(width, height int) string {
	title := "API DETAILS"
	subtitle := ""
	content := ""
	switch {
	case m.hasConstant:
		title = "CONSTANT"
		subtitle = m.currentConstant.Name
		content = m.detail.View()
	case m.hasStructure:
		title = "STRUCTURE"
		subtitle = m.currentStructure.Name
		content = m.detail.View()
	case m.hasCurrent:
		subtitle = m.current.Name
		if m.showExamples && m.current.Example != nil {
			subtitle += " · example"
		}
		content = m.detail.View()
	default:
		content = strings.Join(m.welcomeDetail(maxInt(1, width-4)), "\n")
	}
	return m.panel(title, subtitle, content, m.focus == panelDetail, width, height)
}

func (m Model) welcomeDetail(width int) []string {
	var lines []string
	add := func(text string) { lines = append(lines, wrapText(text, width)...) }
	lines = append(lines, m.styles.Brand.Render("NtExplorer"))
	add("")
	add("An offline Windows API reference for learning Windows internals, malware analysis and defensive investigation.")
	add("")
	add("Search the catalog with /, browse a category on the left, and press Enter to read an entry. Tab moves between panels; ? lists every key.")
	add("To copy one panel without the other columns, press y and choose an entry or detail section. Terminal-native selection spans the whole row.")
	add("")
	add("NtExplorer only documents APIs. It never calls them, executes examples, or touches other processes.")
	if m.app != nil && len(m.app.Warnings) > 0 {
		add("")
		lines = append(lines, m.styles.Warning.Render("Startup notes"))
		for _, warning := range m.app.Warnings {
			add("• " + warning)
		}
	}
	return lines
}

// windowRows returns at most height rows centered on the selected row.
func windowRows(rows []string, selected, height int) []string {
	if height <= 0 {
		return nil
	}
	if len(rows) <= height {
		return rows
	}
	offset := selected - height/2
	if offset < 0 {
		offset = 0
	}
	if offset > len(rows)-height {
		offset = len(rows) - height
	}
	return rows[offset : offset+height]
}

// truncatePlain shortens a single plain-text line for display.
func truncatePlain(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if runewidth.StringWidth(text) <= width {
		return text
	}
	if width == 1 {
		return "…"
	}
	return runewidth.Truncate(text, width, "…")
}

func padRight(text string, width int) string {
	gap := width - lipgloss.Width(text)
	if gap <= 0 {
		return text
	}
	return text + strings.Repeat(" ", gap)
}

// wrapText word-wraps prose to width, preserving explicit line breaks.
func wrapText(text string, width int) []string {
	if width <= 0 {
		return nil
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	var out []string
	for _, paragraph := range strings.Split(text, "\n") {
		paragraph = strings.TrimRight(paragraph, " \t")
		if paragraph == "" {
			out = append(out, "")
			continue
		}
		words := strings.Fields(paragraph)
		line := ""
		for _, word := range words {
			for runewidth.StringWidth(word) > width {
				if line != "" {
					out = append(out, line)
					line = ""
				}
				head, tail := splitAtWidth(word, width)
				out = append(out, head)
				word = tail
			}
			switch {
			case line == "":
				line = word
			case runewidth.StringWidth(line)+1+runewidth.StringWidth(word) <= width:
				line += " " + word
			default:
				out = append(out, line)
				line = word
			}
		}
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func splitAtWidth(text string, width int) (string, string) {
	if width <= 0 {
		return "", text
	}
	widthSoFar := 0
	for i, r := range text {
		w := runewidth.RuneWidth(r)
		if widthSoFar+w > width {
			return text[:i], text[i:]
		}
		widthSoFar += w
	}
	return text, ""
}

// wrapCode hard-wraps code lines to width without reflowing tokens.
func wrapCode(code string, width int) []string {
	if width <= 0 {
		return nil
	}
	code = strings.TrimRight(strings.ReplaceAll(code, "\r\n", "\n"), "\n")
	var out []string
	for _, line := range strings.Split(code, "\n") {
		line = strings.ReplaceAll(line, "\t", "    ")
		if line == "" {
			out = append(out, "")
			continue
		}
		for runewidth.StringWidth(line) > width {
			head, tail := splitAtWidth(line, width)
			out = append(out, head)
			line = tail
		}
		out = append(out, line)
	}
	return out
}

func indentLines(lines []string, prefix string) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		if line == "" {
			out[i] = ""
		} else {
			out[i] = prefix + line
		}
	}
	return out
}
