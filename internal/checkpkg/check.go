// Package checkpkg validates a package directory.
//
// Default mode mirrors the hard errors of the official typst/packages
// bundler (what CI runs on your submission PR). Use Local for the
// compiler-minimal rules (name/version/entrypoint present).
package checkpkg

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/bmatcuk/doublestar/v4"
	"golang.org/x/image/webp"

	"github.com/Vncntvx/typkg/internal/manifest"
	"github.com/Vncntvx/typkg/internal/walker"
)

// Options tunes strictness.
type Options struct {
	// Local skips Universe submission rules (license/README/template...).
	Local bool
}

// Run validates dir with default (Universe) rules.
func Run(dir string) error { return RunWith(dir, Options{}) }

// RunWith validates dir.
func RunWith(dir string, opt Options) error {
	m, err := manifest.Read(dir)
	if err != nil {
		return err
	}
	warns := 0
	warn := func(f string, a ...any) {
		warns++
		fmt.Fprintf(os.Stderr, "WARN: "+f+"\n", a...)
	}

	if opt.Local {
		if _, err := os.Stat(filepath.Join(dir, m.Package.Entrypoint)); err != nil {
			return fmt.Errorf("entrypoint %q not found: %w", m.Package.Entrypoint, err)
		}
		fmt.Fprintln(os.Stderr, "No issues found (local rules)")
		return nil
	}

	// Manifest-only Universe rules (unknown fields, authors, license, ...).
	if err := m.ValidateUniverse(); err != nil {
		return err
	}
	for _, w := range m.WarnUniverse() {
		warn("%s", w)
	}

	// Directory name should match manifest (packages/preview/<name>/<version>).
	checkDirName(dir, m, warn)

	// Entrypoint: exists, .typ, valid UTF-8.
	if err := checkTypstFile(dir, m.Package.Entrypoint, "package entrypoint"); err != nil {
		return err
	}

	// README.md is required and must not be excluded.
	readme, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		return fmt.Errorf("failed to read README.md: README is required for Universe submissions")
	}
	if excluded(m, "README.md") {
		return fmt.Errorf("README.md must not be excluded (see \"what to exclude\")")
	}

	// License: LICENSE file or link in README; must not be excluded.
	if err := checkLicenseFile(dir, string(readme)); err != nil {
		return err
	}
	if excluded(m, "LICENSE") {
		return fmt.Errorf("LICENSE must not be excluded")
	}

	// Template checks.
	if m.Template != nil {
		if err := checkTemplate(dir, m, warn); err != nil {
			return err
		}
	}

	// Bundle: entrypoint must survive excludes; size warnings.
	if err := checkBundle(dir, m, warn); err != nil {
		return err
	}

	if warns == 0 {
		fmt.Fprintln(os.Stderr, "No issues found")
	} else {
		fmt.Fprintf(os.Stderr, "No errors found (%d warning(s))\n", warns)
	}
	return nil
}

func checkDirName(dir string, m *manifest.Manifest, warn func(string, ...any)) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return
	}
	ver := filepath.Base(abs)
	name := filepath.Base(filepath.Dir(abs))
	if ver == m.Package.Version && name == m.Package.Name {
		return
	}
	// Only nag when it looks like a packages checkout (parent chain exists).
	if manifest.ValidateVersion(ver) == nil && manifest.IsIdent(name) {
		warn("package directory name %q/%q does not match manifest %s/%s (expected packages/preview/<name>/<version>)",
			name, ver, m.Package.Name, m.Package.Version)
	}
}

func checkTypstFile(dir, rel, what string) error {
	p := filepath.Join(dir, filepath.FromSlash(rel))
	st, err := os.Stat(p)
	if err != nil || st.IsDir() {
		return fmt.Errorf("%s is missing (%q)", what, rel)
	}
	if !strings.HasSuffix(strings.ToLower(rel), ".typ") {
		return fmt.Errorf("%s must have a .typ extension (%q)", what, rel)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return fmt.Errorf("failed to read %s file: %w", what, err)
	}
	if !utf8.Valid(data) {
		return fmt.Errorf("%s file is not valid UTF-8 (%q)", what, rel)
	}
	return nil
}

func checkLicenseFile(dir, readme string) error {
	if _, err := os.Stat(filepath.Join(dir, "LICENSE")); err == nil {
		return nil
	}
	// Accept a link to a license in the README.
	lower := strings.ToLower(readme)
	if strings.Contains(lower, "licen") && strings.Contains(lower, "http") {
		return nil
	}
	return fmt.Errorf("package must contain a LICENSE file or link to one in README.md")
}

func checkTemplate(dir string, m *manifest.Manifest, warn func(string, ...any)) error {
	t := m.Template
	tplDir := filepath.Join(dir, filepath.FromSlash(t.Path))
	if st, err := os.Stat(tplDir); err != nil || !st.IsDir() {
		return fmt.Errorf("template.path %q not found", t.Path)
	}
	// Entrypoint is relative to the template path (official bundler).
	entry := filepath.Join(tplDir, filepath.FromSlash(t.Entrypoint))
	rel, _ := filepath.Rel(dir, entry)
	if err := checkTypstFile(dir, filepath.ToSlash(rel), "template entrypoint"); err != nil {
		return err
	}
	if t.Thumbnail == nil {
		return fmt.Errorf("template.thumbnail is required for Universe submissions")
	}
	thumbPath := filepath.Join(dir, filepath.FromSlash(*t.Thumbnail))
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(*t.Thumbnail), "."))
	if ext != "png" && ext != "webp" {
		return fmt.Errorf("thumbnail must be a PNG or WebP image (%q)", *t.Thumbnail)
	}
	st, err := os.Stat(thumbPath)
	if err != nil {
		return fmt.Errorf("thumbnail %q not found", *t.Thumbnail)
	}
	if st.Size() > 3*1024*1024 {
		return fmt.Errorf("thumbnail must be smaller than 3 MiB (%q)", *t.Thumbnail)
	}
	longest, err := imageLongestEdge(thumbPath, ext)
	if err != nil {
		return fmt.Errorf("failed to decode thumbnail %q: %w", *t.Thumbnail, err)
	}
	if longest < 1080 {
		return fmt.Errorf("thumbnail's longest edge must be at least 1080 px long (%q is %d px)", *t.Thumbnail, longest)
	}
	// Thumbnail is auto-excluded from the bundle and must not be referenced.
	base := filepath.Base(*t.Thumbnail)
	if refsInTypFiles(dir, base) {
		warn("thumbnail %q must not be referenced anywhere in the package", base)
	}
	return nil
}

func imageLongestEdge(path, ext string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	var w, h int
	switch ext {
	case "png":
		cfg, err := png.DecodeConfig(f)
		if err != nil {
			return 0, err
		}
		w, h = cfg.Width, cfg.Height
	case "webp":
		cfg, err := webp.DecodeConfig(f)
		if err != nil {
			return 0, err
		}
		w, h = cfg.Width, cfg.Height
	}
	if w > h {
		return w, nil
	}
	return h, nil
}

// refsInTypFiles reports whether any .typ file mentions name.
func refsInTypFiles(dir, name string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || found || d.IsDir() {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") || !strings.HasSuffix(d.Name(), ".typ") {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		if strings.Contains(string(data), name) {
			found = true
		}
		return nil
	})
	return found
}

// excluded reports whether rel (slash path, e.g. README.md) is filtered out
// by package.exclude.
func excluded(m *manifest.Manifest, rel string) bool {
	for _, p := range m.Package.Exclude {
		if ok, _ := doublestar.Match(p, rel); ok {
			return true
		}
		if !strings.Contains(p, "/") {
			if ok, _ := doublestar.Match(p, filepath.Base(rel)); ok {
				return true
			}
		}
	}
	return false
}

func checkBundle(dir string, m *manifest.Manifest, warn func(string, ...any)) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	entries, err := walker.ListPublish(abs)
	if err != nil {
		return err
	}
	thumb := ""
	if m.Template != nil && m.Template.Thumbnail != nil {
		thumb = filepath.ToSlash(*m.Template.Thumbnail)
	}
	var total int64
	var files int
	included := map[string]bool{}
	for _, e := range entries {
		if e == abs {
			continue
		}
		rel, _ := filepath.Rel(abs, e)
		relSlash := filepath.ToSlash(rel)
		st, err := os.Stat(e)
		if err != nil || st.IsDir() {
			continue
		}
		if relSlash == thumb {
			continue // auto-excluded like the bundler
		}
		if excluded(m, relSlash) {
			continue
		}
		included[relSlash] = true
		files++
		total += st.Size()
		if st.Size() > 1*1024*1024 {
			warn("bundle file %q is larger than 1 MiB; consider exclude (see \"what to exclude\")", relSlash)
		}
	}
	ep := filepath.ToSlash(m.Package.Entrypoint)
	if !included[ep] {
		return fmt.Errorf("package entrypoint %q is excluded from the bundle", ep)
	}
	if !included["typst.toml"] {
		return fmt.Errorf("typst.toml is excluded from the bundle")
	}
	if files > 500 {
		warn("bundle contains %d files; keep packages small", files)
	}
	if total > 5*1024*1024 {
		warn("bundle is %.1f MiB; keep packages small and exclude docs assets", float64(total)/1048576)
	}
	return nil
}
