package catalog

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// Severity classifies a validation finding.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Issue is one validation finding tied to a source file and record.
type Issue struct {
	Severity Severity
	File     string
	RecordID string
	Field    string
	Message  string
}

func (i Issue) String() string {
	var b strings.Builder
	b.WriteString(string(i.Severity))
	b.WriteString(": ")
	if i.File != "" {
		b.WriteString(i.File)
		if i.RecordID != "" {
			b.WriteString("#")
			b.WriteString(i.RecordID)
		}
		b.WriteString(": ")
	} else if i.RecordID != "" {
		b.WriteString(i.RecordID)
		b.WriteString(": ")
	}
	if i.Field != "" {
		b.WriteString(i.Field)
		b.WriteString(": ")
	}
	b.WriteString(i.Message)
	return b.String()
}

// HasErrors reports whether any issue is an error.
func HasErrors(issues []Issue) bool {
	for _, issue := range issues {
		if issue.Severity == SeverityError {
			return true
		}
	}
	return false
}

// FormatIssues renders issues one per line, errors first.
func FormatIssues(issues []Issue) string {
	if len(issues) == 0 {
		return "catalog is valid"
	}
	sorted := append([]Issue(nil), issues...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Severity != sorted[j].Severity {
			return sorted[i].Severity == SeverityError
		}
		if sorted[i].File != sorted[j].File {
			return sorted[i].File < sorted[j].File
		}
		return sorted[i].RecordID < sorted[j].RecordID
	})
	var b strings.Builder
	for i, issue := range sorted {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(issue.String())
	}
	return b.String()
}

var (
	apiIDPattern = regexp.MustCompile(`^[a-z0-9]+(\.[a-z0-9_]+)+$`)
	tagPattern   = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	exampleLangs = map[string]bool{"c": true, "cpp": true}
	allowedURLs  = map[string]bool{"http": true, "https": true}
)

// Validate checks the merged catalog for the rules defined by the project
// specification. It never mutates the catalog.
func Validate(c *Catalog) []Issue {
	if c == nil {
		return []Issue{{Severity: SeverityError, Message: "catalog is nil"}}
	}
	var issues []Issue

	// Catalog metadata.
	if strings.TrimSpace(c.Name) == "" {
		issues = append(issues, Issue{Severity: SeverityWarning, File: "catalog.json", Field: "name", Message: "catalog name is empty"})
	}
	if strings.TrimSpace(c.CatalogVersion) == "" {
		issues = append(issues, Issue{Severity: SeverityWarning, File: "catalog.json", Field: "catalog_version", Message: "catalog version is empty"})
	}
	if len(c.Categories) == 0 {
		issues = append(issues, Issue{Severity: SeverityError, File: "catalog.json", Field: "categories", Message: "no categories declared"})
	}
	seenCategories := map[string]bool{}
	for _, category := range c.Categories {
		if strings.TrimSpace(category.Name) == "" {
			issues = append(issues, Issue{Severity: SeverityError, File: "catalog.json", Field: "categories", Message: "category with empty name"})
			continue
		}
		if seenCategories[category.Name] {
			issues = append(issues, Issue{Severity: SeverityError, File: "catalog.json", RecordID: category.Name, Field: "categories", Message: "duplicate category name"})
		}
		seenCategories[category.Name] = true
	}
	seenTags := map[string]bool{}
	for _, tag := range c.Tags {
		if !tagPattern.MatchString(tag) {
			issues = append(issues, Issue{Severity: SeverityError, File: "catalog.json", RecordID: tag, Field: "tags", Message: "tag must be lowercase kebab-case"})
		}
		if seenTags[tag] {
			issues = append(issues, Issue{Severity: SeverityWarning, File: "catalog.json", RecordID: tag, Field: "tags", Message: "duplicate tag in vocabulary"})
		}
		seenTags[tag] = true
	}

	// API records.
	seenIDs := map[string]bool{}
	seenNames := map[string]string{}
	aliasOwner := map[string]string{}
	for _, api := range c.APIs {
		issues = append(issues, validateAPI(c, api, seenIDs, seenNames, aliasOwner)...)
	}
	// Alias collisions are checked after all names are known.
	for _, api := range c.APIs {
		for _, alias := range api.Aliases {
			if owner, ok := seenNames[strings.ToLower(alias)]; ok && owner != api.ID {
				issues = append(issues, Issue{
					Severity: SeverityWarning,
					File:     api.Source,
					RecordID: api.ID,
					Field:    "aliases",
					Message:  fmt.Sprintf("alias %q is also the name of %s", alias, owner),
				})
			}
		}
	}

	seenStructureIDs := map[string]bool{}
	for _, s := range c.Structures {
		if strings.TrimSpace(s.ID) == "" {
			issues = append(issues, Issue{Severity: SeverityError, File: s.Source, Field: "id", Message: "structure has an empty id"})
		}
		if seenStructureIDs[s.ID] {
			issues = append(issues, Issue{Severity: SeverityError, File: s.Source, RecordID: s.ID, Field: "id", Message: "duplicate structure id"})
		}
		seenStructureIDs[s.ID] = true
		if strings.TrimSpace(s.Name) == "" {
			issues = append(issues, Issue{Severity: SeverityError, File: s.Source, RecordID: s.ID, Field: "name", Message: "structure has an empty name"})
		}
		if strings.TrimSpace(s.Summary) == "" {
			issues = append(issues, Issue{Severity: SeverityWarning, File: s.Source, RecordID: s.ID, Field: "summary", Message: "structure has no summary"})
		}
		seenMembers := map[string]bool{}
		for _, m := range s.Members {
			if strings.TrimSpace(m.Name) == "" || strings.TrimSpace(m.Type) == "" {
				issues = append(issues, Issue{Severity: SeverityWarning, File: s.Source, RecordID: s.ID, Field: "members", Message: "member with empty name or type"})
			}
			if seenMembers[m.Name] {
				issues = append(issues, Issue{Severity: SeverityWarning, File: s.Source, RecordID: s.ID, Field: "members", Message: fmt.Sprintf("duplicate member %q", m.Name)})
			}
			seenMembers[m.Name] = true
		}
		issues = append(issues, validateReferences(s.Source, s.ID, "structures", s.References, false)...)
	}

	seenConstantIDs := map[string]bool{}
	for _, k := range c.Constants {
		if strings.TrimSpace(k.ID) == "" {
			issues = append(issues, Issue{Severity: SeverityError, File: k.Source, Field: "id", Message: "constant has an empty id"})
		}
		if seenConstantIDs[k.ID] {
			issues = append(issues, Issue{Severity: SeverityError, File: k.Source, RecordID: k.ID, Field: "id", Message: "duplicate constant id"})
		}
		seenConstantIDs[k.ID] = true
		if strings.TrimSpace(k.Name) == "" {
			issues = append(issues, Issue{Severity: SeverityError, File: k.Source, RecordID: k.ID, Field: "name", Message: "constant has an empty name"})
		}
		if strings.TrimSpace(k.Value) == "" {
			issues = append(issues, Issue{Severity: SeverityWarning, File: k.Source, RecordID: k.ID, Field: "value", Message: "constant has no value"})
		}
		issues = append(issues, validateReferences(k.Source, k.ID, "constants", k.References, false)...)
	}

	seenStatusIDs := map[string]bool{}
	for _, s := range c.Statuses {
		if strings.TrimSpace(s.ID) == "" {
			issues = append(issues, Issue{Severity: SeverityError, File: s.Source, Field: "id", Message: "NTSTATUS record has an empty id"})
		}
		if seenStatusIDs[s.ID] {
			issues = append(issues, Issue{Severity: SeverityError, File: s.Source, RecordID: s.ID, Field: "id", Message: "duplicate NTSTATUS id"})
		}
		seenStatusIDs[s.ID] = true
		if strings.TrimSpace(s.Name) == "" || strings.TrimSpace(s.Value) == "" {
			issues = append(issues, Issue{Severity: SeverityError, File: s.Source, RecordID: s.ID, Field: "name", Message: "NTSTATUS record needs a name and value"})
		}
		issues = append(issues, validateReferences(s.Source, s.ID, "statuses", s.References, false)...)
	}

	return issues
}

func validateAPI(c *Catalog, api API, seenIDs map[string]bool, seenNames map[string]string, aliasOwner map[string]string) []Issue {
	var issues []Issue

	add := func(severity Severity, field, message string) {
		issues = append(issues, Issue{
			Severity: severity,
			File:     api.Source,
			RecordID: api.ID,
			Field:    field,
			Message:  message,
		})
	}

	if strings.TrimSpace(api.ID) == "" {
		add(SeverityError, "id", "API has an empty id")
	} else {
		if seenIDs[api.ID] {
			add(SeverityError, "id", "duplicate API id")
		}
		seenIDs[api.ID] = true
		if !apiIDPattern.MatchString(api.ID) {
			add(SeverityWarning, "id", "id does not follow the lowercase dotted convention, e.g. win32.virtualalloc")
		}
	}

	if strings.TrimSpace(api.Name) == "" {
		add(SeverityError, "name", "API has an empty name")
	} else {
		key := strings.ToLower(api.Name)
		if owner, ok := seenNames[key]; ok {
			add(SeverityError, "name", fmt.Sprintf("duplicate API name; already used by %s", owner))
		} else {
			seenNames[key] = api.ID
		}
		if strings.IndexFunc(api.Name, unicode.IsSpace) >= 0 {
			add(SeverityWarning, "name", "API name contains whitespace")
		}
	}

	if strings.TrimSpace(api.Category) == "" {
		add(SeverityError, "category", "API has no category")
	} else if !c.HasCategory(api.Category) {
		add(SeverityError, "category", fmt.Sprintf("unknown category %q", api.Category))
	}

	if strings.TrimSpace(api.DLL) == "" {
		add(SeverityWarning, "dll", "API has no DLL")
	}
	if strings.TrimSpace(api.Summary) == "" {
		add(SeverityError, "summary", "API has no summary")
	}
	if strings.TrimSpace(api.Description) == "" {
		add(SeverityError, "description", "API has no description")
	}
	if strings.TrimSpace(api.Prototype) == "" {
		add(SeverityError, "prototype", "API has no prototype")
	} else if strings.TrimSpace(api.Name) != "" && !strings.Contains(strings.ToLower(api.Prototype), strings.ToLower(api.Name)) {
		add(SeverityWarning, "prototype", "prototype does not contain the API name")
	}
	if strings.TrimSpace(api.Return.Type) == "" {
		add(SeverityError, "return.type", "API has no return type")
	}
	if api.Parameters == nil {
		add(SeverityWarning, "parameters", "parameters field is missing; use [] for documented zero-parameter functions")
	}
	seenParameters := map[string]bool{}
	for _, p := range api.Parameters {
		if strings.TrimSpace(p.Name) == "" {
			add(SeverityError, "parameters", "parameter with an empty name")
			continue
		}
		if seenParameters[p.Name] {
			add(SeverityError, "parameters", fmt.Sprintf("duplicate parameter name %q", p.Name))
		}
		seenParameters[p.Name] = true
		if strings.TrimSpace(p.Type) == "" {
			add(SeverityWarning, "parameters", fmt.Sprintf("parameter %q has no type", p.Name))
		}
		if strings.TrimSpace(p.Description) == "" {
			add(SeverityWarning, "parameters", fmt.Sprintf("parameter %q has no description", p.Name))
		}
	}

	seenAPITags := map[string]bool{}
	for _, tag := range api.Tags {
		if !tagPattern.MatchString(tag) {
			add(SeverityError, "tags", fmt.Sprintf("tag %q is not lowercase kebab-case", tag))
		}
		if !c.HasTag(tag) {
			add(SeverityWarning, "tags", fmt.Sprintf("tag %q is not in the declared tag vocabulary", tag))
		}
		if seenAPITags[tag] {
			add(SeverityWarning, "tags", fmt.Sprintf("duplicate tag %q", tag))
		}
		seenAPITags[tag] = true
	}

	for _, id := range api.RelatedAPIs {
		if id == api.ID {
			add(SeverityWarning, "related_apis", "API references itself")
			continue
		}
		if _, ok := c.API(id); !ok {
			add(SeverityError, "related_apis", fmt.Sprintf("related API %q does not resolve", id))
		}
	}
	for _, id := range api.RelatedStructures {
		if _, ok := c.Structure(id); !ok {
			severity := SeverityWarning
			if len(c.Structures) > 0 {
				severity = SeverityError
			}
			add(severity, "related_structures", fmt.Sprintf("related structure %q does not resolve", id))
		}
	}
	for _, id := range api.RelatedConstants {
		if _, ok := c.Constant(id); !ok {
			severity := SeverityWarning
			if len(c.Constants) > 0 {
				severity = SeverityError
			}
			add(severity, "related_constants", fmt.Sprintf("related constant %q does not resolve", id))
		}
	}

	// Duplicate aliases inside one record and against other records' names.
	seenAliases := map[string]bool{}
	for _, alias := range api.Aliases {
		if strings.TrimSpace(alias) == "" {
			add(SeverityWarning, "aliases", "empty alias")
			continue
		}
		key := strings.ToLower(alias)
		if seenAliases[key] {
			add(SeverityWarning, "aliases", fmt.Sprintf("duplicate alias %q", alias))
		}
		if prev, ok := aliasOwner[key]; ok && prev != api.ID {
			add(SeverityWarning, "aliases", fmt.Sprintf("alias %q is declared by multiple APIs", alias))
		}
		seenAliases[key] = true
		aliasOwner[key] = api.ID
	}

	issues = append(issues, validateReferences(api.Source, api.ID, "references", api.References, true)...)

	if api.Example != nil {
		if !exampleLangs[strings.ToLower(api.Example.Language)] {
			add(SeverityError, "example.language", "example language must be \"c\" or \"cpp\"")
		}
		if strings.TrimSpace(api.Example.Title) == "" {
			add(SeverityWarning, "example.title", "example has no title")
		}
		if strings.TrimSpace(api.Example.Code) == "" {
			add(SeverityError, "example.code", "example has no code")
		}
		if strings.TrimSpace(api.Example.Notes) == "" {
			add(SeverityWarning, "example.notes", "example has no notes")
		}
	}

	if api.NativeAPI != nil {
		if strings.TrimSpace(api.NativeAPI.Name) == "" {
			add(SeverityWarning, "native_api", "native_api has no name")
		}
		switch api.NativeAPI.Confidence {
		case "", "documented", "likely", "unverified":
		default:
			add(SeverityWarning, "native_api", "unexpected confidence value; use documented, likely, or unverified")
		}
	}

	return issues
}

func validateReferences(file, recordID, field string, refs []Reference, requireOne bool) []Issue {
	var issues []Issue
	add := func(sev Severity, msg string) {
		issues = append(issues, Issue{Severity: sev, File: file, RecordID: recordID, Field: field, Message: msg})
	}
	if len(refs) == 0 {
		if requireOne {
			add(SeverityError, "at least one reference is required")
		}
		return issues
	}
	for _, ref := range refs {
		if strings.TrimSpace(ref.Title) == "" {
			add(SeverityError, "reference has no title")
		}
		u, err := url.Parse(strings.TrimSpace(ref.URL))
		if err != nil || u.Scheme == "" || u.Host == "" || !allowedURLs[u.Scheme] {
			add(SeverityError, fmt.Sprintf("reference %q does not contain a valid http(s) URL", ref.Title))
		}
	}
	return issues
}
