package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
)

// Categories and Disciplines mirror src/model.rs in the Rust original.
var Categories = []string{
	"components", "visualization", "model", "layout", "text",
	"languages", "scripting", "integration", "utility", "fun",
	"book", "report", "paper", "thesis", "poster", "flyer",
	"presentation", "cv", "office",
}

var Disciplines = []string{
	"agriculture", "anthropology", "archaeology", "architecture",
	"biology", "business", "chemistry", "communication",
	"computer-science", "design", "drawing", "economics", "education",
	"engineering", "fashion", "film", "geography", "geology", "history",
	"journalism", "law", "linguistics", "literature", "mathematics",
	"medicine", "music", "painting", "philosophy", "photography",
	"physics", "politics", "psychology", "sociology", "theater",
	"theology", "transportation",
}

// Manifest maps typst.toml. Unknown [tool.*] tables are preserved via Tool.
// Undecoded holds unknown top-level/package/template keys (bundler rejects them).
type Manifest struct {
	Package  PackageInfo    `toml:"package"`
	Template *TemplateInfo  `toml:"template,omitempty"`
	Tool     map[string]any `toml:"tool,omitempty"`

	undecoded []string
}

type PackageInfo struct {
	Name        string   `toml:"name"`
	Version     string   `toml:"version"`
	Entrypoint  string   `toml:"entrypoint"`
	Authors     []string `toml:"authors"`
	License     *string  `toml:"license,omitempty"`
	Description *string  `toml:"description,omitempty"`
	Homepage    *string  `toml:"homepage,omitempty"`
	Repository  *string  `toml:"repository,omitempty"`
	Keywords    []string `toml:"keywords,omitempty"`
	Categories  []string `toml:"categories,omitempty"`
	Disciplines []string `toml:"disciplines,omitempty"`
	Compiler    *string  `toml:"compiler,omitempty"`
	Exclude     []string `toml:"exclude,omitempty"`
}

type TemplateInfo struct {
	Path       string  `toml:"path"`
	Entrypoint string  `toml:"entrypoint"`
	Thumbnail  *string `toml:"thumbnail,omitempty"`
}

var (
	versionRe = regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.\-]+)?(\+[0-9A-Za-z.\-]+)?$`)
)

// IsIdent reports whether s is a valid Typst identifier: first char is a
// Unicode letter or '_', rest are letters, digits, '_' or '-'.
// (Approximates unicode_ident XID rules used by the official bundler.)
func IsIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, c := range s {
		switch {
		case i == 0:
			if !(unicode.IsLetter(c) || c == '_') {
				return false
			}
		default:
			if !(unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_' || c == '-') {
				return false
			}
		}
	}
	return true
}

func ValidateName(name string) error {
	if !IsIdent(name) {
		return fmt.Errorf("package name %q is not a valid identifier", name)
	}
	return nil
}

func ValidateVersion(v string) error {
	if !versionRe.MatchString(strings.TrimSpace(v)) {
		return fmt.Errorf("invalid package version %q: expected semver x.y.z", v)
	}
	return nil
}

// CompareVersions compares two semver strings numerically on major.minor.patch,
// then lexically on pre-release. Returns -1/0/+1.
func CompareVersions(a, b string) int {
	pa := parseVer(a)
	pb := parseVer(b)
	for i := 0; i < 3; i++ {
		if pa[i] != pb[i] {
			if pa[i] < pb[i] {
				return -1
			}
			return 1
		}
	}
	preA := preRelease(a)
	preB := preRelease(b)
	switch {
	case preA == preB:
		return 0
	case preA == "":
		return 1 // release > prerelease
	case preB == "":
		return -1
	default:
		return strings.Compare(preA, preB)
	}
}

func parseVer(v string) [3]int {
	var out [3]int
	core := v
	if i := strings.Index(core, "-"); i >= 0 {
		core = core[:i]
	} else if i := strings.Index(core, "+"); i >= 0 {
		core = core[:i]
	}
	parts := strings.Split(core, ".")
	for i := 0; i < 3 && i < len(parts); i++ {
		var n int
		fmt.Sscanf(strings.TrimSpace(parts[i]), "%d", &n)
		out[i] = n
	}
	return out
}

func preRelease(v string) string {
	// strip build metadata first
	if i := strings.Index(v, "+"); i >= 0 {
		v = v[:i]
	}
	if i := strings.Index(v, "-"); i >= 0 {
		return v[i+1:]
	}
	return ""
}

func ValidateCompiler(c string) error {
	c = strings.TrimSpace(c)
	if c == "" {
		return nil
	}
	// Accept "1.2.3", "^1.2", ">=0.11", "<=x", "~x" style loosely:
	// strip leading constraint operators then check semver-ish prefix.
	t := strings.TrimLeft(c, " \t^~<>=!")
	if !regexp.MustCompile(`^\d+(\.\d+)?(\.\d+)?`).MatchString(t) {
		return fmt.Errorf("invalid compiler version bound %q", c)
	}
	return nil
}

func ValidateEntrypoint(e string) error {
	if !strings.HasSuffix(e, ".typ") {
		return fmt.Errorf("entrypoint must end with '.typ', got %q", e)
	}
	return nil
}

// Read reads and validates typst.toml under dir (compiler-minimal rules).
func Read(dir string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, "typst.toml"))
	if err != nil {
		return nil, fmt.Errorf("failed to read the package manifest file: %w", err)
	}
	var m Manifest
	md, err := toml.Decode(string(data), &m)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the package manifest: %w", err)
	}
	for _, k := range md.Undecoded() {
		m.undecoded = append(m.undecoded, k.String())
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// Undecoded returns unknown manifest keys (rejected by the official bundler).
func (m *Manifest) Undecoded() []string { return m.undecoded }

// Validate checks compiler-minimal fields (name/version/entrypoint/authors).
func (m *Manifest) Validate() error {
	if err := ValidateName(m.Package.Name); err != nil {
		return err
	}
	if err := ValidateVersion(m.Package.Version); err != nil {
		return err
	}
	if len(m.Package.Authors) == 0 {
		return fmt.Errorf("package.authors must not be empty")
	}
	if err := ValidateEntrypoint(m.Package.Entrypoint); err != nil {
		return err
	}
	if m.Package.Compiler != nil {
		if err := ValidateCompiler(*m.Package.Compiler); err != nil {
			return err
		}
	}
	if m.Template != nil {
		if strings.TrimSpace(m.Template.Path) == "" {
			return fmt.Errorf("template.path must not be empty")
		}
		if err := ValidateEntrypoint(m.Template.Entrypoint); err != nil {
			return fmt.Errorf("template.entrypoint: %w", err)
		}
	}
	return nil
}

// ValidateUniverse checks the manifest-only Universe submission rules,
// mirroring typst/packages bundler parse_manifest. Filesystem checks
// (entrypoint/README/thumbnail existence, excludes) live in checkpkg.
func (m *Manifest) ValidateUniverse() error {
	if len(m.undecoded) > 0 {
		return fmt.Errorf("unknown fields: %v", m.undecoded)
	}
	if err := m.Validate(); err != nil {
		return err
	}
	for _, a := range m.Package.Authors {
		if err := ValidateAuthor(a); err != nil {
			return fmt.Errorf("error while checking author name: %w", err)
		}
	}
	if m.Package.Description == nil {
		return fmt.Errorf("package description is missing")
	}
	if len(m.Package.Categories) > 3 {
		return fmt.Errorf("package can have at most 3 categories")
	}
	for _, c := range m.Package.Categories {
		if !IsCategory(c) {
			return fmt.Errorf("unknown category %q", c)
		}
	}
	for _, d := range m.Package.Disciplines {
		if !IsDiscipline(d) {
			return fmt.Errorf("unknown discipline %q", d)
		}
	}
	if m.Package.License == nil {
		return fmt.Errorf("package license is missing")
	}
	if err := ValidateLicense(*m.Package.License); err != nil {
		return err
	}
	if m.Template != nil && len(m.Package.Categories) == 0 {
		return fmt.Errorf("template packages must have at least one category")
	}
	for _, e := range m.Package.Exclude {
		if strings.HasPrefix(e, "!") {
			return fmt.Errorf("exclude globs with '!' are not supported: %q", e)
		}
	}
	return nil
}

// WarnUniverse returns style warnings (naming/description guidance).
// These never fail the official bundler but commonly delay acceptance.
func (m *Manifest) WarnUniverse() []string {
	var out []string
	if strings.Contains(strings.ToLower(m.Package.Name), "typst") {
		out = append(out, "package name should not include the word \"typst\" (redundant)")
	}
	if m.Package.Name != strings.ToLower(m.Package.Name) {
		out = append(out, "package name should use kebab-case")
	}
	if d := m.Package.Description; d != nil {
		n := len([]rune(*d))
		if n < 10 {
			out = append(out, "package description looks too short (aim for one sentence, 40-60 chars)")
		} else if n > 200 {
			out = append(out, "package description looks too long (aim for one sentence, 40-60 chars)")
		}
		lower := strings.ToLower(*d)
		if strings.Contains(lower, "typst package") || strings.Contains(lower, "typst template") {
			out = append(out, "avoid the redundant words \"Typst\"/\"package\"/\"template\" in the description")
		}
	}
	if m.Package.Homepage != nil && m.Package.Repository != nil &&
		*m.Package.Homepage == *m.Package.Repository {
		out = append(out, "homepage duplicates repository; omit homepage and prefer repository")
	}
	return out
}

func IsCategory(c string) bool {
	for _, k := range Categories {
		if k == c {
			return true
		}
	}
	return false
}

func IsDiscipline(d string) bool {
	for _, k := range Disciplines {
		if k == d {
			return true
		}
	}
	return false
}

// Write writes typst.toml under dir.
func Write(dir string, m *Manifest) error {
	var sb strings.Builder
	if err := toml.NewEncoder(&sb).Encode(m); err != nil {
		return fmt.Errorf("failed to encode the package manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "typst.toml"), []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("failed to write the package manifest file: %w", err)
	}
	return nil
}
