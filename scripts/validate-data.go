// Command validate-data checks the bundled catalog and exits non-zero if any
// errors are found. Run with:
//
//	go run ./scripts/validate-data.go
package main

import (
	"fmt"
	"os"

	"ntexplorer/data"
	"ntexplorer/internal/catalog"
)

func main() {
	cat, loadIssues, err := catalog.Load(data.FS, ".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "catalog load failed:", err)
		os.Exit(1)
	}
	issues := append(loadIssues, catalog.Validate(cat)...)

	errors, warnings := 0, 0
	for _, issue := range issues {
		if issue.Severity == catalog.SeverityError {
			errors++
		} else {
			warnings++
		}
	}
	if len(issues) > 0 {
		fmt.Println(catalog.FormatIssues(issues))
		fmt.Println()
	}
	fmt.Printf("catalog %s: %d APIs, %d structures, %d constants, %d NTSTATUS values\n",
		cat.CatalogVersion, len(cat.APIs), len(cat.Structures), len(cat.Constants), len(cat.Statuses))
	fmt.Printf("result: %d error(s), %d warning(s)\n", errors, warnings)
	if errors > 0 {
		os.Exit(1)
	}
}
