package catalog

import (
	"strings"
	"testing"
	"testing/fstest"
)

const testMeta = `{
  "schema_version": 1,
  "name": "test catalog",
  "catalog_version": "test",
  "categories": [{"name": "Memory Management", "description": "memory"}],
  "tags": ["memory-management", "analysis-relevant"]
}`

func validAPI(id, name string) API {
	return API{
		ID:          id,
		Name:        name,
		Category:    "Memory Management",
		DLL:         "Kernel32.dll",
		Header:      "memoryapi.h",
		Summary:     "Summary for " + name + ".",
		Description: "Description for " + name + ".",
		Prototype:   "LPVOID " + name + "(LPVOID lpAddress, SIZE_T dwSize);",
		Parameters: []Parameter{
			{Name: "lpAddress", Type: "LPVOID", Description: "Address."},
			{Name: "dwSize", Type: "SIZE_T", Description: "Size."},
		},
		Return:     ReturnInfo{Type: "LPVOID", Success: "pointer", Failure: "NULL"},
		Tags:       []string{"memory-management"},
		References: []Reference{{Title: name + " documentation", URL: "https://learn.microsoft.com/", SourceType: "official-documentation"}},
		Source:     "apis/test.json",
	}
}

func testCatalogFS(apiFiles map[string]string) fstest.MapFS {
	files := fstest.MapFS{
		"catalog.json": {Data: []byte(testMeta)},
	}
	for name, content := range apiFiles {
		files["apis/"+name] = &fstest.MapFile{Data: []byte(content)}
	}
	return files
}

func TestLoadMergesAndSorts(t *testing.T) {
	fsys := testCatalogFS(map[string]string{
		"a.json": `{"schema_version":1,"apis":[
			{"id":"win32.zwrite","name":"ZWrite","category":"Memory Management","dll":"Kernel32.dll","header":"memoryapi.h","summary":"s","description":"d","prototype":"void ZWrite(void);","parameters":[],"return":{"type":"void"},"references":[{"title":"ref","url":"https://learn.microsoft.com/"}]},
			{"id":"win32.aread","name":"ARead","category":"Memory Management","dll":"Kernel32.dll","header":"memoryapi.h","summary":"s","description":"d","prototype":"void ARead(void);","parameters":[],"return":{"type":"void"},"references":[{"title":"ref","url":"https://learn.microsoft.com/"}]}
		]}`,
	})

	cat, issues, err := Load(fsys, ".")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %v", issues)
	}
	if len(cat.APIs) != 2 {
		t.Fatalf("got %d APIs, want 2", len(cat.APIs))
	}
	if cat.APIs[0].Name != "ARead" || cat.APIs[1].Name != "ZWrite" {
		t.Fatalf("APIs not sorted by name: %v, %v", cat.APIs[0].Name, cat.APIs[1].Name)
	}
	if api, ok := cat.API("win32.aread"); !ok || api.Name != "ARead" {
		t.Fatalf("ID index not rebuilt after sort")
	}
}

func TestLoadDuplicateIDReported(t *testing.T) {
	entry := `{"schema_version":1,"apis":[{"id":"win32.dup","name":"Dup","category":"Memory Management","dll":"Kernel32.dll","summary":"s","description":"d","prototype":"void Dup(void);","parameters":[],"return":{"type":"void"},"references":[{"title":"ref","url":"https://learn.microsoft.com/"}]}]}`
	cat, issues, err := Load(testCatalogFS(map[string]string{"a.json": entry, "b.json": entry}), ".")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cat.APIs) != 1 {
		t.Fatalf("duplicate should keep one API, got %d", len(cat.APIs))
	}
	if !HasErrors(issues) {
		t.Fatalf("expected duplicate ID error, got %v", issues)
	}
	if !strings.Contains(FormatIssues(issues), "duplicate API id") {
		t.Fatalf("unexpected issue text: %s", FormatIssues(issues))
	}
}

func TestLoadRejectsFutureSchema(t *testing.T) {
	fsys := fstest.MapFS{
		"catalog.json": {Data: []byte(`{"schema_version":99,"categories":[],"tags":[]}`)},
	}
	if _, _, err := Load(fsys, "."); err == nil || !strings.Contains(err.Error(), "newer") {
		t.Fatalf("expected newer-schema error, got %v", err)
	}
}

func TestLoadMissingMeta(t *testing.T) {
	fsys := fstest.MapFS{"apis/a.json": {Data: []byte(`{"schema_version":1,"apis":[]}`)}}
	if _, _, err := Load(fsys, "."); err == nil {
		t.Fatal("expected error for missing catalog.json")
	}
}

func TestValidateCatchesBadRecords(t *testing.T) {
	api := validAPI("win32.good", "Good")
	bad := validAPI("win32.bad", "Bad")
	bad.Category = "Not A Category"
	bad.Prototype = ""
	bad.Return.Type = ""
	bad.Tags = []string{"Not-Kebab"}
	bad.RelatedAPIs = []string{"win32.missing"}
	bad.Parameters = []Parameter{{Name: "x", Type: "int"}, {Name: "x", Type: "int"}}
	bad.References = []Reference{{Title: "", URL: "not a url"}}

	cat := &Catalog{
		Meta: Meta{
			SchemaVersion:  1,
			Name:           "test",
			CatalogVersion: "test",
			Categories:     []Category{{Name: "Memory Management"}},
			Tags:           []string{"memory-management"},
		},
		APIs: []API{api, bad},
	}
	cat.reindex()
	issues := Validate(cat)
	text := FormatIssues(issues)

	for _, want := range []string{
		"unknown category",
		"prototype",
		"return.type",
		"not lowercase kebab-case",
		"does not resolve",
		"duplicate parameter name",
		"valid http(s) URL",
		"reference has no title",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("validation output missing %q:\n%s", want, text)
		}
	}
	if !HasErrors(issues) {
		t.Error("expected validation errors")
	}
}

func TestValidateAcceptsWellFormedCatalog(t *testing.T) {
	api := validAPI("win32.good", "Good")
	cat := &Catalog{
		Meta: Meta{
			SchemaVersion:  1,
			Name:           "test",
			CatalogVersion: "test",
			Categories:     []Category{{Name: "Memory Management"}},
			Tags:           []string{"memory-management"},
		},
		APIs: []API{api},
	}
	cat.reindex()
	issues := Validate(cat)
	if HasErrors(issues) {
		t.Fatalf("unexpected errors: %s", FormatIssues(issues))
	}
}

func TestValidateStatusRecords(t *testing.T) {
	cat := &Catalog{
		Meta: Meta{Categories: []Category{{Name: "Memory Management"}}},
		Statuses: []NTStatus{
			{
				ID:          "ntstatus.status_access_denied",
				Name:        "STATUS_ACCESS_DENIED",
				Value:       "0xC0000022",
				Severity:    "error",
				Description: "Access denied.",
				References:  []Reference{{Title: "NTSTATUS Values", URL: "https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-erref/", SourceType: "official-documentation"}},
				Source:      "ntstatus/statuses.json",
			},
			{
				ID:       "bad-id",
				Name:     "",
				Value:    "C0000022",
				Severity: "severe",
				Source:   "ntstatus/statuses.json",
			},
			{
				ID:          "ntstatus.status_pending",
				Name:        "STATUS_PENDING",
				Value:       "0xC0000103",
				Severity:    "success",
				Description: "Pending.",
				Source:      "ntstatus/statuses.json",
			},
		},
	}
	issues := Validate(cat)
	text := FormatIssues(issues)
	for _, want := range []string{
		"value must be a 32-bit hexadecimal constant",
		"NTSTATUS record has an empty name",
		"unknown severity",
		"severity \"success\" does not match",
		"id does not follow the lowercase dotted convention",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("validation output missing %q:\n%s", want, text)
		}
	}
}

func TestValidateAcceptsWellFormedStatus(t *testing.T) {
	cat := &Catalog{
		Meta: Meta{Categories: []Category{{Name: "Memory Management"}}},
		Statuses: []NTStatus{{
			ID:          "ntstatus.status_success",
			Name:        "STATUS_SUCCESS",
			Value:       "0x00000000",
			Severity:    "success",
			Description: "The operation completed successfully.",
			References:  []Reference{{Title: "NTSTATUS Values", URL: "https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-erref/", SourceType: "official-documentation"}},
		}},
	}
	if issues := Validate(cat); HasErrors(issues) {
		t.Fatalf("unexpected validation errors:\n%s", FormatIssues(issues))
	}
}
