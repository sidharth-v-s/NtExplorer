# Contributing to NtExplorer

NtExplorer values accuracy over catalog size. A small number of well-sourced entries is better than a large list of uncertain definitions.

## Before you start

- Read the product specification (`NtExplorer_SPEC.md`) and `docs/data-format.md`.
- Check existing records for style and vocabulary.
- For API changes, verify prototypes and parameter order against Microsoft Learn or the Windows SDK headers. Do not copy signatures from unverified sources.

## Development setup

```bash
go run ./cmd/ntexplorer          # run the TUI
go test ./...                    # unit + TUI tests
go run ./scripts/validate-data.go
go vet ./...
```

The catalog is embedded from `data/`. During development you can point at an external catalog without rebuilding:

```bash
go run ./cmd/ntexplorer --data ./data
```

## Adding or editing catalog entries

1. Put the record in the category file that best matches it (`data/apis/*.json`).
2. Keep the exact Windows casing of names, types, and constants. Distinguish `A` and `W` variants.
3. Use `[]` for documented zero-parameter functions.
4. Link only IDs that actually exist. Cross-file links are validated.
5. Add at least one reliable reference and a `source_notes` line describing what you verified.
6. Use approved tags from `data/catalog.json`; never add judgemental tags such as `evil-api`.
7. Keep MalDev notes neutral: describe how the API appears in analysis, not how to abuse it.
8. Write examples only when they clarify normal use. They must be benign, include cleanup, and never assemble an offensive workflow. NtExplorer never executes them.

## Quality bar for prose

- Original wording; do not paste documentation sections.
- Explain behavior, constraints, and failure modes that matter to a reader.
- Separate documented contract behavior from implementation observations.
- If behavior varies by Windows version, say so explicitly.
- Omit fields when the information is not genuinely verified.

## Tests

Every change should pass:

```bash
gofmt -l .
go vet ./...
go test ./...
go run ./scripts/validate-data.go
```

When adding validation rules or search behavior, add focused tests in the relevant package. TUI changes should keep the model testable without a real terminal; see `internal/tui/tui_test.go`.

## Commits and review

- Keep commits focused: code, data, and docs separately when practical.
- Mention the affected API IDs in data commits.
- In review, accuracy and provenance outrank volume.

## Reporting problems

Include the API ID, the field that looks wrong, a source URL, and the catalog version shown in the status bar. Data corrections with a citation are the most useful reports.
