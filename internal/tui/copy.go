package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"ntexplorer/internal/catalog"
	"ntexplorer/internal/export"
)

// copyTarget is one selectable clipboard payload. Text is plain text with no
// panel borders or ANSI styling, ready to paste elsewhere.
type copyTarget struct {
	Label string
	Text  string
}

// openCopy builds a context-aware copy menu for the focused panel.
func (m *Model) openCopy() {
	targets := m.copyTargetsForFocus()
	if len(targets) == 0 {
		m.status = "Nothing to copy in this panel."
		m.statusErr = false
		return
	}
	m.copyTargets = targets
	m.copyIndex = 0
	m.overlay = overlayCopy
	m.status = ""
}

func (m Model) copyTargetsForFocus() []copyTarget {
	switch m.focus {
	case panelCategories:
		return m.sidebarCopyTargets()
	case panelList:
		return m.listCopyTargets()
	case panelDetail:
		return m.detailCopyTargets()
	}
	return nil
}

func (m Model) sidebarCopyTargets() []copyTarget {
	item := m.sidebarItem()
	if item.Header || strings.TrimSpace(item.Label) == "" {
		return nil
	}
	return []copyTarget{{Label: "Selected sidebar item", Text: item.Label}}
}

func (m Model) listCopyTargets() []copyTarget {
	switch m.mode {
	case listTags:
		if tag, ok := m.selectedTag(); ok {
			return []copyTarget{{Label: "Selected tag", Text: tag}}
		}
	case listSoon:
		return nil
	}
	if structure, ok := m.selectedStructure(); ok {
		return []copyTarget{{Label: "Selected structure", Text: structure.Name}}
	}
	if constant, ok := m.selectedConstant(); ok {
		text := constant.Name
		if constant.Value != "" {
			text += "  " + constant.Value
		}
		return []copyTarget{{Label: "Selected constant", Text: text}}
	}
	if api, ok := m.selectedAPI(); ok {
		return []copyTarget{{Label: "Selected API", Text: api.Name}}
	}
	return nil
}

func (m Model) detailCopyTargets() []copyTarget {
	switch {
	case m.constantActive():
		return constantCopyTargets(m.currentConstant)
	case m.structureActive():
		return structureCopyTargets(m.currentStructure)
	case m.hasCurrent:
		return m.apiCopyTargets(m.current)
	}
	return nil
}

func (m Model) apiCopyTargets(api catalog.API) []copyTarget {
	targets := []copyTarget{{
		Label: "Entire entry (Markdown)",
		Text:  export.Markdown(api),
	}}
	add := func(label, text string) {
		if strings.TrimSpace(text) != "" {
			targets = append(targets, copyTarget{Label: label, Text: text})
		}
	}

	add("Description", copySection("DESCRIPTION", api.Description))
	add("Prototype", copySection("PROTOTYPE", api.Prototype))
	if len(api.Parameters) > 0 {
		var body strings.Builder
		for i, parameter := range api.Parameters {
			if i > 0 {
				body.WriteString("\n\n")
			}
			body.WriteString(parameter.Name)
			if parameter.Type != "" {
				body.WriteString("  " + parameter.Type)
			}
			if parameter.Description != "" {
				body.WriteString("\n  " + strings.ReplaceAll(parameter.Description, "\n", "\n  "))
			}
			if len(parameter.Constants) > 0 {
				body.WriteString("\n  Values: " + strings.Join(parameter.Constants, ", "))
			}
		}
		add("Parameters", copySection("PARAMETERS", body.String()))
	}
	if api.Return.Type != "" || api.Return.Success != "" || api.Return.Failure != "" {
		var body strings.Builder
		if api.Return.Type != "" {
			body.WriteString("Type: " + api.Return.Type + "\n")
		}
		if api.Return.Success != "" {
			body.WriteString("Success: " + api.Return.Success + "\n")
		}
		if api.Return.Failure != "" {
			body.WriteString("Failure: " + api.Return.Failure + "\n")
		}
		add("Return value", copySection("RETURN VALUE", body.String()))
	}
	add("Unicode notes", copySection("UNICODE NOTES", api.UnicodeNotes))
	add("Remarks", copySection("REMARKS", api.Remarks))
	add("MalDev context", copySection("MALDEV CONTEXT", api.MalDevNotes))
	add("Defensive context", copySection("DEFENSIVE CONTEXT", api.DefensiveNotes))

	if api.Example != nil {
		var body strings.Builder
		if api.Example.Title != "" {
			body.WriteString(api.Example.Title + "\n\n")
		}
		if api.Example.Language != "" {
			body.WriteString(api.Example.Language + "\n\n")
		}
		body.WriteString(api.Example.Code)
		if api.Example.Notes != "" {
			body.WriteString("\n\n" + api.Example.Notes)
		}
		add("Example", copySection("EXAMPLE", body.String()))
	}

	if related := m.relatedCopyText(api); related != "" {
		add("Related entries", related)
	}
	if api.NativeAPI != nil {
		var body strings.Builder
		body.WriteString(api.NativeAPI.Name)
		if api.NativeAPI.Module != "" {
			body.WriteString("  " + api.NativeAPI.Module)
		}
		if api.NativeAPI.Confidence != "" {
			body.WriteString("\nConfidence: " + api.NativeAPI.Confidence)
		}
		if api.NativeAPI.Notes != "" {
			body.WriteString("\n" + api.NativeAPI.Notes)
		}
		add("Native API", copySection("NATIVE API", body.String()))
	}
	add("References", referencesCopyText(api.References))
	add("Source notes", copySection("SOURCE NOTES", api.SourceNotes))
	return targets
}

func (m Model) relatedCopyText(api catalog.API) string {
	var body strings.Builder
	if len(api.RelatedAPIs) > 0 {
		body.WriteString("APIs\n")
		for _, id := range api.RelatedAPIs {
			body.WriteString("  " + m.apiName(id) + "  " + id + "\n")
		}
	}
	if len(api.RelatedStructures) > 0 {
		if body.Len() > 0 {
			body.WriteString("\n")
		}
		body.WriteString("Structures\n")
		for _, id := range api.RelatedStructures {
			body.WriteString("  " + m.structureName(id) + "  " + id + "\n")
		}
	}
	if len(api.RelatedConstants) > 0 {
		if body.Len() > 0 {
			body.WriteString("\n")
		}
		body.WriteString("Constants\n")
		for _, id := range api.RelatedConstants {
			body.WriteString("  " + m.constantName(id) + "  " + id + "\n")
		}
	}
	return copySection("RELATED", body.String())
}

func structureCopyTargets(structure catalog.Structure) []copyTarget {
	targets := []copyTarget{{
		Label: "Entire entry (Markdown)",
		Text:  export.StructureMarkdown(structure),
	}}
	add := func(label, text string) {
		if strings.TrimSpace(text) != "" {
			targets = append(targets, copyTarget{Label: label, Text: text})
		}
	}
	add("Definition", copySection("DEFINITION", structure.Definition))
	if len(structure.Members) > 0 {
		var body strings.Builder
		for i, member := range structure.Members {
			if i > 0 {
				body.WriteString("\n\n")
			}
			body.WriteString(member.Name)
			if member.Type != "" {
				body.WriteString("  " + member.Type)
			}
			if member.Description != "" {
				body.WriteString("\n  " + strings.ReplaceAll(member.Description, "\n", "\n  "))
			}
		}
		add("Members", copySection("MEMBERS", body.String()))
	}
	add("Notes", copySection("NOTES", structure.Notes))
	add("References", referencesCopyText(structure.References))
	return targets
}

func constantCopyTargets(constant catalog.Constant) []copyTarget {
	targets := []copyTarget{{
		Label: "Entire entry (Markdown)",
		Text:  export.ConstantMarkdown(constant),
	}}
	add := func(label, text string) {
		if strings.TrimSpace(text) != "" {
			targets = append(targets, copyTarget{Label: label, Text: text})
		}
	}
	add("Value", copySection("VALUE", constant.Value))
	add("Description", copySection("DESCRIPTION", constant.Description))
	add("References", referencesCopyText(constant.References))
	return targets
}

func referencesCopyText(references []catalog.Reference) string {
	var body strings.Builder
	for i, reference := range references {
		if i > 0 {
			body.WriteString("\n")
		}
		if reference.Title != "" {
			fmt.Fprintf(&body, "%d. %s\n   %s\n", i+1, reference.Title, reference.URL)
		} else {
			fmt.Fprintf(&body, "%d. %s\n", i+1, reference.URL)
		}
		if reference.SourceType != "" {
			body.WriteString("   " + reference.SourceType + "\n")
		}
	}
	return copySection("REFERENCES", body.String())
}

// copySection keeps a heading with its content and trims trailing blank lines.
func copySection(title, body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	return title + "\n\n" + body
}

func (m Model) copyContextName() string {
	switch m.focus {
	case panelCategories:
		return m.sidebarItem().Label
	case panelList:
		if targets := m.listCopyTargets(); len(targets) > 0 {
			return targets[0].Text
		}
	case panelDetail:
		if name, ok := m.currentEntryName(); ok {
			return name
		}
	}
	return ""
}

func (m Model) copyContent(width, maxRows int) (string, []string) {
	var lines []string
	for i, target := range m.copyTargets {
		lines = append(lines, m.listRow(target.Label, i == m.copyIndex, width))
	}
	if len(m.copyTargets) == 0 {
		lines = append(lines, m.styles.RowDim.Render("Nothing to copy."))
	}
	if m.copyIndex >= 0 && m.copyIndex < len(m.copyTargets) {
		target := m.copyTargets[m.copyIndex]
		preview := firstLine(target.Text)
		if preview == "" {
			preview = "(empty)"
		}
		lines = append(lines,
			"",
			m.styles.Label.Render("Preview"),
			m.styles.Text.Render(truncatePlain(preview, width)),
			m.styles.RowDim.Render(fmt.Sprintf("%d characters", utf8.RuneCountInString(strings.TrimRight(target.Text, "\r\n")))),
		)
	}
	lines = append(lines, "", m.styles.RowDim.Render("Enter copy · ↑/↓ choose · Esc close"))

	offset := clamp(m.copyIndex-(maxRows-1)/2, 0, maxInt(0, len(lines)-maxRows))
	lines = lines[offset:]
	lines = lines[:minInt(len(lines), maxRows)]

	title := "COPY"
	if name := m.copyContextName(); name != "" {
		title += " · " + name
	}
	return title, lines
}

func firstLine(text string) string {
	text = strings.TrimLeft(text, "\r\n")
	if index := strings.IndexAny(text, "\r\n"); index >= 0 {
		return strings.TrimSpace(text[:index])
	}
	return strings.TrimSpace(text)
}

func (m Model) updateCopyOverlay(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	keys := m.keys
	switch {
	case key.Matches(msg, keys.Escape, keys.Copy, keys.Quit):
		m.overlay = overlayNone
		m.status = ""
	case key.Matches(msg, keys.Up):
		if m.copyIndex > 0 {
			m.copyIndex--
		}
	case key.Matches(msg, keys.Down):
		if m.copyIndex < len(m.copyTargets)-1 {
			m.copyIndex++
		}
	case key.Matches(msg, keys.Enter):
		m.copyCurrent()
	}
	return m, nil
}

func (m *Model) copyCurrent() {
	if m.copyIndex < 0 || m.copyIndex >= len(m.copyTargets) {
		m.overlay = overlayNone
		return
	}
	target := m.copyTargets[m.copyIndex]
	if m.copyWriter == nil {
		m.overlay = overlayNone
		m.status = "Copy failed: terminal clipboard is unavailable."
		m.statusErr = true
		return
	}
	if err := m.copyWriter(target.Text); err != nil {
		m.overlay = overlayNone
		m.status = "Copy failed: " + err.Error()
		m.statusErr = true
		return
	}
	m.overlay = overlayNone
	m.status = fmt.Sprintf("Copied %s (%d characters).", target.Label, utf8.RuneCountInString(strings.TrimRight(target.Text, "\r\n")))
	m.statusErr = false
}
