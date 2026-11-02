package search

import (
	"testing"

	"ntexplorer/internal/catalog"
)

func fixture() []catalog.API {
	return []catalog.API{
		{
			ID:          "win32.virtualalloc",
			Name:        "VirtualAlloc",
			Category:    "Memory Management",
			Summary:     "Reserves or commits virtual memory.",
			Description: "Manages virtual memory in the calling process.",
			Tags:        []string{"memory-management", "analysis-relevant"},
			Parameters:  []catalog.Parameter{{Name: "flProtect", Type: "DWORD", Description: "Memory protection."}},
		},
		{
			ID:          "win32.virtualallocex",
			Name:        "VirtualAllocEx",
			Category:    "Remote Process Memory",
			Summary:     "Allocates memory in another process.",
			Description: "Reserves or commits memory in a target process.",
			Tags:        []string{"remote-memory", "maldev-relevant"},
			Parameters:  []catalog.Parameter{{Name: "hProcess", Type: "HANDLE", Description: "Target process handle."}},
		},
		{
			ID:          "win32.writeprocessmemory",
			Name:        "WriteProcessMemory",
			Aliases:     []string{"WPM"},
			Category:    "Remote Process Memory",
			Summary:     "Writes data into another process's memory.",
			Description: "Copies a buffer into a target process address space.",
			Tags:        []string{"remote-memory"},
			Parameters:  []catalog.Parameter{{Name: "lpBuffer", Type: "LPCVOID", Description: "Source buffer."}},
		},
		{
			ID:          "win32.openprocess",
			Name:        "OpenProcess",
			Category:    "Process Management",
			Summary:     "Opens an existing process object.",
			Description: "Returns a handle to a process with requested access rights.",
			Tags:        []string{"process-enumeration", "analysis-relevant"},
			Parameters:  []catalog.Parameter{{Name: "dwProcessId", Type: "DWORD", Description: "Target process id."}},
		},
	}
}

func names(results []Result) []string {
	out := make([]string, len(results))
	for i, result := range results {
		out[i] = result.API.Name
	}
	return out
}

func TestExactMatchRanksFirst(t *testing.T) {
	results := Query(fixture(), "VirtualAlloc", 0)
	if len(results) < 2 {
		t.Fatalf("expected at least two results, got %v", names(results))
	}
	if results[0].API.Name != "VirtualAlloc" {
		t.Fatalf("exact name should rank first, got %v", names(results))
	}
	if results[1].API.Name != "VirtualAllocEx" {
		t.Fatalf("prefix match should rank second, got %v", names(results))
	}
	if len(results[0].Reasons) == 0 {
		t.Fatal("expected a match reason")
	}
}

func TestCaseInsensitive(t *testing.T) {
	results := Query(fixture(), "virtualallocEX", 0)
	if len(results) == 0 || results[0].API.Name != "VirtualAllocEx" {
		t.Fatalf("case-insensitive search failed: %v", names(results))
	}
}

func TestTagAndCategorySearch(t *testing.T) {
	if got := names(Query(fixture(), "remote-memory", 0)); len(got) != 2 {
		t.Fatalf("tag search: got %v, want the two remote-memory APIs", got)
	}
	if got := names(Query(fixture(), "Process Management", 0)); len(got) == 0 || got[0] != "OpenProcess" {
		t.Fatalf("category phrase should rank category members first: got %v", got)
	}
}

func TestDescriptionAndParameterSearch(t *testing.T) {
	if got := names(Query(fixture(), "target process id", 0)); len(got) != 1 || got[0] != "OpenProcess" {
		t.Fatalf("parameter search: got %v", got)
	}
	if got := names(Query(fixture(), "address space", 0)); len(got) != 1 || got[0] != "WriteProcessMemory" {
		t.Fatalf("description search: got %v", got)
	}
}

func TestAliasSearch(t *testing.T) {
	results := Query(fixture(), "WPM", 0)
	if len(results) != 1 || results[0].API.Name != "WriteProcessMemory" {
		t.Fatalf("alias search failed: %v", names(results))
	}
	for _, reason := range results[0].Reasons {
		if reason == "alias WPM" {
			return
		}
	}
	t.Fatalf("alias reason missing: %v", results[0].Reasons)
}

func TestMultiTermRequiresAllTerms(t *testing.T) {
	results := Query(fixture(), "VirtualAllocEx process", 0)
	if len(results) != 1 || results[0].API.Name != "VirtualAllocEx" {
		t.Fatalf("multi-term search: got %v", names(results))
	}
	if got := Query(fixture(), "VirtualAllocX nonexistent", 0); len(got) != 0 {
		t.Fatalf("unmatched term should exclude results, got %v", names(got))
	}
}

func TestLimit(t *testing.T) {
	if got := Query(fixture(), "", 2); len(got) != 2 {
		t.Fatalf("limit ignored: got %d", len(got))
	}
	if got := Query(fixture(), "virtual", 1); len(got) != 1 {
		t.Fatalf("limit ignored: got %d", len(got))
	}
}

func TestEmptyQueryReturnsAll(t *testing.T) {
	results := Query(fixture(), "  ", 0)
	if len(results) != 4 {
		t.Fatalf("empty query: got %d results", len(results))
	}
}

func structureFixture() []catalog.Structure {
	return []catalog.Structure{
		{
			ID:         "win32.startupinfow",
			Name:       "STARTUPINFOW",
			Summary:    "Specifies startup settings for a new process.",
			Definition: "typedef struct _STARTUPINFOW {\n  DWORD cb;\n  HANDLE hStdInput;\n} STARTUPINFOW;",
			Members: []catalog.Member{
				{Name: "cb", Type: "DWORD", Description: "Size of the structure."},
				{Name: "hStdInput", Type: "HANDLE", Description: "Standard input handle."},
			},
			Tags: []string{"process-creation"},
		},
		{
			ID:      "win32.process_information",
			Name:    "PROCESS_INFORMATION",
			Summary: "Receives handles and identifiers for a new process.",
			Members: []catalog.Member{
				{Name: "hProcess", Type: "HANDLE", Description: "Handle to the new process."},
				{Name: "dwProcessId", Type: "DWORD", Description: "Identifier for the new process."},
			},
			Tags: []string{"process-creation", "process-enumeration"},
		},
		{
			ID:      "win32.security_attributes",
			Name:    "SECURITY_ATTRIBUTES",
			Summary: "Specifies a security descriptor and handle inheritance.",
			Members: []catalog.Member{
				{Name: "bInheritHandle", Type: "BOOL", Description: "Whether handles can be inherited."},
			},
			Tags: []string{"process-creation"},
		},
	}
}

func structureNames(results []StructureResult) []string {
	out := make([]string, len(results))
	for i, result := range results {
		out[i] = result.Structure.Name
	}
	return out
}

func TestStructureExactNameRanksFirst(t *testing.T) {
	results := QueryStructures(structureFixture(), "startupinfow", 0)
	if len(results) == 0 || results[0].Structure.Name != "STARTUPINFOW" {
		t.Fatalf("exact structure name should rank first: %v", structureNames(results))
	}
	if len(results[0].Reasons) == 0 {
		t.Fatal("expected a match reason")
	}
}

func TestStructureMemberSearch(t *testing.T) {
	results := QueryStructures(structureFixture(), "hProcess", 0)
	if len(results) != 1 || results[0].Structure.Name != "PROCESS_INFORMATION" {
		t.Fatalf("member search: %v", structureNames(results))
	}
	found := false
	for _, reason := range results[0].Reasons {
		if reason == "member hProcess" {
			found = true
		}
	}
	if !found {
		t.Fatalf("member reason missing: %v", results[0].Reasons)
	}
}

func TestStructureTagAndDescriptionSearch(t *testing.T) {
	if got := structureNames(QueryStructures(structureFixture(), "process-enumeration", 0)); len(got) != 1 || got[0] != "PROCESS_INFORMATION" {
		t.Fatalf("tag search: %v", got)
	}
	if got := structureNames(QueryStructures(structureFixture(), "inheritance", 0)); len(got) != 1 || got[0] != "SECURITY_ATTRIBUTES" {
		t.Fatalf("summary search: %v", got)
	}
}

func TestStructureMultiTermAndLimit(t *testing.T) {
	if got := structureNames(QueryStructures(structureFixture(), "process identifier", 0)); len(got) != 1 || got[0] != "PROCESS_INFORMATION" {
		t.Fatalf("multi-term structure search: %v", got)
	}
	if got := QueryStructures(structureFixture(), "", 2); len(got) != 2 {
		t.Fatalf("structure limit ignored: got %d", len(got))
	}
	if got := QueryStructures(structureFixture(), "nonexistent", 0); len(got) != 0 {
		t.Fatalf("unexpected matches: %v", structureNames(got))
	}
}

func constantFixture() []catalog.Constant {
	return []catalog.Constant{
		{
			ID:          "win32.mem_commit",
			Name:        "MEM_COMMIT",
			Value:       "0x00001000",
			Category:    "Memory Management",
			Description: "Allocates committed pages in the specified region.",
			Tags:        []string{"memory-management", "memory-protection"},
			References:  []catalog.Reference{{Title: "VirtualAlloc function", URL: "https://learn.microsoft.com/en-us/windows/win32/api/memoryapi/nf-memoryapi-virtualalloc"}},
		},
		{
			ID:          "win32.mem_reserve",
			Name:        "MEM_RESERVE",
			Value:       "0x00002000",
			Category:    "Memory Management",
			Description: "Reserves a range of the process address space.",
			Tags:        []string{"memory-management"},
		},
		{
			ID:          "win32.page_readwrite",
			Name:        "PAGE_READWRITE",
			Value:       "0x04",
			Category:    "Memory Management",
			Description: "Enables read and write access to the committed region.",
			Tags:        []string{"memory-protection"},
		},
		{
			ID:          "win32.page_noaccess",
			Name:        "PAGE_NOACCESS",
			Value:       "0x01",
			Category:    "Memory Management",
			Description: "Disables all access to the committed region.",
			Tags:        []string{"memory-protection"},
		},
	}
}

func constantNames(results []ConstantResult) []string {
	out := make([]string, len(results))
	for i, result := range results {
		out[i] = result.Constant.Name
	}
	return out
}

func TestConstantExactNameRanksFirst(t *testing.T) {
	results := QueryConstants(constantFixture(), "MEM_COMMIT", 0)
	if len(results) == 0 || results[0].Constant.Name != "MEM_COMMIT" {
		t.Fatalf("exact constant name should rank first: %v", constantNames(results))
	}
	if len(results[0].Reasons) == 0 {
		t.Fatal("expected a match reason")
	}
}

func TestConstantNumericValueForms(t *testing.T) {
	for _, query := range []string{"0x1000", "4096", "0x00001000", "0X00001000"} {
		results := QueryConstants(constantFixture(), query, 0)
		if len(results) == 0 || results[0].Constant.Name != "MEM_COMMIT" {
			t.Fatalf("value query %q: %v", query, constantNames(results))
		}
	}
	if got := constantNames(QueryConstants(constantFixture(), "0x2000", 0)); len(got) != 1 || got[0] != "MEM_RESERVE" {
		t.Fatalf("hex value search: %v", got)
	}
}

func TestConstantTagAndCategorySearch(t *testing.T) {
	if got := constantNames(QueryConstants(constantFixture(), "memory-protection", 0)); len(got) != 3 {
		t.Fatalf("tag search: %v", got)
	}
	if got := constantNames(QueryConstants(constantFixture(), "Memory Management", 0)); len(got) != 4 {
		t.Fatalf("category phrase search: %v", got)
	}
}

func TestConstantMultiTermAndLimit(t *testing.T) {
	if got := constantNames(QueryConstants(constantFixture(), "RESERVE address", 0)); len(got) != 1 || got[0] != "MEM_RESERVE" {
		t.Fatalf("multi-term constant search: %v", got)
	}
	if got := QueryConstants(constantFixture(), "MEM", 2); len(got) != 2 {
		t.Fatalf("constant limit ignored: got %d", len(got))
	}
	if got := QueryConstants(constantFixture(), "zzzznotaconstant", 0); len(got) != 0 {
		t.Fatalf("unexpected matches: %v", constantNames(got))
	}
}

func TestConstantEmptyQueryReturnsAll(t *testing.T) {
	results := QueryConstants(constantFixture(), "  ", 0)
	if len(results) != 4 {
		t.Fatalf("empty query: got %d results", len(results))
	}
}
