package util

import "path/filepath"

// RelEntry pairs an absolute path as returned by walk/ListInstall/ListPublish
// with its slash-separated path relative to the walk root.
type RelEntry struct {
	Abs string // absolute path from the walker
	Rel string // slash-separated path relative to the walk root
}

// RelEntries maps walked entries onto root-relative paths, skipping the walk
// root itself and preserving the walker's (sorted) order. root may be relative;
// it is resolved to an absolute path first so it always matches the entries.
func RelEntries(entries []string, root string) ([]RelEntry, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	out := make([]RelEntry, 0, len(entries))
	for _, abs := range entries {
		rel, err := filepath.Rel(absRoot, abs)
		if err != nil {
			return nil, err
		}
		if rel == "." {
			continue // skip the walk root itself
		}
		out = append(out, RelEntry{Abs: abs, Rel: filepath.ToSlash(rel)})
	}
	return out, nil
}
