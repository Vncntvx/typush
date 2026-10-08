package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Vncntvx/typush/manifest"
	"github.com/Vncntvx/typush/util"
)

// Bump updates package version in typst.toml and synchronizes self-references in README.md.
func Bump(dir, target string) error {
	m, err := manifest.Read(dir)
	if err != nil {
		return err
	}
	curVer := m.Package.Version
	fmt.Fprintf(os.Stderr, "Current version: %s\n", curVer)

	if target == "" {
		nextPatch, err := manifest.BumpVersion(curVer, "patch")
		if err != nil {
			return err
		}
		prompt := "Enter new version (or patch/minor/major)"
		v, err := util.PromptLine(prompt, nextPatch, false)
		if err != nil {
			return err
		}
		target = strings.TrimSpace(v)
	}

	nextVer, err := manifest.BumpVersion(curVer, target)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Bumping to %s...\n", nextVer)

	// Update typst.toml
	m.Package.Version = nextVer
	if err := manifest.Write(dir, m); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "✓ Updated typst.toml")

	// Update self-references in README.md
	readmePath := filepath.Join(dir, "README.md")
	if data, err := os.ReadFile(readmePath); err == nil {
		oldRef := fmt.Sprintf("@preview/%s:%s", m.Package.Name, curVer)
		newRef := fmt.Sprintf("@preview/%s:%s", m.Package.Name, nextVer)
		content := string(data)
		if strings.Contains(content, oldRef) {
			count := strings.Count(content, oldRef)
			updated := strings.ReplaceAll(content, oldRef, newRef)
			if err := os.WriteFile(readmePath, []byte(updated), 0o644); err == nil {
				fmt.Fprintf(os.Stderr, "✓ Updated %d version reference(s) in README.md\n", count)
			}
		}
	}

	return nil
}
