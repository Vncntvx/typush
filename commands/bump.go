package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Vncntvx/typush/manifest"
	"github.com/Vncntvx/typush/util"
)

// BumpOptions configures Bump.
type BumpOptions struct {
	Dir     string
	Target  string   // patch, minor, major, or explicit version
	Include []string // additional files to update version in
	Tag     string   // HTML/XML tag enclosing the version to replace
	DryRun  bool
}

// Bump updates package version in typst.toml and optionally in additional files.
func Bump(opt BumpOptions) error {
	dir := opt.Dir
	if dir == "" {
		dir = "."
	}

	m, err := manifest.Read(dir)
	if err != nil {
		return err
	}
	curVer := m.Package.Version
	infof("Current version: %s", curVer)

	// The next patch version doubles as the prompt default and as the target a
	// preview assumes when no argument was given.
	nextPatch, err := manifest.BumpVersion(curVer, "patch")
	if err != nil {
		return err
	}

	target := opt.Target
	if target == "" {
		if opt.DryRun {
			target = nextPatch
		} else {
			prompt := "Enter new version (or patch/minor/major)"
			v, err := util.PromptLine(prompt, nextPatch, false)
			if err != nil {
				return err
			}
			target = strings.TrimSpace(v)
		}
	}

	nextVer, err := manifest.BumpVersion(curVer, target)
	if err != nil {
		return err
	}

	// Prepare replacements for included files
	var re *regexp.Regexp
	var repl string

	if opt.Tag != "" {
		// Matches <tag>curVer</tag> or <!-- tag -->curVer<!-- /tag -->
		pattern := fmt.Sprintf(`(<%s>|<!--\s*%s\s*-->)%s(</%s>|<!--\s*/%s\s*-->)`,
			regexp.QuoteMeta(opt.Tag), regexp.QuoteMeta(opt.Tag),
			regexp.QuoteMeta(curVer),
			regexp.QuoteMeta(opt.Tag), regexp.QuoteMeta(opt.Tag))
		re = regexp.MustCompile(pattern)
		repl = "${1}" + nextVer + "${2}"
	} else {
		// Matches @preview/pkg-name:curVer
		pattern := fmt.Sprintf(`(@preview/%s:)%s`, regexp.QuoteMeta(m.Package.Name), regexp.QuoteMeta(curVer))
		re = regexp.MustCompile(pattern)
		repl = "${1}" + nextVer
	}

	var allEdits []VersionEdit
	type fileChanges struct {
		path            string
		lines           []string
		trailingNewline bool
		count           int
	}
	var pendingWrites []fileChanges

	for _, relPath := range opt.Include {
		absPath := relPath
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(dir, relPath)
		}

		data, err := os.ReadFile(absPath)
		if err != nil {
			return fmt.Errorf("failed to read included file %q: %w", relPath, err)
		}

		lines, trailingNewline := splitContent(data)
		fileCount := 0

		for i, line := range lines {
			newLine := re.ReplaceAllString(line, repl)
			if newLine != line {
				fileCount++
				if opt.DryRun {
					allEdits = append(allEdits, VersionEdit{
						FilePath:   absPath,
						LineNumber: i + 1,
						OldLine:    line,
						NewLine:    newLine,
					})
				}
				lines[i] = newLine
			}
		}

		if fileCount > 0 {
			pendingWrites = append(pendingWrites, fileChanges{
				path:            absPath,
				lines:           lines,
				trailingNewline: trailingNewline,
				count:           fileCount,
			})
		}
	}

	if opt.DryRun {
		infof("Would bump version from %s to %s in typst.toml", curVer, nextVer)
		if len(allEdits) > 0 {
			var items []string
			for _, e := range allEdits {
				items = append(items, fmt.Sprintf("%s:%d:\n- %s\n+ %s",
					displayPath(e.FilePath), e.LineNumber, e.OldLine, e.NewLine))
			}
			previewItems("version updates in included files", items)
			previewNote("%d occurrence(s) in %d included file(s) would be updated", len(allEdits), len(pendingWrites))
		} else {
			previewNote("would bump version from %s to %s in typst.toml", curVer, nextVer)
		}
		return nil
	}

	infof("Bumping to %s...", nextVer)

	// Update typst.toml
	m.Package.Version = nextVer
	if err := manifest.Write(dir, m); err != nil {
		return err
	}
	infof("✓ Updated typst.toml")

	// Update included files
	for _, pw := range pendingWrites {
		content := joinContent(pw.lines, pw.trailingNewline)
		if err := util.WriteFileAtomic(pw.path, content); err != nil {
			return fmt.Errorf("failed to update %s: %w", pw.path, err)
		}
		infof("✓ Updated %s (%d occurrence(s))", displayPath(pw.path), pw.count)
	}

	return nil
}
