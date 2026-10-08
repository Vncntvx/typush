package commands

import (
	"fmt"

	"github.com/Vncntvx/typush/manifest"
	"github.com/Vncntvx/typush/util"
)

// Add appends files to package.exclude (dedup) after validating globs.
// If dryRun is true, it previews the added patterns without writing typst.toml.
func Add(dir string, files []string, dryRun bool) error {
	m, err := manifest.Read(dir)
	if err != nil {
		return err
	}
	// Validate the resulting pattern set without walking the tree: a malformed
	// glob must fail identically whether or not we preview.
	if err := util.ValidateExcludePatterns(append(append([]string{}, m.Package.Exclude...), files...)); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, e := range m.Package.Exclude {
		seen[e] = true
	}
	var newlyAdded []string
	for _, f := range files {
		if seen[f] {
			continue
		}
		seen[f] = true
		m.Package.Exclude = append(m.Package.Exclude, f)
		newlyAdded = append(newlyAdded, f)
	}
	if dryRun {
		if len(newlyAdded) == 0 {
			previewNote("no new exclude patterns")
			return nil
		}
		previewItems("patterns to exclude", newlyAdded)
		previewNote("typst.toml unchanged")
		return nil
	}
	if len(newlyAdded) == 0 {
		// Nothing to do: leave typst.toml byte-identical instead of re-encoding it.
		fmt.Printf("Excluded 0 pattern(s)\n")
		return nil
	}
	if err := manifest.Write(dir, m); err != nil {
		return err
	}
	fmt.Printf("Excluded %d pattern(s)\n", len(newlyAdded))
	return nil
}
