// Package catalog defines the NtExplorer catalog data model, loader and
// validator. It has no dependency on the TUI.
package catalog

// SupportedSchemaVersion is the highest catalog schema version understood by
// this build. A catalog that declares a newer version is rejected rather than
// silently misread.
const SupportedSchemaVersion = 1

// Meta is the content of data/catalog.json. It describes the catalog itself
// and defines the vocabulary used by API records.
type Meta struct {
	SchemaVersion  int        `json:"schema_version"`
	Name           string     `json:"name"`
	CatalogVersion string     `json:"catalog_version"`
	Categories     []Category `json:"categories"`
	Tags           []string   `json:"tags"`
}

// Category is a primary API category such as "Memory Management".
type Category struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// APIFile is the on-disk shape of one data/apis/*.json file.
type APIFile struct {
	SchemaVersion int   `json:"schema_version"`
	APIs          []API `json:"apis"`
}

// StructureFile is the on-disk shape of data/structures/*.json.
type StructureFile struct {
	SchemaVersion int         `json:"schema_version"`
	Structures    []Structure `json:"structures"`
}

// ConstantFile is the on-disk shape of data/constants/*.json.
type ConstantFile struct {
	SchemaVersion int        `json:"schema_version"`
	Constants     []Constant `json:"constants"`
}

// StatusFile is the on-disk shape of data/ntstatus/*.json.
type StatusFile struct {
	SchemaVersion int        `json:"schema_version"`
	Statuses      []NTStatus `json:"statuses"`
}

// Catalog is the merged, in-memory view of every bundled data file.
type Catalog struct {
	Meta
	APIs       []API
	Structures []Structure
	Constants  []Constant
	Statuses   []NTStatus

	apiByID       map[string]*API
	structureByID map[string]*Structure
	constantByID  map[string]*Constant
	statusByID    map[string]*NTStatus
	categorySet   map[string]struct{}
	tagSet        map[string]struct{}
}

// API is one documented Windows API entry.
type API struct {
	ID                string      `json:"id"`
	Name              string      `json:"name"`
	Aliases           []string    `json:"aliases,omitempty"`
	Category          string      `json:"category"`
	DLL               string      `json:"dll,omitempty"`
	Header            string      `json:"header,omitempty"`
	Library           string      `json:"library,omitempty"`
	MinimumWindows    string      `json:"minimum_windows,omitempty"`
	Summary           string      `json:"summary"`
	Description       string      `json:"description"`
	Prototype         string      `json:"prototype"`
	Parameters        []Parameter `json:"parameters"`
	Return            ReturnInfo  `json:"return"`
	Tags              []string    `json:"tags,omitempty"`
	RelatedAPIs       []string    `json:"related_apis,omitempty"`
	RelatedStructures []string    `json:"related_structures,omitempty"`
	RelatedConstants  []string    `json:"related_constants,omitempty"`
	NativeAPI         *NativeLink `json:"native_api,omitempty"`
	UnicodeNotes      string      `json:"unicode_notes,omitempty"`
	Remarks           string      `json:"remarks,omitempty"`
	MalDevNotes       string      `json:"maldev_notes,omitempty"`
	DefensiveNotes    string      `json:"defensive_notes,omitempty"`
	Example           *Example    `json:"example,omitempty"`
	References        []Reference `json:"references"`
	SourceNotes       string      `json:"source_notes,omitempty"`

	// Source is filled by the loader with the data file the record came from.
	// It is never read from or written to JSON.
	Source string `json:"-"`
}

// Parameter is one ordered parameter of an API.
type Parameter struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Constants   []string `json:"constants,omitempty"`
}

// ReturnInfo documents the return value contract of an API.
type ReturnInfo struct {
	Type    string `json:"type"`
	Success string `json:"success,omitempty"`
	Failure string `json:"failure,omitempty"`
}

// NativeLink points at a related Native API when the relationship is known
// and sourced. It is intentionally conservative.
type NativeLink struct {
	Name       string `json:"name"`
	Module     string `json:"module,omitempty"`
	Confidence string `json:"confidence,omitempty"`
	Notes      string `json:"notes,omitempty"`
}

// Example is a short, benign C/C++ snippet shown as text only.
type Example struct {
	Language string `json:"language"`
	Title    string `json:"title"`
	Code     string `json:"code"`
	Notes    string `json:"notes,omitempty"`
}

// Reference is a provenance entry for a published record.
type Reference struct {
	Title      string `json:"title"`
	URL        string `json:"url"`
	SourceType string `json:"source_type,omitempty"`
}

// Structure is a documented Windows structure (Release 0.2 catalog module).
type Structure struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Summary    string      `json:"summary"`
	Definition string      `json:"definition,omitempty"`
	Members    []Member    `json:"members,omitempty"`
	Notes      string      `json:"notes,omitempty"`
	Tags       []string    `json:"tags,omitempty"`
	References []Reference `json:"references,omitempty"`
	Source     string      `json:"-"`
}

// Member is one field of a Structure.
type Member struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// Constant is a named Windows constant or flag value.
type Constant struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Value       string      `json:"value"`
	Category    string      `json:"category,omitempty"`
	Description string      `json:"description,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
	References  []Reference `json:"references,omitempty"`
	Source      string      `json:"-"`
}

// NTStatus is a documented NTSTATUS value.
type NTStatus struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Value       string      `json:"value"`
	Description string      `json:"description,omitempty"`
	Severity    string      `json:"severity,omitempty"`
	References  []Reference `json:"references,omitempty"`
	Source      string      `json:"-"`
}

// API returns the record with the given ID.
func (c *Catalog) API(id string) (*API, bool) {
	if c == nil {
		return nil, false
	}
	api, ok := c.apiByID[id]
	return api, ok
}

// Structure returns the structure with the given ID.
func (c *Catalog) Structure(id string) (*Structure, bool) {
	if c == nil {
		return nil, false
	}
	s, ok := c.structureByID[id]
	return s, ok
}

// Constant returns the constant with the given ID.
func (c *Catalog) Constant(id string) (*Constant, bool) {
	if c == nil {
		return nil, false
	}
	k, ok := c.constantByID[id]
	return k, ok
}

// Status returns the NTSTATUS record with the given ID.
func (c *Catalog) Status(id string) (*NTStatus, bool) {
	if c == nil {
		return nil, false
	}
	s, ok := c.statusByID[id]
	return s, ok
}

// HasCategory reports whether name is a declared primary category.
func (c *Catalog) HasCategory(name string) bool {
	_, ok := c.categorySet[name]
	return ok
}

// HasTag reports whether tag belongs to the declared tag vocabulary.
func (c *Catalog) HasTag(tag string) bool {
	_, ok := c.tagSet[tag]
	return ok
}

// TagCount is the number of APIs carrying a tag.
type TagCount struct {
	Tag   string
	Count int
}

// TagCounts returns every declared tag with the number of APIs using it.
func (c *Catalog) TagCounts() []TagCount {
	counts := make(map[string]int, len(c.tagSet))
	for _, api := range c.APIs {
		for _, tag := range api.Tags {
			counts[tag]++
		}
	}
	out := make([]TagCount, 0, len(c.Tags))
	for _, tag := range c.Tags {
		out = append(out, TagCount{Tag: tag, Count: counts[tag]})
	}
	return out
}

// APIsForCategory returns all APIs in a category, in catalog order. An empty
// category returns the whole catalog.
func (c *Catalog) APIsForCategory(category string) []API {
	if category == "" {
		return c.APIs
	}
	out := make([]API, 0, 16)
	for _, api := range c.APIs {
		if api.Category == category {
			out = append(out, api)
		}
	}
	return out
}

// APIsWithTag returns all APIs carrying tag, in catalog order.
func (c *Catalog) APIsWithTag(tag string) []API {
	out := make([]API, 0, 16)
	for _, api := range c.APIs {
		for _, t := range api.Tags {
			if t == tag {
				out = append(out, api)
				break
			}
		}
	}
	return out
}
