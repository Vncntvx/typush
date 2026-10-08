package commands

import (
	"fmt"
	"os"
)

// Execute and Preview name the two dry-run modes, so call sites read as
// CleanOne(name, Preview) instead of CleanOne(name, true).
const (
	Execute bool = false
	Preview bool = true
)

// previewLine writes an informational preview line to stderr without a prefix.
//
// Every piece of dry-run output goes through this file: previews belong on
// stderr so stdout stays reserved for real command output that CI consumes.
func previewLine(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// previewNote writes the single terminal "Dry run: ..." line of a preview.
func previewNote(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Dry run: "+format+"\n", args...)
}

// previewItems writes "Dry run: <header> (N item(s)):" followed by an indented
// list of items, all on stderr.
func previewItems(header string, items []string) {
	previewNote("%s (%d item(s)):", header, len(items))
	for _, item := range items {
		fmt.Fprintf(os.Stderr, "  %s\n", item)
	}
}
