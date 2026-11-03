# Catalog data format

The catalog is a set of versioned JSON files bundled into the executable. Data files are prose plus structured metadata; nothing in them is executed.

## Layout

```
data/
├── catalog.json          catalog metadata, categories, tag vocabulary
├── apis/*.json           API records, split by topic
├── structures/*.json     structure definitions, members, and notes
├── constants/*.json      named constants and flags
└── ntstatus/*.json       NTSTATUS values, severity, and references
```

Every file declares a `schema_version`. The current supported version is **1**. A newer version is rejected with an upgrade message rather than being misinterpreted.

## catalog.json

```json
{
  "schema_version": 1,
  "name": "NtExplorer bundled catalog",
  "catalog_version": "0.5.0",
  "categories": [
    { "name": "Memory Management", "description": "Virtual memory ..." }
  ],
  "tags": ["memory-management", "remote-memory", "analysis-relevant"]
}
```

- `categories` defines the valid values for an API's `category`; each API has exactly one primary category.
- `tags` defines the approved MalDev / analysis vocabulary. Tags are lowercase kebab-case and are reusable across categories.
- `catalog_version` is displayed in the TUI and embedded in JSON exports.

## API files

```json
{
  "schema_version": 1,
  "apis": [ { "...": "see below" } ]
}
```

### API record

| Field | Required | Notes |
|---|---|---|
| `id` | yes | Stable lowercase dotted ID, e.g. `win32.virtualalloc`. Unique across the catalog. |
| `name` | yes | Exact documented spelling, e.g. `VirtualAllocEx`. Preserve `A`/`W` variants. |
| `aliases` | no | Other names users may search for. Do not use aliases to mask duplicates. |
| `category` | yes | Must match a category declared in `catalog.json`. |
| `dll` | yes | Documented DLL, e.g. `Kernel32.dll`. |
| `library` | no | Import library, e.g. `Kernel32.lib`. |
| `header` | no | Windows SDK header, e.g. `memoryapi.h`. |
| `minimum_windows` | no | Only when verified against documentation. |
| `summary` | yes | One sentence. |
| `description` | yes | Complete behavior explanation in original wording. |
| `prototype` | yes | Exact C/C++ declaration, including return type and parameters. |
| `parameters` | yes | Ordered array; use `[]` for documented zero-parameter functions. |
| `return` | yes | Return type plus success/failure behavior. |
| `tags` | no | Values from the declared tag vocabulary. |
| `related_apis` | no | API IDs that must resolve. |
| `related_structures` | no | Structure IDs; unresolved IDs are errors once the structure catalog is populated. |
| `related_constants` | no | Constant IDs from `data/constants`. |
| `native_api` | no | Only when the relationship is sourced: `{name, module, confidence, notes}`. |
| `unicode_notes` | no | ANSI/Unicode variant guidance. |
| `remarks` | no | Limitations, common misconceptions, lifetime rules. |
| `maldev_notes` | no | Neutral explanation of why analysts encounter the API. Never operational abuse guidance. |
| `defensive_notes` | no | Telemetry or investigation context, stated conservatively. |
| `example` | yes | `{language, title, code, notes}`; language is `c` or `cpp`. Every bundled API has one. |
| `references` | yes | At least one `{title, url, source_type}` with an http(s) URL. |
| `source_notes` | no | Provenance and version-specific caveats. |

### Parameter record

```json
{
  "name": "flAllocationType",
  "type": "DWORD",
  "description": "Flags controlling reservation and commitment.",
  "constants": ["MEM_RESERVE", "MEM_COMMIT"]
}
```

Parameter names must be unique within an API. `constants` lists constant *names* for display; catalog cross-links use `related_constants` IDs.

### Return record

```json
{
  "type": "LPVOID",
  "success": "Pointer to the base of the allocated region.",
  "failure": "NULL; call GetLastError."
}
```

### Example record

```json
{
  "language": "cpp",
  "title": "Reserve and commit one page",
  "code": "#include <windows.h>\n\nint main() { /* ... */ }\n",
  "notes": "Illustrative only; NtExplorer never executes examples."
}
```

Examples must be benign, self-contained, demonstrate correct cleanup, and avoid multi-step offensive workflows. Every example must call the API it documents. Prefer the current process over another process. Privileged or destructive operations must act only on resources the example creates itself and removes afterward, or gate the call behind an explicit opt-in flag; the `notes` field must state what is changed and how it is cleaned up. Kernel-mode-only routines may describe inputs in the code when the call cannot be made from user mode, but the example should still show the call site and explain the constraint in `notes`.

## Constants

Constants live in `data/constants/*.json`. Each record needs a stable ID, a name, and a value; tags and at least one reference are strongly recommended and present on every bundled record.

```json
{
  "id": "win32.mem_commit",
  "name": "MEM_COMMIT",
  "value": "0x00001000",
  "category": "Memory Management",
  "description": "Allocates committed pages ...",
  "tags": ["memory-management", "analysis-relevant"],
  "references": [
    {
      "title": "VirtualAlloc function",
      "url": "https://learn.microsoft.com/en-us/windows/win32/api/memoryapi/nf-memoryapi-virtualalloc",
      "source_type": "official-documentation"
    }
  ]
}
```

Values are display strings; keep the documented representation (hex for flags, decimal for enumerations). Search compares numeric values, so `0x1000`, `4096`, and `0x00001000` are equivalent. APIs link to constants through `related_constants` IDs; parameter `constants` lists also contribute to the reverse link shown on the constant page.

## Structures

Structure records live in `data/structures/*.json`. Each record needs a stable ID, a display name, a summary, and at least one reference. `definition` and `members` document the C shape; `notes` carries usage and lifetime caveats, and `tags` use the same declared vocabulary as APIs.

```json
{
  "id": "win32.memory_basic_information",
  "name": "MEMORY_BASIC_INFORMATION",
  "summary": "Describes a range of pages.",
  "definition": "typedef struct _MEMORY_BASIC_INFORMATION {\n  PVOID BaseAddress;\n  ...\n} MEMORY_BASIC_INFORMATION;",
  "members": [
    { "name": "BaseAddress", "type": "PVOID", "description": "..." }
  ],
  "notes": "VirtualQuery fills this structure.",
  "tags": ["memory-management", "analysis-relevant"],
  "references": [
    {
      "title": "MEMORY_BASIC_INFORMATION structure",
      "url": "https://learn.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-memory_basic_information",
      "source_type": "official-documentation"
    }
  ]
}
```

APIs link to structures through `related_structures`. The TUI computes the reverse link at runtime, so a structure's related-API list always reflects the current API records.

## Native APIs

Native/RTL records use the same API schema and live in `data/apis/native_apis.json` with the `Native APIs` category. Their prototypes come from phnt or the Windows SDK, their behavior is checked against NtDoc and the linked Microsoft DDI page where one exists, and `source_notes` records that the contract is not a stable Win32 guarantee. Native APIs keep functional tags such as `remote-memory` or `kernel-mode` in addition to `native-api`.

## NTSTATUS

Each file contains a `schema_version` and a `statuses` array. The bundled catalog mirrors the Microsoft MS-ERREF NTSTATUS table, including `STATUS_*`, `DBG_*`, and `RPC_NT_*` values.

```json
{
  "id": "ntstatus.status_access_denied",
  "name": "STATUS_ACCESS_DENIED",
  "value": "0xC0000022",
  "description": "The caller does not have the access rights required for the object or operation.",
  "severity": "error",
  "references": [
    {
      "title": "NTSTATUS Values (MS-ERREF)",
      "url": "https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-erref/596a1078-e883-4972-9bbc-49e60bebca55",
      "source_type": "official-documentation"
    }
  ]
}
```

- `value` is always an eight-digit hexadecimal NTSTATUS constant.
- `severity` is `success`, `informational`, `warning`, or `error`, derived from bits 30 and 31 of the value.
- Descriptions are concise summaries. Curated entries use original wording; values present only in MS-ERREF use the table's short definition.
- Every bundled record carries at least one Microsoft reference. The shipped catalog uses the MS-ERREF documentation page and its MicrosoftDocs source file.

## Validation rules

`go run ./scripts/validate-data.go` and `ntexplorer validate-data` check:

1. Every file parses and declares a supported schema version.
2. API/structure/constant/NTSTATUS IDs are non-empty and unique.
3. API names are non-empty, unique, and whitespace-free.
4. Categories and tags are declared; tags are lowercase kebab-case.
5. APIs have summary, description, prototype, and return type.
6. Parameter names are non-empty and unique within an API.
7. `related_apis`, `related_constants`, and `related_structures` resolve against the bundled catalog.
8. References include a title and a valid http(s) URL; APIs need at least one, and every bundled structure, constant, and NTSTATUS record currently carries one.
9. Examples use `c`/`cpp`, have code, and explain themselves; the bundled catalog test requires an example on every API.
10. NTSTATUS values use eight-digit hexadecimal form, severity is one of the four documented labels and matches bits 30-31, and descriptions are present.
11. IDs follow the lowercase dotted convention (warning only).

Issues identify the source file, record ID, and field. A malformed catalog aborts normal startup with a readable error; a malformed individual record is reported rather than crashing.

## Adding an API

1. Choose the category file that best matches the API.
2. Add a record with every required field.
3. Search for existing related IDs before linking; never link names that do not exist.
4. Add a Microsoft Learn reference (or SDK header) and a short `source_notes`.
5. Run both validation commands and the test suite.
6. Keep wording original — do not paste large documentation sections.
