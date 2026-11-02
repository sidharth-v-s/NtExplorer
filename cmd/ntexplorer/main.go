package main

import (
	"fmt"
	"io/fs"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"ntexplorer/data"
	"ntexplorer/internal/app"
	"ntexplorer/internal/catalog"
	"ntexplorer/internal/config"
	"ntexplorer/internal/tui"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "ntexplorer:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	command := "tui"
	dataDir := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "validate-data":
			command = "validate-data"
		case "help", "--help", "-h":
			printHelp()
			return nil
		case "version", "--version":
			fmt.Println("NtExplorer " + tui.Version)
			return nil
		case "--data":
			if i+1 >= len(args) {
				return fmt.Errorf("--data requires a directory")
			}
			i++
			dataDir = args[i]
		default:
			return fmt.Errorf("unknown argument %q (try --help)", args[i])
		}
	}

	cfg, configWarnings, err := config.Load()
	if err != nil {
		// Load already falls back to defaults; keep going.
		configWarnings = append(configWarnings, err.Error())
	}
	if dataDir == "" {
		dataDir = cfg.CatalogDir
	}

	var fsys fs.FS
	source := "embedded"
	if dataDir != "" {
		fsys = os.DirFS(dataDir)
		source = dataDir
	} else {
		fsys = data.FS
	}

	cat, loadIssues, err := catalog.Load(fsys, ".")
	if err != nil {
		return fmt.Errorf("load catalog from %s: %w", source, err)
	}
	issues := append(loadIssues, catalog.Validate(cat)...)

	if command == "validate-data" {
		return reportValidation(cat, issues, source)
	}

	if catalog.HasErrors(issues) {
		return fmt.Errorf("catalog validation failed; run `ntexplorer validate-data` for details:\n%s", catalog.FormatIssues(issues))
	}

	a := app.New(cat, cfg, configWarnings, issues)
	a.DataDir = source
	model := tui.New(a)
	program := tea.NewProgram(model, tea.WithAltScreen())
	_, err = program.Run()
	return err
}

func reportValidation(cat *catalog.Catalog, issues []catalog.Issue, source string) error {
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
	fmt.Printf("source: %s\n", source)
	fmt.Printf("catalog: %s, %d APIs, %d structures, %d constants, %d NTSTATUS values\n",
		cat.CatalogVersion, len(cat.APIs), len(cat.Structures), len(cat.Constants), len(cat.Statuses))
	fmt.Printf("result: %d error(s), %d warning(s)\n", errors, warnings)
	if errors > 0 {
		return fmt.Errorf("catalog validation failed with %d error(s)", errors)
	}
	return nil
}

func printHelp() {
	fmt.Print(`NtExplorer — offline Windows API reference

Usage:
  ntexplorer                 Browse the bundled catalog
  ntexplorer --data DIR      Use an external catalog directory
  ntexplorer validate-data   Validate catalog data and exit
  ntexplorer version         Print the version

The TUI is keyboard driven. Press ? inside the app for every binding.
NtExplorer documents APIs only; it never executes them.
`)
}
