package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/tabwriter"
)

// UniverseOptions are the flags every Universe-reading command shares: JSON
// output on stdout and whether to ignore the cached index. A struct rather than
// positional booleans, so the two cannot be transposed at a call site.
type UniverseOptions struct {
	AsJSON  bool
	Refresh bool
}

// DefaultSearchLimit is the number of search results shown when the caller does
// not ask for a specific count. main.go uses it as the --limit default.
const DefaultSearchLimit = 20

// maxDescriptionRunes caps the description column in the search table.
const maxDescriptionRunes = 60

// ellipsis marks a truncated table cell.
const ellipsis = "..."

// writeJSON encodes v as indented JSON to stdout.
func writeJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// newTable returns a tabwriter over stdout and its flush function.
func newTable() (*tabwriter.Writer, func() error) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	return w, w.Flush
}

// truncateRunes shortens s to at most max runes, appending an ellipsis when it
// has to cut. When max is too small to hold the ellipsis, the result is cut to
// max runes without one.
func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	suffix := []rune(ellipsis)
	if max <= len(suffix) {
		return string(runes[:max])
	}
	return string(runes[:max-len(suffix)]) + ellipsis
}

// workingDir resolves the process working directory once per command, so
// rendering output never repeats the syscall. An empty result means unknown.
var workingDir = sync.OnceValue(func() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
})

// displayPath renders an absolute path relative to the working directory when shorter.
func displayPath(abs string) string {
	cwd := workingDir()
	if cwd == "" {
		return abs
	}
	rel, err := filepath.Rel(cwd, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return abs
	}
	return rel
}

// warnf writes a warning to stderr.
func warnf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "WARN: "+format+"\n", args...)
}

// infof writes one unstyled informational line to stderr. Progress notes and the
// plain lines of a --dry-run preview both go through it, so stdout stays
// reserved for command output.
func infof(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}
