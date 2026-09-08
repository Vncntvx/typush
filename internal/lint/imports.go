package lint

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Vncntvx/typkg/internal/manifest"
)

var (
	importRe  = regexp.MustCompile(`#import\s+"([^"]+)"`)
	includeRe = regexp.MustCompile(`#include\s+"([^"]+)"`)
	// @preview/name:1.2.3 or @preview/name:1.2.3-beta (also other namespaces).
	pkgSpecRe = regexp.MustCompile(`@([A-Za-z0-9_-]+)/([A-Za-z0-9_-]+):([0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.\-]+)?(?:\+[0-9A-Za-z.\-]+)?)`)
)

// Imports checks .typ sources:
//   - relative imports that resolve to the package entrypoint -> warning
//     (should use the package specification instead)
//   - imports of the package itself with an older version -> error
//   - imports of the package itself with a newer version -> warning
func Imports(dir string, m *manifest.Manifest, sources map[string]string) []Diag {
	var out []Diag
	epAbs, _ := filepath.Abs(filepath.Join(dir, filepath.FromSlash(m.Package.Entrypoint)))
	for rel, src := range sources {
		for _, im := range findImports(src) {
			if spec := pkgSpecRe.FindStringSubmatch(im); spec != nil {
				name, ver := spec[2], spec[3]
				if name != m.Package.Name {
					continue
				}
				switch manifest.CompareVersions(ver, m.Package.Version) {
				case -1:
					out = append(out, errf("import/outdated",
						fmt.Sprintf("%s: import uses older version %s of this package (current is %s).", rel, ver, m.Package.Version)))
				case 1:
					out = append(out, warnf("import/outdated",
						fmt.Sprintf("%s: import uses newer version %s of this package (current is %s).", rel, ver, m.Package.Version)))
				}
				continue
			}
			// Relative path: does it resolve to the entrypoint?
			if strings.HasPrefix(im, "@") || strings.HasPrefix(im, ":") {
				continue
			}
			abs, _ := filepath.Abs(filepath.Join(dir, filepath.Dir(rel), filepath.FromSlash(im)))
			if abs == epAbs {
				out = append(out, warnf("import/relative",
					fmt.Sprintf("%s: this import should use the package specification, not a relative path.", rel)))
			}
		}
	}
	return out
}

func findImports(src string) []string {
	var out []string
	for _, re := range []*regexp.Regexp{importRe, includeRe} {
		for _, m := range re.FindAllStringSubmatch(src, -1) {
			out = append(out, m[1])
		}
	}
	return out
}

// LoadTypSources reads all .typ files under dir (slash-relative keys).
// Files that cannot be read are skipped.
func LoadTypSources(dir string) map[string]string {
	out := map[string]string{}
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d.IsDir() && d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".typ") {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return nil
		}
		out[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	return out
}
