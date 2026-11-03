// Package export renders catalog entries as JSON and Markdown. It only writes
// text files; it never executes or loads exported content.
package export

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ntexplorer/internal/catalog"
	"ntexplorer/internal/search"
)

// ErrExists is returned by WriteNew when the destination already exists.
var ErrExists = errors.New("destination file already exists")

// EntryDocument is the JSON shape of a single exported API entry.
type EntryDocument struct {
	SchemaVersion int         `json:"schema_version"`
	ExportedAt    string      `json:"exported_at"`
	Catalog       string      `json:"catalog_version,omitempty"`
	API           catalog.API `json:"api"`
}

// StructureDocument is the JSON shape of a single exported structure entry.
type StructureDocument struct {
	SchemaVersion int               `json:"schema_version"`
	ExportedAt    string            `json:"exported_at"`
	Catalog       string            `json:"catalog_version,omitempty"`
	Structure     catalog.Structure `json:"structure"`
}

// ConstantDocument is the JSON shape of a single exported constant entry.
type ConstantDocument struct {
	SchemaVersion int              `json:"schema_version"`
	ExportedAt    string           `json:"exported_at"`
	Catalog       string           `json:"catalog_version,omitempty"`
	Constant      catalog.Constant `json:"constant"`
}

// StatusDocument is the JSON shape of a single exported NTSTATUS entry.
type StatusDocument struct {
	SchemaVersion int              `json:"schema_version"`
	ExportedAt    string           `json:"exported_at"`
	Catalog       string           `json:"catalog_version,omitempty"`
	Status        catalog.NTStatus `json:"status"`
}

// ResultsDocument is the JSON shape of an exported search result list.
type ResultsDocument struct {
	SchemaVersion int           `json:"schema_version"`
	ExportedAt    string        `json:"exported_at"`
	Query         string        `json:"query"`
	Results       []ResultEntry `json:"results"`
}

// ResultEntry is one API identifier and summary in a results export.
type ResultEntry struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Summary string   `json:"summary"`
	Score   int      `json:"score,omitempty"`
	Reasons []string `json:"reasons,omitempty"`
}

// APIJSON renders one API entry using the catalog schema.
func APIJSON(api catalog.API, catalogVersion string) ([]byte, error) {
	doc := EntryDocument{
		SchemaVersion: catalog.SupportedSchemaVersion,
		ExportedAt:    time.Now().UTC().Format(time.RFC3339),
		Catalog:       catalogVersion,
		API:           api,
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// StructureJSON renders one structure entry using the catalog schema.
func StructureJSON(structure catalog.Structure, catalogVersion string) ([]byte, error) {
	doc := StructureDocument{
		SchemaVersion: catalog.SupportedSchemaVersion,
		ExportedAt:    time.Now().UTC().Format(time.RFC3339),
		Catalog:       catalogVersion,
		Structure:     structure,
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// ConstantJSON renders one constant entry using the catalog schema.
func ConstantJSON(constant catalog.Constant, catalogVersion string) ([]byte, error) {
	doc := ConstantDocument{
		SchemaVersion: catalog.SupportedSchemaVersion,
		ExportedAt:    time.Now().UTC().Format(time.RFC3339),
		Catalog:       catalogVersion,
		Constant:      constant,
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// StatusJSON renders one NTSTATUS entry using the catalog schema.
func StatusJSON(status catalog.NTStatus, catalogVersion string) ([]byte, error) {
	doc := StatusDocument{
		SchemaVersion: catalog.SupportedSchemaVersion,
		ExportedAt:    time.Now().UTC().Format(time.RFC3339),
		Catalog:       catalogVersion,
		Status:        status,
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// ResultsJSON renders a ranked result list.
func ResultsJSON(query string, results []search.Result) ([]byte, error) {
	doc := ResultsDocument{
		SchemaVersion: catalog.SupportedSchemaVersion,
		ExportedAt:    time.Now().UTC().Format(time.RFC3339),
		Query:         query,
		Results:       make([]ResultEntry, 0, len(results)),
	}
	for _, result := range results {
		doc.Results = append(doc.Results, ResultEntry{
			ID:      result.API.ID,
			Name:    result.API.Name,
			Summary: result.API.Summary,
			Score:   result.Score,
			Reasons: result.Reasons,
		})
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// Markdown renders a readable reference page for one API.
func Markdown(api catalog.API) string {
	var b strings.Builder
	write := func(format string, args ...any) {
		fmt.Fprintf(&b, format, args...)
	}

	write("# %s\n\n", api.Name)
	write("%s\n\n", api.Summary)

	write("| Field | Value |\n|---|---|\n")
	write("| Category | %s |\n", mdCell(api.Category))
	if api.DLL != "" {
		write("| DLL | %s |\n", mdCell(api.DLL))
	}
	if api.Library != "" {
		write("| Library | %s |\n", mdCell(api.Library))
	}
	if api.Header != "" {
		write("| Header | %s |\n", mdCell(api.Header))
	}
	if api.MinimumWindows != "" {
		write("| Minimum Windows | %s |\n", mdCell(api.MinimumWindows))
	}
	if len(api.Aliases) > 0 {
		write("| Aliases | %s |\n", mdCell(strings.Join(api.Aliases, ", ")))
	}
	if len(api.Tags) > 0 {
		write("| Tags | %s |\n", mdCell(strings.Join(api.Tags, ", ")))
	}
	write("\n")

	write("## Description\n\n%s\n\n", api.Description)

	if api.Prototype != "" {
		write("## Prototype\n\n")
		writeFenced(&b, "c", api.Prototype)
		write("\n")
	}

	if len(api.Parameters) > 0 {
		write("## Parameters\n\n")
		for _, p := range api.Parameters {
			write("### `%s`\n\n", p.Name)
			write("Type: `%s`\n\n", p.Type)
			write("%s\n\n", p.Description)
			if len(p.Constants) > 0 {
				write("Related constants: %s\n\n", strings.Join(p.Constants, ", "))
			}
		}
	}

	write("## Return value\n\n")
	write("Type: `%s`\n\n", api.Return.Type)
	if api.Return.Success != "" {
		write("Success: %s\n\n", api.Return.Success)
	}
	if api.Return.Failure != "" {
		write("Failure: %s\n\n", api.Return.Failure)
	}

	if api.UnicodeNotes != "" {
		write("## Unicode notes\n\n%s\n\n", api.UnicodeNotes)
	}
	if api.Remarks != "" {
		write("## Remarks\n\n%s\n\n", api.Remarks)
	}
	if api.MalDevNotes != "" {
		write("## MalDev context\n\n%s\n\n", api.MalDevNotes)
	}
	if api.DefensiveNotes != "" {
		write("## Defensive context\n\n%s\n\n", api.DefensiveNotes)
	}

	if api.Example != nil {
		write("## Example: %s\n\n", api.Example.Title)
		writeFenced(&b, api.Example.Language, api.Example.Code)
		write("\n")
		if api.Example.Notes != "" {
			write("%s\n\n", api.Example.Notes)
		}
	}

	if len(api.RelatedAPIs) > 0 {
		write("## Related APIs\n\n")
		for _, id := range api.RelatedAPIs {
			write("- `%s`\n", id)
		}
		write("\n")
	}

	if len(api.RelatedConstants) > 0 {
		write("## Related constants\n\n")
		for _, id := range api.RelatedConstants {
			write("- `%s`\n", id)
		}
		write("\n")
	}

	if len(api.References) > 0 {
		write("## References\n\n")
		for _, ref := range api.References {
			if ref.Title != "" {
				write("- [%s](%s)\n", ref.Title, ref.URL)
			} else {
				write("- <%s>\n", ref.URL)
			}
		}
		write("\n")
	}

	if api.SourceNotes != "" {
		write("## Source notes\n\n%s\n\n", api.SourceNotes)
	}

	write("---\n\nGenerated by NtExplorer from catalog record `%s`.\n", api.ID)
	return b.String()
}

// StructureMarkdown renders a readable reference page for one structure.
func StructureMarkdown(structure catalog.Structure) string {
	var b strings.Builder
	write := func(format string, args ...any) {
		fmt.Fprintf(&b, format, args...)
	}

	write("# %s\n\n", structure.Name)
	write("%s\n\n", structure.Summary)

	if len(structure.Tags) > 0 {
		write("Tags: %s\n\n", strings.Join(structure.Tags, ", "))
	}

	if structure.Definition != "" {
		write("## Definition\n\n")
		writeFenced(&b, "c", structure.Definition)
		write("\n")
	}

	if len(structure.Members) > 0 {
		write("## Members\n\n")
		write("| Member | Type | Description |\n|---|---|---|\n")
		for _, member := range structure.Members {
			write("| `%s` | `%s` | %s |\n", mdCell(member.Name), mdCell(member.Type), mdCell(member.Description))
		}
		write("\n")
	}

	if structure.Notes != "" {
		write("## Notes\n\n%s\n\n", structure.Notes)
	}

	if len(structure.References) > 0 {
		write("## References\n\n")
		for _, ref := range structure.References {
			if ref.Title != "" {
				write("- [%s](%s)\n", ref.Title, ref.URL)
			} else {
				write("- <%s>\n", ref.URL)
			}
		}
		write("\n")
	}

	write("---\n\nGenerated by NtExplorer from catalog record `%s`.\n", structure.ID)
	return b.String()
}

// ConstantMarkdown renders a readable reference page for one constant.
func ConstantMarkdown(constant catalog.Constant) string {
	var b strings.Builder
	write := func(format string, args ...any) {
		fmt.Fprintf(&b, format, args...)
	}

	write("# %s\n\n", constant.Name)
	write("`%s`\n\n", constant.Value)

	write("| Field | Value |\n|---|---|\n")
	if constant.Category != "" {
		write("| Category | %s |\n", mdCell(constant.Category))
	}
	if len(constant.Tags) > 0 {
		write("| Tags | %s |\n", mdCell(strings.Join(constant.Tags, ", ")))
	}
	write("\n")

	if constant.Description != "" {
		write("## Description\n\n%s\n\n", constant.Description)
	}

	if len(constant.References) > 0 {
		write("## References\n\n")
		for _, ref := range constant.References {
			if ref.Title != "" {
				write("- [%s](%s)\n", ref.Title, ref.URL)
			} else {
				write("- <%s>\n", ref.URL)
			}
		}
		write("\n")
	}

	write("---\n\nGenerated by NtExplorer from catalog record `%s`.\n", constant.ID)
	return b.String()
}

// StatusMarkdown renders a readable reference page for one NTSTATUS value.
func StatusMarkdown(status catalog.NTStatus) string {
	var b strings.Builder
	write := func(format string, args ...any) {
		fmt.Fprintf(&b, format, args...)
	}

	write("# %s\n\n", status.Name)
	write("`%s`\n\n", status.Value)

	write("| Field | Value |\n|---|---|\n")
	if status.Severity != "" {
		write("| Severity | %s |\n", mdCell(status.Severity))
	}
	write("\n")

	if status.Description != "" {
		write("## Description\n\n%s\n\n", status.Description)
	}

	if len(status.References) > 0 {
		write("## References\n\n")
		for _, ref := range status.References {
			if ref.Title != "" {
				write("- [%s](%s)\n", ref.Title, ref.URL)
			} else {
				write("- <%s>\n", ref.URL)
			}
		}
		write("\n")
	}

	write("---\n\nGenerated by NtExplorer from catalog record `%s`.\n", status.ID)
	return b.String()
}

// mdCell escapes characters that would break a Markdown table cell.
func mdCell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

// writeFenced writes a code block using a fence longer than any backtick run
// in the content, so embedded backticks cannot terminate the block early.
func writeFenced(b *strings.Builder, language, content string) {
	fence := "```"
	for strings.Contains(content, fence) {
		fence += "`"
	}
	fmt.Fprintf(b, "%s%s\n%s\n%s\n", fence, language, content, fence)
}

// DefaultFilename returns a safe file name for an API entry.
func DefaultFilename(api catalog.API, extension string) string {
	return entryFilename(api.Name, api.ID, extension)
}

// StructureFilename returns a safe file name for a structure entry.
func StructureFilename(structure catalog.Structure, extension string) string {
	return entryFilename(structure.Name, structure.ID, extension)
}

// ConstantFilename returns a safe file name for a constant entry.
func ConstantFilename(constant catalog.Constant, extension string) string {
	return entryFilename(constant.Name, constant.ID, extension)
}

// StatusFilename returns a safe file name for an NTSTATUS entry.
func StatusFilename(status catalog.NTStatus, extension string) string {
	return entryFilename(status.Name, status.ID, extension)
}

func entryFilename(name, id, extension string) string {
	base := sanitizeFilename(name)
	if base == "" {
		base = sanitizeFilename(id)
	}
	if base == "" {
		base = "entry"
	}
	extension = strings.TrimPrefix(extension, ".")
	return base + "." + extension
}

func sanitizeFilename(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ', r == '.':
			b.WriteRune('_')
		}
	}
	return strings.Trim(b.String(), "_")
}

// WriteNew writes data to path without overwriting an existing file. The
// caller is responsible for asking the user before replacing a file.
func WriteNew(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return ErrExists
		}
		return err
	}
	defer file.Close()
	_, err = file.Write(data)
	return err
}

// WriteReplace writes data to path, replacing an existing file.
func WriteReplace(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
