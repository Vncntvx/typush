package checker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileEntry describes one on-disk file for Files().
type FileEntry struct {
	RelSlash string // slash-separated path relative to package root
	Size     int64
}

var fontExts = map[string]bool{"otf": true, "ttf": true, "woff": true, "woff2": true}

var manualNames = map[string]bool{
	"manual.pdf": true, "doc.pdf": true, "docs.pdf": true, "documentation.pdf": true,
}

// commonIgnored are hidden helper files that never trigger ignored warnings.
var commonIgnored = map[string]bool{
	".gitattributes": true, ".gitignore": true, ".gitkeep": true,
	".ignore": true, ".keep": true, ".typstignore": true,
}

// Files mirrors package-check files::check.
//   - all: every file under the package dir (relative slash paths)
//   - bundled: files that survive ignore+exclude filtering (i.e. in the bundle)
//   - excluded: rel paths filtered out by package.exclude
//   - linked: local files linked from README.md
func Files(all []FileEntry, bundled map[string]bool, excluded map[string]bool, linked []string) []Diag {
	var out []Diag
	linkedSet := map[string]bool{}
	for _, l := range linked {
		linkedSet[l] = true
	}
	for _, f := range all {
		rel := f.RelSlash
		isBundled := bundled[rel]
		isExcluded := excluded[rel]

		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(rel), "."))
		if fontExts[ext] {
			out = append(out, errf("files/fonts",
				fmt.Sprintf("%s: font files are not allowed. Delete them and instruct users to install them manually.", rel)))
		}

		base := filepath.Base(rel)
		lowerBase := strings.ToLower(base)
		if !isExcluded {
			if strings.Contains(lowerBase, "example") {
				out = append(out, warnf("exclude/example",
					fmt.Sprintf("%s seems to be an example and should probably be added to `exclude`.", rel)))
			} else if strings.Contains(lowerBase, "test") {
				out = append(out, warnf("exclude/test",
					fmt.Sprintf("%s seems to be a test and should probably be added to `exclude`.", rel)))
			}
		}

		if f.Size >= 50*1024*1024 {
			out = append(out, warnf("size/extra-large",
				fmt.Sprintf("%s is really large (%dMB). If possible, do not include it in this repository at all.", rel, f.Size/1024/1024)))
		} else if f.Size >= 1024*1024 && ext != "wasm" && !isExcluded {
			out = append(out, warnf("size/large",
				fmt.Sprintf("%s is quite large (%dMB). If it is not required to use the package, add it to `exclude`.", rel, f.Size/1024/1024)))
		}

		if !isBundled && !isExcluded && !commonIgnored[base] && !strings.HasPrefix(base, ".") {
			out = append(out, warnf("files/ignored",
				fmt.Sprintf("%s won't be present in the bundled package (ignore file). If used for documentation and linked in the README, add it to `exclude`; otherwise consider removing it.", rel)))
		}

		if manualNames[lowerBase] && !linkedSet[rel] {
			msg := fmt.Sprintf("%s seems to be a manual but isn't linked in the README. It will be inaccessible on Typst Universe.", rel)
			if !isExcluded {
				msg += " It should also be added to `exclude`."
			}
			out = append(out, warnf("files/manual/unlinked", msg))
		}
	}
	return out
}

// WalkAll lists every file under dir (slash-separated, relative), including
// hidden and ignored ones. Directories named .git are pruned.
func WalkAll(dir string) ([]FileEntry, error) {
	var out []FileEntry
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return nil
		}
		var size int64
		if fi, err := d.Info(); err == nil {
			size = fi.Size()
		}
		out = append(out, FileEntry{RelSlash: filepath.ToSlash(rel), Size: size})
		return nil
	})
	return out, err
}
