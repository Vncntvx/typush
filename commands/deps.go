package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Vncntvx/typush/manifest"
	"github.com/Vncntvx/typush/util"
)

// previewSpecRe matches a quoted @preview spec, capturing the package name and
// the referenced version. Scanning and rewriting share this pattern, so they
// cannot disagree about which specs count as dependencies.
var previewSpecRe = regexp.MustCompile(`["']@preview/([A-Za-z0-9_-]+):([0-9A-Za-z.\-+]+)["']`)

// DependencyStatus describes how a dependency compares to Typst Universe.
type DependencyStatus int

const (
	// StatusUnknown is the zero value, for a comparison that never ran. Keeping it
	// out of StatusUpToDate means an unset status cannot pass as one.
	StatusUnknown DependencyStatus = iota
	StatusUpToDate
	StatusOutdated
	StatusNotInUniverse
)

const (
	statusUnknownJSON       = "unknown"
	statusUpToDateJSON      = "up-to-date"
	statusOutdatedJSON      = "outdated"
	statusNotInUniverseJSON = "not-in-universe"
)

func (s DependencyStatus) String() string {
	switch s {
	case StatusUpToDate:
		return statusUpToDateJSON
	case StatusOutdated:
		return statusOutdatedJSON
	case StatusNotInUniverse:
		return statusNotInUniverseJSON
	default:
		return statusUnknownJSON
	}
}

// MarshalJSON renders the status string for JSON serialization.
func (s DependencyStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON parses the status string MarshalJSON writes, so the JSON output
// can be read back.
func (s *DependencyStatus) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return err
	}
	switch name {
	case statusUnknownJSON:
		*s = StatusUnknown
	case statusUpToDateJSON:
		*s = StatusUpToDate
	case statusOutdatedJSON:
		*s = StatusOutdated
	case statusNotInUniverseJSON:
		*s = StatusNotInUniverse
	default:
		return fmt.Errorf("unknown dependency status %q", name)
	}
	return nil
}

// ImportOccurrence records one @preview import. FilePath is an absolute path.
type ImportOccurrence struct {
	FilePath   string `json:"filePath"`
	LineNumber int    `json:"lineNumber"`
	Package    string `json:"package"`
	Version    string `json:"version"`
}

// PackageDependency aggregates every import of one package across the scanned
// files.
type PackageDependency struct {
	Name            string             `json:"name"`
	CurrentVersions []string           `json:"currentVersions"`
	LatestVersion   string             `json:"latestVersion"`
	Status          DependencyStatus   `json:"status"`
	Occurrences     []ImportOccurrence `json:"occurrences"`
}

// VersionEdit represents a planned line replacement.
type VersionEdit struct {
	FilePath   string `json:"filePath"`
	LineNumber int    `json:"lineNumber"`
	OldLine    string `json:"oldLine"`
	NewLine    string `json:"newLine"`
}

// plannedUpdate is one package rewrite chosen for this run: the version to
// write and the files that reference the package.
type plannedUpdate struct {
	Name    string
	Version string
	Files   []string
}

// scanDependencies finds every @preview import under target (a .typ file or directory).
func scanDependencies(target string) ([]ImportOccurrence, error) {
	fi, err := os.Stat(target)
	if err != nil {
		return nil, err
	}

	var files []string
	if fi.IsDir() {
		if files, err = util.ListTypSources(target); err != nil {
			return nil, err
		}
	} else {
		// Match util.ListTypSources: only lowercase ".typ" is a Typst source.
		if !strings.HasSuffix(target, ".typ") {
			return nil, fmt.Errorf("target %q is not a .typ file", target)
		}
		abs, err := filepath.Abs(target)
		if err != nil {
			return nil, err
		}
		files = []string{abs}
	}

	var occurrences []ImportOccurrence
	for _, file := range files {
		occs, err := scanTypFile(file)
		if err != nil {
			return nil, err
		}
		occurrences = append(occurrences, occs...)
	}
	return occurrences, nil
}

// scanTypFile reads a .typ file and returns its @preview imports.
func scanTypFile(path string) ([]ImportOccurrence, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	lines, _ := splitContent(data)
	var occurrences []ImportOccurrence
	for i, line := range lines {
		for _, m := range previewSpecRe.FindAllStringSubmatch(line, -1) {
			occurrences = append(occurrences, ImportOccurrence{
				FilePath:   abs,
				LineNumber: i + 1,
				Package:    m[1],
				Version:    m[2],
			})
		}
	}
	return occurrences, nil
}

// groupDependencies aggregates occurrences by package and compares each against
// the Universe index.
func groupDependencies(occurrences []ImportOccurrence, idx *util.UniverseIndex) []PackageDependency {
	byName := make(map[string][]ImportOccurrence)
	for _, occ := range occurrences {
		byName[occ.Package] = append(byName[occ.Package], occ)
	}

	deps := make([]PackageDependency, 0, len(byName))
	for name, occs := range byName {
		versions := uniqueVersions(occs)
		latest, found := idx.Latest(name)

		dep := PackageDependency{
			Name:            name,
			CurrentVersions: versions,
			Occurrences:     occs,
		}
		switch {
		case !found:
			dep.Status = StatusNotInUniverse
		case isBehind(versions, latest.Version):
			dep.Status = StatusOutdated
			dep.LatestVersion = latest.Version
		default:
			dep.Status = StatusUpToDate
			dep.LatestVersion = latest.Version
		}
		deps = append(deps, dep)
	}

	sort.Slice(deps, func(i, j int) bool { return deps[i].Name < deps[j].Name })
	return deps
}

// uniqueVersions returns the distinct referenced versions, newest first.
func uniqueVersions(occs []ImportOccurrence) []string {
	seen := make(map[string]struct{}, len(occs))
	versions := make([]string, 0, len(occs))
	for _, occ := range occs {
		if _, ok := seen[occ.Version]; ok {
			continue
		}
		seen[occ.Version] = struct{}{}
		versions = append(versions, occ.Version)
	}
	sort.Slice(versions, func(i, j int) bool {
		return manifest.CompareVersions(versions[i], versions[j]) > 0
	})
	return versions
}

// isBehind reports whether any referenced version is older than latest.
func isBehind(versions []string, latest string) bool {
	for _, v := range versions {
		if manifest.CompareVersions(v, latest) < 0 {
			return true
		}
	}
	return false
}

// rewritePlan describes what a rewrite pass did, or would do, to a set of files.
type rewritePlan struct {
	// Edits lists the replaced lines, ordered by file and then by line.
	Edits []VersionEdit
	// Hits counts replaced version strings per package name.
	Hits map[string]int
	// Files lists the files whose content changed, in the order they were scanned.
	Files []string
}

// Occurrences returns how many version strings the pass replaced.
func (p rewritePlan) Occurrences() int {
	total := 0
	for _, n := range p.Hits {
		total += n
	}
	return total
}

// rewriteSpecs replaces the version of every spec that is older than the version
// the plan wants for its package. Specs already at that version, and specs that
// are newer, are left as they are: a file edited after the index was read keeps
// its version rather than being downgraded. hits, when non-nil, counts the
// replacements per package name.
func rewriteSpecs(line string, targets map[string]string, hits map[string]int) (string, int) {
	// A line without the namespace cannot hold a spec.
	if !strings.Contains(line, "@preview") {
		return line, 0
	}
	matches := previewSpecRe.FindAllStringSubmatchIndex(line, -1)
	if matches == nil {
		return line, 0
	}

	var buf bytes.Buffer
	count := 0
	last := 0
	for _, m := range matches {
		name := line[m[2]:m[3]]
		target, ok := targets[name]
		if !ok {
			continue
		}
		start, end := m[4], m[5] // group 2: the referenced version
		if manifest.CompareVersions(line[start:end], target) >= 0 {
			continue
		}
		buf.WriteString(line[last:start])
		buf.WriteString(target)
		last = end
		count++
		if hits != nil {
			hits[name]++
		}
	}
	if count == 0 {
		return line, 0
	}
	buf.WriteString(line[last:])
	return buf.String(), count
}

// rewriteFiles rewrites every spec the plan covers, reading each file once and
// writing only the files whose content changed. write persists the new content; a
// nil write makes the pass read-only, which is how --dry-run previews a run.
func rewriteFiles(files []string, targets map[string]string, write func(string, []byte) error) (rewritePlan, error) {
	plan := rewritePlan{Hits: make(map[string]int)}
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			return plan, err
		}

		lines, trailingNewline := splitContent(data)
		var edits []VersionEdit
		for i, line := range lines {
			newLine, count := rewriteSpecs(line, targets, plan.Hits)
			if count == 0 {
				continue
			}
			lines[i] = newLine
			edits = append(edits, VersionEdit{
				FilePath:   path,
				LineNumber: i + 1,
				OldLine:    line,
				NewLine:    newLine,
			})
		}
		if len(edits) == 0 {
			continue
		}

		plan.Edits = append(plan.Edits, edits...)
		plan.Files = append(plan.Files, path)
		if write != nil {
			if err := write(path, joinContent(lines, trailingNewline)); err != nil {
				return plan, err
			}
		}
	}
	return plan, nil
}

// planRewrites reports the rewrites the plan would apply, without writing.
func planRewrites(files []string, targets map[string]string) (rewritePlan, error) {
	return rewriteFiles(files, targets, nil)
}

// applyRewrites applies the plan to disk, replacing each changed file atomically.
func applyRewrites(files []string, targets map[string]string) (rewritePlan, error) {
	return rewriteFiles(files, targets, util.WriteFileAtomic)
}

// splitContent splits content into lines without terminators and reports
// whether it ended with a newline.
func splitContent(data []byte) (lines []string, trailingNewline bool) {
	if len(data) == 0 {
		return nil, false
	}
	text := string(data)
	trailingNewline = strings.HasSuffix(text, "\n")
	lines = strings.Split(text, "\n")
	if trailingNewline {
		lines = lines[:len(lines)-1]
	}
	return lines, trailingNewline
}

// joinContent reassembles lines produced by splitContent.
func joinContent(lines []string, trailingNewline bool) []byte {
	text := strings.Join(lines, "\n")
	if trailingNewline {
		text += "\n"
	}
	return []byte(text)
}
