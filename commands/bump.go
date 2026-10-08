package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/Vncntvx/typush/manifest"
	"github.com/Vncntvx/typush/util"
)

// Bump updates the package version in typst.toml.
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

	return nil
}
