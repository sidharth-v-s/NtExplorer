# Architecture

NtExplorer uses a small layered architecture so catalog data and business rules stay out of the terminal UI.

```
┌───────────────────────────────────────────────┐
│ Bubble Tea TUI (internal/tui)                 │
│ screens · navigation · key bindings · styles  │
└───────────────────────┬───────────────────────┘
                        │ method calls only
┌───────────────────────▼───────────────────────┐
│ Application layer (internal/app)              │
│ search · browsing · favorites · recent items  │
└───────────────────────┬───────────────────────┘
                        │
┌───────────────────────▼───────────────────────┐
│ Catalog layer (internal/catalog, search)      │
│ models · loader · validator · ranking         │
└───────────────────────┬───────────────────────┘
                        │
┌───────────────────────▼───────────────────────┐
│ Data layer (data/, internal/config, export)   │
│ embedded JSON · schema checks · user state    │
└───────────────────────────────────────────────┘
```

## Packages

| Package | Responsibility |
|---|---|
| `cmd/ntexplorer` | Flag/subcommand parsing, catalog source selection, program startup. |
| `internal/app` | Combines the catalog with API/structure/constant search, favorites, recent entries, and startup warnings. The TUI calls only this layer. |
| `internal/catalog` | JSON models, embedded/external loading, cross-file merging, schema and record validation. |
| `internal/search` | Case-insensitive, multi-term ranked API, structure, and constant search with match reasons. |
| `internal/tui` | Bubble Tea model, update loop, panel layout, overlays, styles, centralized keys. |
| `internal/config` | Optional `config.json` and `state.json` in the standard user config directory. |
| `internal/export` | JSON and Markdown rendering, safe file writing. |
| `data` | `embed.FS` for the bundled offline catalog. |

Dependencies point downward only: `tui → app → catalog/search → data/config`. The TUI never parses JSON; the catalog package never imports Bubble Tea.

## Loading flow

1. `main` loads optional configuration and chooses a catalog source (`--data`, `catalog_dir`, or the embedded FS).
2. `catalog.Load` reads `catalog.json`, merges every `apis/*.json`, then loads optional `structures`, `constants`, and `ntstatus` modules. Duplicate IDs are reported and the first definition is kept.
3. `catalog.Validate` checks record-level rules. Errors abort normal startup with a readable message; warnings are shown in the UI. The embedded-catalog test also requires an example on every API.
4. `app.New` loads favorites/recent state (treating invalid state as empty).
5. `tui.New` builds the sidebar and initial selection, then Bubble Tea runs the model.

## TUI model

- Panels: categories (hidden below 100 columns), list, and a detail viewport. Below 72 columns the list and detail stack vertically.
- Modes: API browse, API search, tag list, structure browse, structure search, constant browse, constant search, and an NTSTATUS placeholder. Native/RTL routines are ordinary API records in the `Native APIs` category.
- Overlays: help, related APIs, references, and export. Overlays own key handling while open.
- The detail viewport is re-rendered with the current panel width on resize and when the selected entry changes. API, structure, and constant detail state are mutually exclusive.
- Search is scoped to the active view: `/` searches structures while the structure list is active, constants while the constant list is active, and APIs elsewhere.
- Related links are computed at runtime in both directions: APIs list their documented structures and constants, and each structure or constant lists the APIs that reference it.
- Search input mode routes ordinary characters to the text field; navigation keys still move the result selection.

## State and writes

Only two things are written to disk by the app itself:

- `state.json` (favorites, recent entries, last category), saved atomically with `0600` permissions.
- Explicit exports chosen by the user, written under `exports/` relative to the working directory. Existing files are never overwritten silently; the export overlay asks for confirmation.

No telemetry, network access, or automatic catalog updates occur in this release.

## Error policy

- Malformed JSON or an unsupported schema version is a startup error with the source file named.
- Invalid individual records produce validation issues with file/record/field context.
- Invalid or missing config/state is non-fatal.
- Rendering is defensive at small terminal sizes and never panics on empty data.
