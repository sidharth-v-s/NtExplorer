package catalog_test

import (
	"testing"

	"ntexplorer/data"
	"ntexplorer/internal/catalog"
)

// TestEmbeddedCatalogIsValid keeps the shipped data honest: every build must
// carry a catalog that passes the same validation as `validate-data`.
func TestEmbeddedCatalogIsValid(t *testing.T) {
	cat, loadIssues, err := catalog.Load(data.FS, ".")
	if err != nil {
		t.Fatalf("load embedded catalog: %v", err)
	}
	issues := append(loadIssues, catalog.Validate(cat)...)
	if catalog.HasErrors(issues) {
		t.Fatalf("embedded catalog has validation errors:\n%s", catalog.FormatIssues(issues))
	}
	if len(cat.APIs) < 370 {
		t.Errorf("embedded catalog has %d APIs, release 0.4 requires at least 370", len(cat.APIs))
	}
	if len(cat.Structures) < 10 {
		t.Errorf("embedded catalog has %d structures, release 0.2 requires at least 10", len(cat.Structures))
	}
	if len(cat.Constants) < 50 {
		t.Errorf("embedded catalog has %d constants, release 0.3 requires at least 50", len(cat.Constants))
	}
	if len(cat.Statuses) < 100 {
		t.Errorf("embedded catalog has %d NTSTATUS values, release 0.5 requires at least 100", len(cat.Statuses))
	}
	for _, status := range cat.Statuses {
		if len(status.References) == 0 {
			t.Errorf("NTSTATUS %s has no references", status.ID)
		}
		if status.Value == "" || status.Severity == "" || status.Description == "" {
			t.Errorf("NTSTATUS %s is missing value, severity, or description", status.ID)
		}
	}
	if len(cat.Categories) == 0 || len(cat.Tags) == 0 {
		t.Error("catalog metadata is incomplete")
	}
	for _, constant := range cat.Constants {
		if len(constant.References) == 0 {
			t.Errorf("constant %s has no references", constant.ID)
		}
		if constant.Value == "" {
			t.Errorf("constant %s has no value", constant.ID)
		}
	}
	for _, api := range cat.APIs {
		if len(api.References) == 0 {
			t.Errorf("%s has no references", api.ID)
		}
		if api.DLL == "" {
			t.Errorf("%s has no DLL", api.ID)
		}
		if api.Example == nil || api.Example.Code == "" {
			t.Errorf("%s has no example", api.ID)
		}
	}
}
