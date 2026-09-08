package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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
type Manifest struct {
	Package  PackageInfo    `toml:"package"`
	Template *TemplateInfo  `toml:"template,omitempty"`
	Tool     map[string]any `toml:"tool,omitempty"`
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
	nameRe    = regexp.MustCompile(`^[a-zA-Z_-][a-zA-Z0-9_-]*$`)
	versionRe = regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.\-]+)?(\+[0-9A-Za-z.\-]+)?$`)
)

func ValidateName(name string) error {
	if !nameRe.MatchString(name) {
		return fmt.Errorf("invalid package name %q: must match %s", name, nameRe.String())
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

// Read reads and validates typst.toml under dir.
func Read(dir string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, "typst.toml"))
	if err != nil {
		return nil, fmt.Errorf("failed to read the package manifest file: %w", err)
	}
	var m Manifest
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse the package manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// Validate checks required fields.
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
