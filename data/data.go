// Package data embeds the bundled offline catalog. User-provided data files
// can be loaded instead with --data or the catalog_dir config option.
package data

import "embed"

// FS contains catalog.json and the catalog module directories.
//
//go:embed all:catalog.json all:apis all:constants all:structures all:ntstatus
var FS embed.FS
