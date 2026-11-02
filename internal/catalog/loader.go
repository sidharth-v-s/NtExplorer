package catalog

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// Load reads a catalog from an fs.FS rooted at root. The root must contain
// catalog.json and an apis/ directory. Optional structures/, constants/ and
// ntstatus/ directories are loaded when present.
//
// Load never panics on bad data: it returns an error for problems that make
// the catalog unusable and a slice of Issues for record-level problems.
func Load(fsys fs.FS, root string) (*Catalog, []Issue, error) {
	if fsys == nil {
		return nil, nil, fmt.Errorf("catalog: nil filesystem")
	}
	root = cleanRoot(root)

	metaRaw, err := fs.ReadFile(fsys, path.Join(root, "catalog.json"))
	if err != nil {
		return nil, nil, fmt.Errorf("catalog: read catalog.json: %w", err)
	}
	var meta Meta
	if err := json.Unmarshal(metaRaw, &meta); err != nil {
		return nil, nil, fmt.Errorf("catalog: parse catalog.json: %w", err)
	}
	if meta.SchemaVersion == 0 {
		return nil, nil, fmt.Errorf("catalog: catalog.json does not declare schema_version")
	}
	if meta.SchemaVersion > SupportedSchemaVersion {
		return nil, nil, fmt.Errorf(
			"catalog: catalog.json schema_version %d is newer than supported version %d; upgrade NtExplorer",
			meta.SchemaVersion, SupportedSchemaVersion)
	}

	c := &Catalog{
		Meta:          meta,
		apiByID:       map[string]*API{},
		structureByID: map[string]*Structure{},
		constantByID:  map[string]*Constant{},
		statusByID:    map[string]*NTStatus{},
		categorySet:   map[string]struct{}{},
		tagSet:        map[string]struct{}{},
	}
	for _, category := range meta.Categories {
		c.categorySet[category.Name] = struct{}{}
	}
	for _, tag := range meta.Tags {
		c.tagSet[tag] = struct{}{}
	}

	var issues []Issue
	if err := c.loadAPIs(fsys, root, &issues); err != nil {
		return nil, issues, err
	}
	if err := c.loadStructures(fsys, root, &issues); err != nil {
		return nil, issues, err
	}
	if err := c.loadConstants(fsys, root, &issues); err != nil {
		return nil, issues, err
	}
	if err := c.loadStatuses(fsys, root, &issues); err != nil {
		return nil, issues, err
	}

	sort.SliceStable(c.APIs, func(i, j int) bool {
		return strings.ToLower(c.APIs[i].Name) < strings.ToLower(c.APIs[j].Name)
	})
	sort.SliceStable(c.Structures, func(i, j int) bool {
		return strings.ToLower(c.Structures[i].Name) < strings.ToLower(c.Structures[j].Name)
	})
	sort.SliceStable(c.Constants, func(i, j int) bool {
		return strings.ToLower(c.Constants[i].Name) < strings.ToLower(c.Constants[j].Name)
	})
	sort.SliceStable(c.Statuses, func(i, j int) bool {
		return strings.ToLower(c.Statuses[i].Name) < strings.ToLower(c.Statuses[j].Name)
	})

	// Sorting moves slice elements, so rebuild the ID indexes afterwards.
	c.reindex()

	return c, issues, nil
}

func (c *Catalog) reindex() {
	c.categorySet = make(map[string]struct{}, len(c.Categories))
	for _, category := range c.Categories {
		c.categorySet[category.Name] = struct{}{}
	}
	c.tagSet = make(map[string]struct{}, len(c.Tags))
	for _, tag := range c.Tags {
		c.tagSet[tag] = struct{}{}
	}
	c.apiByID = make(map[string]*API, len(c.APIs))
	for i := range c.APIs {
		c.apiByID[c.APIs[i].ID] = &c.APIs[i]
	}
	c.structureByID = make(map[string]*Structure, len(c.Structures))
	for i := range c.Structures {
		c.structureByID[c.Structures[i].ID] = &c.Structures[i]
	}
	c.constantByID = make(map[string]*Constant, len(c.Constants))
	for i := range c.Constants {
		c.constantByID[c.Constants[i].ID] = &c.Constants[i]
	}
	c.statusByID = make(map[string]*NTStatus, len(c.Statuses))
	for i := range c.Statuses {
		c.statusByID[c.Statuses[i].ID] = &c.Statuses[i]
	}
}

func cleanRoot(root string) string {
	root = strings.TrimSpace(root)
	root = strings.TrimPrefix(root, "./")
	root = strings.TrimSuffix(root, "/")
	if root == "" {
		return "."
	}
	return root
}

// filePaths lists data files under dir matching pattern. A missing directory
// yields no files and no error, which keeps optional modules optional.
func filePaths(fsys fs.FS, dir, pattern string) ([]string, error) {
	matches, err := fs.Glob(fsys, path.Join(dir, pattern))
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	return matches, nil
}

func (c *Catalog) loadAPIs(fsys fs.FS, root string, issues *[]Issue) error {
	files, err := filePaths(fsys, path.Join(root, "apis"), "*.json")
	if err != nil {
		return fmt.Errorf("catalog: list API files: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("catalog: no API data files found under %s/apis", root)
	}

	for _, file := range files {
		raw, err := fs.ReadFile(fsys, file)
		if err != nil {
			return fmt.Errorf("catalog: read %s: %w", file, err)
		}
		var doc APIFile
		if err := json.Unmarshal(raw, &doc); err != nil {
			return fmt.Errorf("catalog: parse %s: %w", file, err)
		}
		if doc.SchemaVersion == 0 {
			return fmt.Errorf("catalog: %s does not declare schema_version", file)
		}
		if doc.SchemaVersion > SupportedSchemaVersion {
			return fmt.Errorf(
				"catalog: %s schema_version %d is newer than supported version %d",
				file, doc.SchemaVersion, SupportedSchemaVersion)
		}
		for i := range doc.APIs {
			api := doc.APIs[i]
			api.Source = displayPath(file)
			if api.ID == "" {
				*issues = append(*issues, Issue{
					Severity: SeverityError,
					File:     api.Source,
					Field:    "id",
					Message:  fmt.Sprintf("record %d (%q) has an empty id", i+1, api.Name),
				})
				continue
			}
			if _, exists := c.apiByID[api.ID]; exists {
				*issues = append(*issues, Issue{
					Severity: SeverityError,
					File:     api.Source,
					RecordID: api.ID,
					Field:    "id",
					Message:  "duplicate API id; the first definition is kept",
				})
				continue
			}
			c.APIs = append(c.APIs, api)
			stored := &c.APIs[len(c.APIs)-1]
			c.apiByID[api.ID] = stored
		}
	}
	return nil
}

func (c *Catalog) loadStructures(fsys fs.FS, root string, issues *[]Issue) error {
	files, err := filePaths(fsys, path.Join(root, "structures"), "*.json")
	if err != nil {
		return fmt.Errorf("catalog: list structure files: %w", err)
	}
	for _, file := range files {
		raw, err := fs.ReadFile(fsys, file)
		if err != nil {
			return fmt.Errorf("catalog: read %s: %w", file, err)
		}
		var doc StructureFile
		if err := json.Unmarshal(raw, &doc); err != nil {
			return fmt.Errorf("catalog: parse %s: %w", file, err)
		}
		for i := range doc.Structures {
			s := doc.Structures[i]
			s.Source = displayPath(file)
			if s.ID == "" {
				*issues = append(*issues, Issue{
					Severity: SeverityError,
					File:     s.Source,
					Field:    "id",
					Message:  fmt.Sprintf("record %d (%q) has an empty id", i+1, s.Name),
				})
				continue
			}
			if _, exists := c.structureByID[s.ID]; exists {
				*issues = append(*issues, Issue{
					Severity: SeverityError,
					File:     s.Source,
					RecordID: s.ID,
					Field:    "id",
					Message:  "duplicate structure id; the first definition is kept",
				})
				continue
			}
			c.Structures = append(c.Structures, s)
			stored := &c.Structures[len(c.Structures)-1]
			c.structureByID[s.ID] = stored
		}
	}
	return nil
}

func (c *Catalog) loadConstants(fsys fs.FS, root string, issues *[]Issue) error {
	files, err := filePaths(fsys, path.Join(root, "constants"), "*.json")
	if err != nil {
		return fmt.Errorf("catalog: list constant files: %w", err)
	}
	for _, file := range files {
		raw, err := fs.ReadFile(fsys, file)
		if err != nil {
			return fmt.Errorf("catalog: read %s: %w", file, err)
		}
		var doc ConstantFile
		if err := json.Unmarshal(raw, &doc); err != nil {
			return fmt.Errorf("catalog: parse %s: %w", file, err)
		}
		for i := range doc.Constants {
			k := doc.Constants[i]
			k.Source = displayPath(file)
			if k.ID == "" {
				*issues = append(*issues, Issue{
					Severity: SeverityError,
					File:     k.Source,
					Field:    "id",
					Message:  fmt.Sprintf("record %d (%q) has an empty id", i+1, k.Name),
				})
				continue
			}
			if _, exists := c.constantByID[k.ID]; exists {
				*issues = append(*issues, Issue{
					Severity: SeverityError,
					File:     k.Source,
					RecordID: k.ID,
					Field:    "id",
					Message:  "duplicate constant id; the first definition is kept",
				})
				continue
			}
			c.Constants = append(c.Constants, k)
			stored := &c.Constants[len(c.Constants)-1]
			c.constantByID[k.ID] = stored
		}
	}
	return nil
}

func (c *Catalog) loadStatuses(fsys fs.FS, root string, issues *[]Issue) error {
	files, err := filePaths(fsys, path.Join(root, "ntstatus"), "*.json")
	if err != nil {
		return fmt.Errorf("catalog: list NTSTATUS files: %w", err)
	}
	for _, file := range files {
		raw, err := fs.ReadFile(fsys, file)
		if err != nil {
			return fmt.Errorf("catalog: read %s: %w", file, err)
		}
		var doc StatusFile
		if err := json.Unmarshal(raw, &doc); err != nil {
			return fmt.Errorf("catalog: parse %s: %w", file, err)
		}
		for i := range doc.Statuses {
			s := doc.Statuses[i]
			s.Source = displayPath(file)
			if s.ID == "" {
				*issues = append(*issues, Issue{
					Severity: SeverityError,
					File:     s.Source,
					Field:    "id",
					Message:  fmt.Sprintf("record %d (%q) has an empty id", i+1, s.Name),
				})
				continue
			}
			if _, exists := c.statusByID[s.ID]; exists {
				*issues = append(*issues, Issue{
					Severity: SeverityError,
					File:     s.Source,
					RecordID: s.ID,
					Field:    "id",
					Message:  "duplicate NTSTATUS id; the first definition is kept",
				})
				continue
			}
			c.Statuses = append(c.Statuses, s)
			stored := &c.Statuses[len(c.Statuses)-1]
			c.statusByID[s.ID] = stored
		}
	}
	return nil
}

func displayPath(file string) string {
	return strings.TrimPrefix(file, "./")
}
