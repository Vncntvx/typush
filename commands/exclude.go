package commands

import (
	"fmt"

	"github.com/Vncntvx/typkg/manifest"
	"github.com/Vncntvx/typkg/util"
)

// Add appends files to package.exclude (dedup) after validating globs.
func Add(dir string, files []string) error {
	m, err := manifest.Read(dir)
	if err != nil {
		return err
	}
	// validate by running the walker filter once
	if _, err := util.ListInstall(dir, append(append([]string{}, m.Package.Exclude...), files...)); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, e := range m.Package.Exclude {
		seen[e] = true
	}
	for _, f := range files {
		if !seen[f] {
			seen[f] = true
			m.Package.Exclude = append(m.Package.Exclude, f)
		}
	}
	if err := manifest.Write(dir, m); err != nil {
		return err
	}
	fmt.Printf("Excluded %d pattern(s)\n", len(files))
	return nil
}
