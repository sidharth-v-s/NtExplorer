package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// renderOverlay draws a centered modal over the body area.
func (m Model) renderOverlay(bodyH int) string {
	boxW := minInt(86, m.width-2)
	if boxW < 24 {
		boxW = maxInt(10, m.width-2)
	}
	innerW := maxInt(8, boxW-6)
	maxRows := maxInt(3, bodyH-6)

	title := ""
	var lines []string
	switch m.overlay {
	case overlayHelp:
		title, lines = m.helpContent(innerW, maxRows)
	case overlayRelated:
		title, lines = m.relatedContent(innerW, maxRows)
	case overlayReferences:
		title, lines = m.referencesContent(innerW, maxRows)
	case overlayExport:
		title, lines = m.exportContent(innerW, maxRows)
	case overlayCopy:
		title, lines = m.copyContent(innerW, maxRows)
	}

	body := m.styles.OverlayTitle.Render(truncatePlain(title, innerW))
	if len(lines) > 0 {
		body += "\n\n" + strings.Join(lines, "\n")
	}
	// Overlay style has 2 columns of padding on each side plus a border, so
	// Width covers the content and padding while the border adds 2 columns.
	box := m.styles.Overlay.Width(maxInt(1, boxW-2)).Render(body)
	return lipgloss.Place(m.width, bodyH, lipgloss.Center, lipgloss.Center, box)
}

func (m Model) helpContent(width, maxRows int) (string, []string) {
	var lines []string
	add := func(text string) { lines = append(lines, wrapText(text, width)...) }
	heading := func(text string) {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, m.styles.Section.Render(text))
	}
	keyLine := func(keys, description string) {
		line := "  " + keys
		pad := 22 - runewidth.StringWidth(line)
		if pad < 1 {
			pad = 1
		}
		lines = append(lines, m.styles.Accent.Render(line)+strings.Repeat(" ", pad)+m.styles.Text.Render(truncatePlain(description, maxInt(1, width-runewidth.StringWidth(line)-pad))))
	}

	heading("NAVIGATION")
	keyLine("↑/↓ or j/k", "move selection / scroll detail")
	keyLine("PgUp / PgDn", "page through a panel")
	keyLine("Home / End", "first / last item")
	keyLine("Ctrl+U / Ctrl+D", "half-page detail scroll")
	keyLine("Tab / Shift+Tab", "move between panels")
	keyLine("Enter", "open API or apply tag")
	keyLine("Esc", "clear search or go back")

	heading("SEARCH")
	keyLine("/", "search APIs, structures, or constants in the active view")
	keyLine("Enter", "open the highlighted result")
	add("Search is case-insensitive. Results show why each entry matched. In the Structures view, / searches names, members, tags, and notes. In the Constants view, / searches names, numeric values, tags, and descriptions.")

	heading("ENTRY ACTIONS")
	keyLine("g", "related APIs, or reverse API links")
	keyLine("r", "references")
	keyLine("f", "toggle favorite (API entries)")
	keyLine("e", "export as JSON or Markdown")
	keyLine("y", "copy the focused panel or detail section")
	keyLine("x", "show or hide code examples")
	keyLine("s", "open structures; from an API, its first related structure")
	keyLine("c", "open constants; from an API, its first related constant")
	keyLine("q", "quit")
	add("Copy uses the terminal clipboard (OSC 52). Your terminal or multiplexer must allow clipboard writes; tmux users may need `set -g set-clipboard on`.")

	heading("DATA")
	if m.app != nil && m.app.Catalog != nil {
		add(fmt.Sprintf("Catalog %s · %d APIs · %d structures · %d constants · %d categories · %d tags",
			valueOr(m.app.Catalog.CatalogVersion, "unversioned"),
			len(m.app.Catalog.APIs),
			len(m.app.Catalog.Structures),
			len(m.app.Catalog.Constants),
			len(m.app.Catalog.Categories),
			len(m.app.Catalog.Tags)))
	}
	add("Bundled offline data. Sources include Microsoft Learn and Windows SDK headers; see each entry's references. Catalog text is summarized in original wording.")
	add("License: MIT. NtExplorer is not affiliated with or endorsed by Microsoft or System Informer.")
	add("NtExplorer documents APIs only. It never calls them, executes examples, or manipulates processes.")

	if m.app != nil && len(m.app.Warnings) > 0 {
		heading("STARTUP NOTES")
		for _, warning := range m.app.Warnings {
			add("• " + warning)
		}
	}

	lines = append(lines, "", m.styles.RowDim.Render("Esc / ? close · ↑/↓ scroll"))
	offset := clamp(m.helpOffset, 0, maxInt(0, len(lines)-maxRows))
	if offset > 0 {
		lines = lines[offset:]
	}
	lines = lines[:minInt(len(lines), maxRows)]
	return "HELP · NtExplorer v" + Version, lines
}

func (m Model) relatedContent(width, maxRows int) (string, []string) {
	name, ok := m.currentEntryName()
	if !ok {
		return "RELATED APIs", []string{"No entry selected."}
	}
	var lines []string
	lines = append(lines, m.styles.Text.Render(truncatePlain(name, width)), "")
	related := m.currentRelatedIDs()
	selectedLine := 0
	for i, id := range related {
		apiName := m.apiName(id)
		text := apiName + "  " + id
		if i == m.overlayIndex {
			selectedLine = len(lines)
		}
		lines = append(lines, m.listRow(text, i == m.overlayIndex, width))
	}
	if len(related) == 0 {
		lines = append(lines, m.styles.RowDim.Render("No related APIs are documented."))
	} else {
		lines = append(lines, "", m.styles.RowDim.Render("↑/↓ choose · Enter open · Esc close"))
	}
	offset := clamp(selectedLine-(maxRows-1)/2, 0, maxInt(0, len(lines)-maxRows))
	lines = lines[offset:]
	lines = lines[:minInt(len(lines), maxRows)]
	return "RELATED · " + name, lines
}

func (m Model) referencesContent(width, maxRows int) (string, []string) {
	name, references, ok := m.currentReferences()
	if !ok {
		return "REFERENCES", []string{"No entry selected."}
	}
	var lines []string
	for i, ref := range references {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, ref.Title))
		lines = append(lines, "   "+ref.URL)
		if ref.SourceType != "" {
			lines = append(lines, "   "+m.styles.RowDim.Render(ref.SourceType))
		}
		lines = append(lines, "")
	}
	lines = append(lines, m.styles.RowDim.Render("Esc / r close · URLs are shown for reference only"))
	offset := clamp(m.refOffset, 0, maxInt(0, len(lines)-maxRows))
	lines = lines[offset:]
	lines = lines[:minInt(len(lines), maxRows)]
	return "REFERENCES · " + name, lines
}

func (m Model) exportContent(width, maxRows int) (string, []string) {
	name, ok := m.currentEntryName()
	if !ok {
		return "EXPORT", []string{"No entry selected."}
	}
	var lines []string
	if m.exportConfirm {
		lines = append(lines,
			m.styles.Warning.Render(truncatePlain("File already exists:", width)),
			m.styles.Text.Render(truncatePlain(m.pendingPath, width)),
			"",
			m.styles.Text.Render("Overwrite it?"),
			"",
			m.styles.Accent.Render("y")+m.styles.Text.Render(" overwrite   ")+
				m.styles.Accent.Render("n")+m.styles.Text.Render(" cancel"),
		)
		return "EXPORT CONFIRMATION", lines
	}

	lines = append(lines, m.styles.Text.Render(truncatePlain(name, width)), "")
	for i, target := range m.exportTargets {
		lines = append(lines, m.listRow(m.exportName(target), i == m.exportIndex, width))
	}
	if m.exportIndex >= 0 && m.exportIndex < len(m.exportTargets) {
		lines = append(lines,
			"",
			m.styles.Label.Render("Destination"),
			m.styles.Text.Render(truncatePlain(m.exportDestination(m.exportTargets[m.exportIndex]), width)),
		)
	}
	lines = append(lines, "", m.styles.RowDim.Render("Enter export · ↑/↓ choose · Esc close"))
	lines = lines[:minInt(len(lines), maxRows)]
	return "EXPORT", lines
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
