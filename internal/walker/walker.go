// Package walker lists files for publish/install.
//
// Semantics (compatible with the Rust original):
//   - always skip .git/
//   - skip dotfiles/dotdirs (hidden) except the root itself; .typstignore
//     files are read for patterns but never returned.
//   - respect .typstignore files (root + nested), gitignore-ish syntax.
//   - install additionally filters manifest package.exclude globs matched
//     against the slash-separated path relative to root.
package walker

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// ListPublish returns absolute paths (dirs + files, including root) for publishing.
func ListPublish(root string) ([]string, error) {
	return walk(root, nil)
}

// ListInstall returns absolute paths filtered by exclude globs.
func ListInstall(root string, excludes []string) ([]string, error) {
	for _, p := range excludes {
		if strings.TrimSpace(p) == "" {
			return nil, fmt.Errorf("invalid empty exclude pattern")
		}
		if _, err := doublestar.Match(p, "x"); err != nil {
			return nil, fmt.Errorf("invalid exclude pattern %q: %w", p, err)
		}
	}
	return walk(root, excludes)
}

type ignoreSet struct {
	patterns []string
}

func loadIgnoreFile(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		// strip leading ./ and trailing spaces
		t = strings.TrimPrefix(t, "./")
		out = append(out, t)
	}
	return out
}

func matchIgnore(pattern, rel string, isDir bool) bool {
	neg := false
	if strings.HasPrefix(pattern, "!") {
		neg = true
		pattern = pattern[1:]
		_ = neg // negation not fully supported; treat as non-match (conservative include)
		return false
	}
	pattern = strings.TrimSuffix(pattern, "\r")
	if pattern == "" {
		return false
	}
	// Directory-only pattern "foo/" matches "foo" and "foo/..."
	if strings.HasSuffix(pattern, "/") {
		base := strings.TrimSuffix(pattern, "/")
		if rel == base || strings.HasPrefix(rel, base+"/") {
			return true
		}
	}
	// Plain basename without slash matches at any depth, gitignore-style.
	if !strings.Contains(pattern, "/") {
		base := rel
		if i := strings.LastIndex(rel, "/"); i >= 0 {
			base = rel[i+1:]
		}
		if ok, _ := doublestar.Match(pattern, base); ok {
			return true
		}
		if isDir && (rel == pattern || strings.HasPrefix(rel, pattern+"/")) {
			return true
		}
		return false
	}
	// Path pattern: match against full rel, and as prefix for dirs.
	if ok, _ := doublestar.Match(pattern, rel); ok {
		return true
	}
	// "**/x" style already handled by doublestar; also try prefix match for dirs under it.
	if isDir {
		if ok, _ := doublestar.Match(pattern, rel+"/"); ok {
			return true
		}
		// pattern "a/b" should ignore everything under "a/b/"
		if strings.HasPrefix(rel, strings.TrimSuffix(pattern, "/")+"/") {
			// only when pattern has no wildcards
			if !strings.ContainsAny(pattern, "*?[") {
				return true
			}
		}
	}
	return false
}

func isHiddenName(name string) bool {
	return strings.HasPrefix(name, ".") && name != "." && name != ".."
}

func walk(root string, excludes []string) ([]string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	var out []string
	// stack of ignore patterns per directory
	type frame struct {
		dir      string
		patterns []string
	}
	_ = frame{}

	var walkFn func(dir string, inherited []string) error
	walkFn = func(dir string, inherited []string) error {
		// accumulate patterns from this dir's .typstignore
		cur := append([]string{}, inherited...)
		cur = append(cur, loadIgnoreFile(filepath.Join(dir, ".typstignore"))...)

		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		// include dir itself (except we add root before recursion)
		for _, e := range entries {
			name := e.Name()
			full := filepath.Join(dir, name)
			rel, _ := filepath.Rel(absRoot, full)
			relSlash := filepath.ToSlash(rel)

			// never return .git or dotfiles; still need to skip traversal
			if name == ".git" {
				continue
			}
			if isHiddenName(name) && name != ".typstignore" {
				// skip hidden files/dirs entirely (matches ignore crate standard filters)
				if e.IsDir() {
					continue
				}
				continue
			}
			if name == ".typstignore" {
				continue // read but never returned
			}

			isDir := e.IsDir()
			if isDir {
				// check symlink-to-dir? ReadDir FileInfo: use entry type
				if e.Type()&fs.ModeSymlink != 0 {
					// include symlink entries as-is; do not descend
					if ignoredBy(cur, relSlash, false) || excludedBy(excludes, relSlash) {
						continue
					}
					out = append(out, full)
					continue
				}
			}

			if ignoredBy(cur, relSlash, isDir) {
				if isDir {
					continue // prune
				}
				continue
			}
			if excludedBy(excludes, relSlash) {
				if isDir {
					// exclusion of a dir prunes it only if pattern matches the dir itself;
					// keep descending otherwise (file-level excludes still apply below).
					// If pattern has no wildcard and equals prefix, prune.
					prune := false
					for _, p := range excludes {
						pp := strings.TrimSuffix(p, "/")
						if !strings.ContainsAny(pp, "*?[") && (relSlash == pp || strings.HasPrefix(relSlash, pp+"/")) {
							prune = true
						}
					}
					if prune {
						continue
					}
				} else {
					continue
				}
			}

			out = append(out, full)
			if isDir {
				if err := walkFn(full, cur); err != nil {
					return err
				}
			}
		}
		return nil
	}

	out = append(out, absRoot)
	if err := walkFn(absRoot, nil); err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

func ignoredBy(patterns []string, rel string, isDir bool) bool {
	for _, p := range patterns {
		if matchIgnore(p, rel, isDir) {
			return true
		}
	}
	return false
}

func excludedBy(excludes []string, rel string) bool {
	for _, p := range excludes {
		// try full-rel match and basename match (gitignore-ish convenience)
		if ok, _ := doublestar.Match(p, rel); ok {
			return true
		}
		if !strings.Contains(p, "/") {
			base := rel
			if i := strings.LastIndex(rel, "/"); i >= 0 {
				base = rel[i+1:]
			}
			if ok, _ := doublestar.Match(p, base); ok {
				return true
			}
		}
		// doublestar "a/**" should also match "a" itself
		if strings.HasSuffix(p, "/**") {
			if strings.TrimSuffix(p, "/**") == rel {
				return true
			}
		}
	}
	return false
}
