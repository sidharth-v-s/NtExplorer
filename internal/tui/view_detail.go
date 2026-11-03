package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"ntexplorer/internal/catalog"
)

// renderDetail builds the scrollable detail document for one API.
func (m Model) renderDetail(api catalog.API) string {
	width := m.detail.Width
	if width < 12 {
		width = 12
	}

	var lines []string
	addBlank := func() { lines = append(lines, "") }
	heading := func(title string) {
		addBlank()
		lines = append(lines, m.styles.Section.Render(title))
	}
	addWrapped := func(text string) {
		lines = append(lines, wrapText(text, width)...)
	}
	addStyled := func(text string, style lipgloss.Style) {
		for _, line := range wrapText(text, width) {
			lines = append(lines, style.Render(line))
		}
	}
	addField := func(label, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		labelText := label + ": "
		available := width - runewidth.StringWidth(labelText)
		if available < 8 {
			lines = append(lines, m.styles.Label.Render(labelText))
			addWrapped(value)
			return
		}
		wrapped := wrapText(value, available)
		if len(wrapped) == 0 {
			return
		}
		lines = append(lines, m.styles.Label.Render(labelText)+m.styles.Text.Render(wrapped[0]))
		padding := strings.Repeat(" ", runewidth.StringWidth(labelText))
		for _, extra := range wrapped[1:] {
			lines = append(lines, padding+m.styles.Text.Render(extra))
		}
	}

	lines = append(lines, m.styles.Brand.Render(truncatePlain(api.Name, width)))
	if api.Summary != "" {
		addStyled(api.Summary, m.styles.Text)
	}
	lines = append(lines, m.styles.RowDim.Render(truncatePlain(api.Category, width)))

	addBlank()
	addField("DLL", api.DLL)
	addField("Library", api.Library)
	addField("Header", api.Header)
	addField("Minimum Windows", api.MinimumWindows)
	addField("Aliases", strings.Join(api.Aliases, ", "))
	addField("Tags", strings.Join(api.Tags, ", "))

	heading("DESCRIPTION")
	addWrapped(api.Description)

	if api.Prototype != "" {
		heading("PROTOTYPE")
		for _, line := range wrapCode(api.Prototype, width-2) {
			if line == "" {
				lines = append(lines, "")
			} else {
				lines = append(lines, "  "+m.styles.Code.Render(line))
			}
		}
	}

	if len(api.Parameters) > 0 {
		heading("PARAMETERS")
		for i, p := range api.Parameters {
			if i > 0 {
				addBlank()
			}
			signature := p.Name
			if p.Type != "" {
				signature += "  " + p.Type
			}
			lines = append(lines, m.styles.Accent.Render(truncatePlain(signature, width)))
			if p.Description != "" {
				lines = append(lines, indentLines(wrapText(p.Description, width-2), "  ")...)
			}
			if len(p.Constants) > 0 {
				values := "  Values: " + strings.Join(p.Constants, ", ")
				lines = append(lines, m.styles.Tag.Render(truncatePlain(values, width)))
			}
		}
	}

	heading("RETURN VALUE")
	if api.Return.Type != "" {
		lines = append(lines, m.styles.Label.Render("Type: ")+m.styles.Text.Render(api.Return.Type))
	}
	if api.Return.Success != "" {
		addField("Success", api.Return.Success)
	}
	if api.Return.Failure != "" {
		addField("Failure", api.Return.Failure)
	}

	if api.UnicodeNotes != "" {
		heading("UNICODE NOTES")
		addWrapped(api.UnicodeNotes)
	}
	if api.Remarks != "" {
		heading("REMARKS")
		addWrapped(api.Remarks)
	}
	if api.MalDevNotes != "" {
		heading("MALDEV CONTEXT")
		addWrapped(api.MalDevNotes)
	}
	if api.DefensiveNotes != "" {
		heading("DEFENSIVE CONTEXT")
		addWrapped(api.DefensiveNotes)
	}

	if api.Example != nil {
		heading("EXAMPLE")
		if m.showExamples {
			if api.Example.Title != "" {
				lines = append(lines, m.styles.Accent.Render(truncatePlain(api.Example.Title, width)))
			}
			lines = append(lines, m.styles.PanelSub.Render(api.Example.Language))
			for _, line := range wrapCode(api.Example.Code, width-2) {
				if line == "" {
					lines = append(lines, "")
				} else {
					lines = append(lines, "  "+m.styles.Code.Render(line))
				}
			}
			if api.Example.Notes != "" {
				addBlank()
				addStyled(api.Example.Notes, m.styles.RowDim)
			}
			lines = append(lines, m.styles.RowDim.Render("Illustrative only; NtExplorer never executes examples."))
		} else {
			lines = append(lines, m.styles.RowDim.Render("Hidden. Press x to show examples."))
		}
	}

	if len(api.RelatedAPIs) > 0 || len(api.RelatedStructures) > 0 || len(api.RelatedConstants) > 0 {
		heading("RELATED")
		if len(api.RelatedAPIs) > 0 {
			lines = append(lines, m.styles.Label.Render("APIs"))
			for _, id := range api.RelatedAPIs {
				name := m.apiName(id)
				line := "  " + name
				if name != id {
					line += "  " + id
				}
				lines = append(lines, m.styles.Row.Render(truncatePlain(line, width)))
			}
			lines = append(lines, m.styles.RowDim.Render("Press g for related navigation."))
		}
		if len(api.RelatedStructures) > 0 {
			lines = append(lines, m.styles.Label.Render("Structures"))
			for _, id := range api.RelatedStructures {
				name := m.structureName(id)
				line := "  " + name
				if name != id {
					line += "  " + id
				}
				lines = append(lines, m.styles.Row.Render(truncatePlain(line, width)))
			}
			lines = append(lines, m.styles.RowDim.Render("Press s to open the first related structure."))
		}
		if len(api.RelatedConstants) > 0 {
			lines = append(lines, m.styles.Label.Render("Constants"))
			for _, id := range api.RelatedConstants {
				name := m.constantName(id)
				line := "  " + name
				if name != id {
					line += "  " + id
				}
				lines = append(lines, m.styles.Row.Render(truncatePlain(line, width)))
			}
			lines = append(lines, m.styles.RowDim.Render("Press c to open the first related constant."))
		}
	}

	if api.NativeAPI != nil {
		heading("NATIVE API")
		line := api.NativeAPI.Name
		if api.NativeAPI.Module != "" {
			line += "  " + api.NativeAPI.Module
		}
		lines = append(lines, m.styles.Accent.Render(truncatePlain(line, width)))
		if api.NativeAPI.Confidence != "" {
			lines = append(lines, m.styles.RowDim.Render("Confidence: "+api.NativeAPI.Confidence))
		}
		if api.NativeAPI.Notes != "" {
			addWrapped(api.NativeAPI.Notes)
		}
	}

	if len(api.References) > 0 {
		heading("REFERENCES")
		lines = append(lines, m.styles.Text.Render(fmt.Sprintf("%d source(s) · press r to inspect", len(api.References))))
	}

	if api.SourceNotes != "" {
		heading("SOURCE NOTES")
		addStyled(api.SourceNotes, m.styles.RowDim)
	}

	lines = append(lines, "")
	lines = append(lines, m.styles.RowDim.Render("f favorite · g related · r references · e export · y copy · x examples"))
	return strings.Join(lines, "\n")
}

// renderStatusDetail builds the scrollable detail document for an NTSTATUS
// value.
func (m Model) renderStatusDetail(status catalog.NTStatus) string {
	width := m.detail.Width
	if width < 12 {
		width = 12
	}

	var lines []string
	addBlank := func() { lines = append(lines, "") }
	heading := func(title string) {
		addBlank()
		lines = append(lines, m.styles.Section.Render(title))
	}
	addField := func(label, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		labelText := label + ": "
		available := width - runewidth.StringWidth(labelText)
		if available < 8 {
			lines = append(lines, m.styles.Label.Render(labelText))
			lines = append(lines, wrapText(value, width)...)
			return
		}
		wrapped := wrapText(value, available)
		if len(wrapped) == 0 {
			return
		}
		lines = append(lines, m.styles.Label.Render(labelText)+m.styles.Text.Render(wrapped[0]))
		padding := strings.Repeat(" ", runewidth.StringWidth(labelText))
		for _, extra := range wrapped[1:] {
			lines = append(lines, padding+m.styles.Text.Render(extra))
		}
	}

	lines = append(lines, m.styles.Brand.Render(truncatePlain(status.Name, width)))
	if status.Value != "" {
		lines = append(lines, "  "+m.styles.Code.Render(truncatePlain(status.Value, width-2)))
	}
	if status.Severity != "" {
		lines = append(lines, m.styles.RowDim.Render(truncatePlain(strings.ToUpper(status.Severity), width)))
	}

	if status.Description != "" {
		heading("DESCRIPTION")
		lines = append(lines, wrapText(status.Description, width)...)
	}

	if len(status.References) > 0 {
		heading("REFERENCES")
		lines = append(lines, m.styles.Text.Render(fmt.Sprintf("%d Microsoft source(s) · press r to inspect", len(status.References))))
	}

	addBlank()
	addField("ID", status.ID)
	lines = append(lines, "")
	lines = append(lines, m.styles.RowDim.Render("r references · e export · y copy"))
	return strings.Join(lines, "\n")
}

// structureName resolves a structure ID for display, falling back to the ID.
func (m Model) structureName(id string) string {
	if m.app != nil {
		if structure, ok := m.app.Structure(id); ok {
			return structure.Name
		}
	}
	return id
}

// constantName resolves a constant ID for display, falling back to the ID.
func (m Model) constantName(id string) string {
	if m.app != nil {
		if constant, ok := m.app.Constant(id); ok {
			return constant.Name
		}
	}
	return id
}

// renderStructureDetail builds the scrollable detail document for a structure.
func (m Model) renderStructureDetail(structure catalog.Structure) string {
	width := m.detail.Width
	if width < 12 {
		width = 12
	}

	var lines []string
	addBlank := func() { lines = append(lines, "") }
	heading := func(title string) {
		addBlank()
		lines = append(lines, m.styles.Section.Render(title))
	}
	addWrapped := func(text string) {
		lines = append(lines, wrapText(text, width)...)
	}
	addStyled := func(text string, style lipgloss.Style) {
		for _, line := range wrapText(text, width) {
			lines = append(lines, style.Render(line))
		}
	}

	lines = append(lines, m.styles.Brand.Render(truncatePlain(structure.Name, width)))
	if structure.Summary != "" {
		addStyled(structure.Summary, m.styles.Text)
	}
	if len(structure.Tags) > 0 {
		lines = append(lines, m.styles.Tag.Render(truncatePlain("tags: "+strings.Join(structure.Tags, ", "), width)))
	}

	if structure.Definition != "" {
		heading("DEFINITION")
		for _, line := range wrapCode(structure.Definition, width-2) {
			if line == "" {
				lines = append(lines, "")
			} else {
				lines = append(lines, "  "+m.styles.Code.Render(line))
			}
		}
	}

	if len(structure.Members) > 0 {
		heading("MEMBERS")
		for i, member := range structure.Members {
			if i > 0 {
				addBlank()
			}
			signature := member.Name
			if member.Type != "" {
				signature += "  " + member.Type
			}
			lines = append(lines, m.styles.Accent.Render(truncatePlain(signature, width)))
			if member.Description != "" {
				lines = append(lines, indentLines(wrapText(member.Description, width-2), "  ")...)
			}
		}
	}

	if structure.Notes != "" {
		heading("NOTES")
		addWrapped(structure.Notes)
	}

	if m.app != nil {
		related := m.app.StructureRelatedAPIs(structure.ID)
		if len(related) > 0 {
			heading("RELATED APIS")
			for _, api := range related {
				line := "  " + api.Name + "  " + api.ID
				lines = append(lines, m.styles.Row.Render(truncatePlain(line, width)))
			}
			lines = append(lines, m.styles.RowDim.Render("Press g for related API navigation."))
		}
	}

	if len(structure.References) > 0 {
		heading("REFERENCES")
		lines = append(lines, m.styles.Text.Render(fmt.Sprintf("%d source(s) · press r to inspect", len(structure.References))))
	}

	lines = append(lines, "")
	lines = append(lines, m.styles.RowDim.Render("g related APIs · r references · e export · y copy · s structures"))
	return strings.Join(lines, "\n")
}

// renderConstantDetail builds the scrollable detail document for a constant.
func (m Model) renderConstantDetail(constant catalog.Constant) string {
	width := m.detail.Width
	if width < 12 {
		width = 12
	}

	var lines []string
	addBlank := func() { lines = append(lines, "") }
	heading := func(title string) {
		addBlank()
		lines = append(lines, m.styles.Section.Render(title))
	}
	addWrapped := func(text string) {
		lines = append(lines, wrapText(text, width)...)
	}
	addField := func(label, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		labelText := label + ": "
		available := width - runewidth.StringWidth(labelText)
		if available < 8 {
			lines = append(lines, m.styles.Label.Render(labelText))
			addWrapped(value)
			return
		}
		wrapped := wrapText(value, available)
		if len(wrapped) == 0 {
			return
		}
		lines = append(lines, m.styles.Label.Render(labelText)+m.styles.Text.Render(wrapped[0]))
		padding := strings.Repeat(" ", runewidth.StringWidth(labelText))
		for _, extra := range wrapped[1:] {
			lines = append(lines, padding+m.styles.Text.Render(extra))
		}
	}

	lines = append(lines, m.styles.Brand.Render(truncatePlain(constant.Name, width)))
	if constant.Value != "" {
		lines = append(lines, "  "+m.styles.Code.Render(truncatePlain(constant.Value, width-2)))
	}
	if constant.Category != "" {
		lines = append(lines, m.styles.RowDim.Render(truncatePlain(constant.Category, width)))
	}
	if len(constant.Tags) > 0 {
		lines = append(lines, m.styles.Tag.Render(truncatePlain("tags: "+strings.Join(constant.Tags, ", "), width)))
	}

	if constant.Description != "" {
		heading("DESCRIPTION")
		addWrapped(constant.Description)
	}

	if m.app != nil {
		related := m.app.ConstantRelatedAPIs(constant.ID)
		if len(related) > 0 {
			heading("RELATED APIS")
			for _, api := range related {
				line := "  " + api.Name + "  " + api.ID
				lines = append(lines, m.styles.Row.Render(truncatePlain(line, width)))
			}
			lines = append(lines, m.styles.RowDim.Render("Press g for related API navigation."))
		}
	}

	if len(constant.References) > 0 {
		heading("REFERENCES")
		lines = append(lines, m.styles.Text.Render(fmt.Sprintf("%d source(s) · press r to inspect", len(constant.References))))
	}

	addBlank()
	addField("ID", constant.ID)
	lines = append(lines, "")
	lines = append(lines, m.styles.RowDim.Render("g related APIs · r references · e export · y copy · c constants"))
	return strings.Join(lines, "\n")
}
