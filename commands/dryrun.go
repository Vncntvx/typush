package commands

import (
	"fmt"
	"os"
	"strings"
)

// Execute and Preview name the dry-run modes for call sites.
const (
	Execute bool = false
	Preview bool = true
)

// A preview is written to stderr, so stdout stays reserved for command output.
// Plain lines go through infof (output.go); the framing around them comes from
// the two helpers below.

// previewNote writes a "Dry run: ..." line to stderr.
func previewNote(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Dry run: "+format+"\n", args...)
}

// previewItems writes a counted "Dry run: <header>" block followed by indented items to stderr.
func previewItems(header string, items []string) {
	previewNote("%s (%d item(s)):", header, len(items))
	for _, item := range items {
		for _, line := range strings.Split(item, "\n") {
			fmt.Fprintf(os.Stderr, "  %s\n", line)
		}
	}
}
