package app

import (
	"testing"

	"ntexplorer/data"
	"ntexplorer/internal/catalog"
	"ntexplorer/internal/config"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	t.Setenv("NTEXPLORER_CONFIG_DIR", t.TempDir())
	cat, _, err := catalog.Load(data.FS, ".")
	if err != nil {
		t.Fatalf("load embedded catalog: %v", err)
	}
	return New(cat, config.Default(), nil, nil)
}

func apiNames(apis []catalog.API) []string {
	names := make([]string, len(apis))
	for i, api := range apis {
		names[i] = api.Name
	}
	return names
}

func hasName(names []string, want string) bool {
	for _, name := range names {
		if name == want {
			return true
		}
	}
	return false
}

func TestBrowseAndResolveConstants(t *testing.T) {
	a := newTestApp(t)
	constants := a.BrowseConstants()
	if len(constants) < 50 {
		t.Fatalf("expected the bundled constants, got %d", len(constants))
	}
	constant, ok := a.Constant("win32.mem_commit")
	if !ok || constant.Name != "MEM_COMMIT" || constant.Value != "0x00001000" {
		t.Fatalf("unexpected constant: %+v (found %v)", constant, ok)
	}
	if _, ok := a.Constant("win32.does_not_exist"); ok {
		t.Fatal("unknown constant should not resolve")
	}
}

func TestSearchConstants(t *testing.T) {
	a := newTestApp(t)
	for _, query := range []string{"MEM_COMMIT", "0x1000", "4096"} {
		results := a.SearchConstants(query)
		if len(results) == 0 || results[0].Constant.Name != "MEM_COMMIT" {
			t.Fatalf("query %q did not rank MEM_COMMIT first: %+v", query, results)
		}
	}
	if got := a.SearchConstants("zzzznotaconstant"); len(got) != 0 {
		t.Fatalf("unexpected constant results: %+v", got)
	}
}

func TestBrowseAndSearchNativeAPIs(t *testing.T) {
	a := newTestApp(t)

	native := a.Browse("Native APIs")
	if len(native) < 50 {
		t.Fatalf("expected the Native APIs category, got %d records", len(native))
	}

	results := a.Search("NtAllocateVirtualMemory")
	if len(results) == 0 || results[0].API.Name != "NtAllocateVirtualMemory" {
		t.Fatalf("native name search did not rank the API first: %+v", results)
	}

	results = a.Search("RtlGetVersion")
	if len(results) == 0 || results[0].API.Name != "RtlGetVersion" {
		t.Fatalf("RTL name search did not rank the API first: %+v", results)
	}
}

func TestConstantRelatedAPIs(t *testing.T) {
	a := newTestApp(t)

	names := apiNames(a.ConstantRelatedAPIs("win32.mem_commit"))
	for _, want := range []string{"VirtualAlloc", "VirtualAllocEx", "VirtualQuery"} {
		if !hasName(names, want) {
			t.Errorf("MEM_COMMIT reverse links missing %s: %v", want, names)
		}
	}

	// PAGE_EXECUTE_READWRITE is linked through parameter symbol lists rather
	// than related_constants, so this exercises the second link source.
	paramOnly := apiNames(a.ConstantRelatedAPIs("win32.page_execute_readwrite"))
	for _, want := range []string{"VirtualAlloc", "VirtualProtect"} {
		if !hasName(paramOnly, want) {
			t.Errorf("parameter-only reverse links missing %s: %v", want, paramOnly)
		}
	}

	if got := a.ConstantRelatedAPIs("win32.does_not_exist"); len(got) != 0 {
		t.Fatalf("unknown constant returned APIs: %v", apiNames(got))
	}
}
