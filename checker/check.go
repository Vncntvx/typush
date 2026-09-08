// Package checkpkg validates a package directory.
//
// Default mode mirrors the hard errors of the official typst/packages
// bundler plus the network-free rules of typst/package-check. Use Local
// for the compiler-minimal rules (name/version/entrypoint present).
package checker

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/bmatcuk/doublestar/v4"
	"golang.org/x/image/webp"

	"github.com/Vncntvx/typush/manifest"
	"github.com/Vncntvx/typush/util"
)

// Options tunes strictness.
type Options struct {
	// Local skips Universe submission rules (license/README/template...).
	Local bool
	// NoCompile skips the local Typst compiler checks (import + template).
	NoCompile bool
}

// Run validates dir with default (Universe) rules.
func Run(dir string) error { return RunWith(dir, Options{}) }

// RunWith validates dir. All errors are reported before returning.
func RunWith(dir string, opt Options) error {
	m, err := manifest.Read(dir)
	if err != nil {
		return err
	}
	c := &collector{}
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
		c.warn("%s", w)
	}

	checkDirName(dir, m, c)
	c.errsFrom(checkRootFileNames(dir))

	// Entrypoint: exists, .typ, valid UTF-8.
	if err := checkTypstFile(dir, m.Package.Entrypoint, "package entrypoint"); err != nil {
		c.err("%s", err)
	}

	// README.md is required.
	readme, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		c.err("failed to read README.md: README is required for Universe submissions")
		readme = nil
	} else {
		diags, linked := Readme(dir, string(readme))
		c.lint(diags)
		c.linked = linked
	}
	if excluded(m, "README.md") {
		c.warn("README.md should not be excluded (see \"what to exclude\")")
	}

	// License: LICENSE file or link in README; must not be excluded.
	if err := checkLicenseFile(dir, string(readme)); err != nil {
		c.err("%s", err)
	}
	if excluded(m, "LICENSE") {
		c.warn("LICENSE should not be excluded")
	}

	// Template checks.
	if m.Template != nil {
		if err := checkTemplate(dir, m, c); err != nil {
			c.err("%s", err)
		}
	}

	// Bundle + files + imports lints.
	c.lint(checkBundle(dir, m, c))
	c.lint(checkFiles(dir, m, c.linked))
	c.lint(Imports(dir, m, LoadTypSources(dir)))

	// Real compiler checks with the local typst binary.
	if !opt.NoCompile {
		c.lintCompile(dir, m)
	}

	return c.result()
}

// collector accumulates errors and prints warnings immediately.
type collector struct {
	errs   []string
	warns  int
	linked []string
}

func (c *collector) err(f string, a ...any) {
	msg := fmt.Sprintf(f, a...)
	c.errs = append(c.errs, msg)
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", msg)
}

func (c *collector) errsFrom(ds []Diag) {
	for _, d := range ds {
		if d.Severity == Error {
			c.err("%s: %s", d.Code, d.Message)
		} else {
			c.warn("%s: %s", d.Code, d.Message)
		}
	}
}

// lint feeds lint diagnostics into the collector.
func (c *collector) lint(ds []Diag) {
	c.errsFrom(ds)
}

func (c *collector) warn(f string, a ...any) {
	c.warns++
	fmt.Fprintf(os.Stderr, "WARN: "+f+"\n", a...)
}

func (c *collector) result() error {
	if len(c.errs) > 0 {
		return fmt.Errorf("check failed with %d error(s)", len(c.errs))
	}
	if c.warns == 0 {
		fmt.Fprintln(os.Stderr, "No issues found")
	} else {
		fmt.Fprintf(os.Stderr, "No errors found (%d warning(s))\n", c.warns)
	}
	return nil
}

// lintCompile runs the local Typst compiler checks.
func (c *collector) lintCompile(dir string, m *manifest.Manifest) {
	bin, ok := LookPath()
	if !ok {
		c.warn("typst binary not found in PATH, compile checks skipped (TYPST_BIN can override)")
		return
	}
	fmt.Fprintln(os.Stderr, "Compiling with local typst...")
	res := CheckLibrary(bin, dir, m.Package.Name, m.Package.Version)
	if m.Template != nil {
		// Inside the initialized project, the template entrypoint is
		// relative to the template path (init copies path/* to root).
		r2 := CheckTemplate(bin, dir, m.Package.Name, m.Package.Version,
			m.Template.Entrypoint)
		res.Errors = append(res.Errors, r2.Errors...)
		res.Warnings = append(res.Warnings, r2.Warnings...)
	}
	for _, e := range res.Errors {
		c.err("compile/error: %s", e)
	}
	for _, w := range res.Warnings {
		c.warn("compile/warning: %s", w)
	}
}

// checkRootFileNames mirrors package-check: LICENCE must be LICENSE,
// license.*/readme.* stems must be ALL CAPS.
func checkRootFileNames(dir string) []Diag {
	var out []Diag
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		stem := name
		if i := strings.LastIndex(name, "."); i >= 0 {
			stem = name[:i]
		}
		upper := strings.ToUpper(stem)
		if upper == "LICENCE" {
			out = append(out, Diag{Severity: Error, Code: "filename/licence",
				Message: fmt.Sprintf("%s: this file should be named LICENSE.", name)})
			continue
		}
		if (upper == "LICENSE" || upper == "README") && stem != upper {
			fixed := upper + name[len(stem):]
			out = append(out, Diag{Severity: Error, Code: "filename/case",
				Message: fmt.Sprintf("%s: please use ALL CAPS for this file (i.e. %s).", name, fixed)})
		}
	}
	return out
}

func checkDirName(dir string, m *manifest.Manifest, c *collector) {
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
		c.warn("package directory name %q/%q does not match manifest %s/%s (expected packages/preview/<name>/<version>)",
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

func checkTemplate(dir string, m *manifest.Manifest, c *collector) error {
	t := m.Template
	tplDir := filepath.Join(dir, filepath.FromSlash(t.Path))
	if st, err := os.Stat(tplDir); err != nil || !st.IsDir() {
		return fmt.Errorf("template.path %q not found", t.Path)
	}
	// Entrypoint is relative to the template path (official spec).
	entry := filepath.Join(tplDir, filepath.FromSlash(t.Entrypoint))
	rel, _ := filepath.Rel(dir, entry)
	if err := checkTypstFile(dir, filepath.ToSlash(rel), "template entrypoint"); err != nil {
		return err
	}
	// The template entrypoint should import the package by spec, not relatively.
	if data, err := os.ReadFile(entry); err == nil {
		spec := "@preview/" + m.Package.Name + ":"
		if !strings.Contains(string(data), spec) {
			c.warn("template entrypoint should import the package via `%s<version>` instead of a relative file import", spec)
		}
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
		c.warn("thumbnail %q must not be referenced anywhere in the package", base)
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
		pat := strings.TrimPrefix(p, "./")
		if ok, _ := doublestar.Match(pat, rel); ok {
			return true
		}
		if !strings.Contains(pat, "/") {
			if ok, _ := doublestar.Match(pat, filepath.Base(rel)); ok {
				return true
			}
		}
	}
	return false
}

// checkBundle computes the effective bundle (publish list minus excludes
// minus auto-excluded thumbnail) and validates membership.
func checkBundle(dir string, m *manifest.Manifest, c *collector) []Diag {
	var out []Diag
	abs, err := filepath.Abs(dir)
	if err != nil {
		return out
	}
	entries, err := util.ListPublish(abs)
	if err != nil {
		return out
	}
	thumb := ""
	if m.Template != nil && m.Template.Thumbnail != nil {
		thumb = filepath.ToSlash(*m.Template.Thumbnail)
		if excluded(m, thumb) {
			out = append(out, Diag{Severity: Error, Code: "manifest/template/thumbnail/exclude",
				Message: "The template thumbnail is automatically excluded; do not list it in `exclude`."})
		}
	}
	for _, e := range m.Package.Exclude {
		if strings.HasPrefix(e, "./") {
			out = append(out, Diag{Severity: Warning, Code: "manifest/package/exclude/leading-dot",
				Message: fmt.Sprintf("Leading `./` of exclusion %q is trimmed. Use an absolute path starting with `/` to avoid recursive matching.", e)})
		}
	}
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
	}
	ep := filepath.ToSlash(m.Package.Entrypoint)
	if !included[ep] {
		out = append(out, Diag{Severity: Error, Code: "bundle/entrypoint",
			Message: fmt.Sprintf("package entrypoint %q is excluded from the bundle", ep)})
	}
	if !included["typst.toml"] {
		out = append(out, Diag{Severity: Error, Code: "bundle/manifest",
			Message: "typst.toml is excluded from the bundle"})
	}
	return out
}

// checkFiles runs the files lint over the on-disk tree.
func checkFiles(dir string, m *manifest.Manifest, linked []string) []Diag {
	all, err := WalkAll(dir)
	if err != nil {
		return nil
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil
	}
	entries, err := util.ListPublish(abs)
	if err != nil {
		return nil
	}
	thumb := ""
	if m.Template != nil && m.Template.Thumbnail != nil {
		thumb = filepath.ToSlash(*m.Template.Thumbnail)
	}
	bundled := map[string]bool{}
	excl := map[string]bool{}
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
			continue
		}
		if excluded(m, relSlash) {
			excl[relSlash] = true
			continue
		}
		bundled[relSlash] = true
	}
	// Excluded files still need size entries for the
	for _, f := range all {
		if excluded(m, f.RelSlash) {
			excl[f.RelSlash] = true
		}
	}
	// The thumbnail is auto-excluded by the bundler: never flag it.
	if thumb != "" {
		excl[thumb] = true
	}
	return Files(all, bundled, excl, linked)
}
