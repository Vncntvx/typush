package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Vncntvx/typush/util"
)

// CloneOptions configures the clone command.
type CloneOptions struct {
	Spec   string
	Dest   string
	Force  bool
	DryRun bool
}

// Clone downloads a package from Typst Universe CDN and extracts it locally.
// Spec accepts: "@preview/name:version", "name:version", or "name" (latest).
// Dest defaults to "./<name>" if empty.
func Clone(opt CloneOptions) error {
	name, version, err := parsePackageSpec(opt.Spec)
	if err != nil {
		return err
	}

	if version == "" {
		idx, err := loadUniverseIndex(false)
		if err != nil {
			return err
		}
		pkg, found := idx.Latest(name)
		if !found {
			return fmt.Errorf("package %q not found in Typst Universe", name)
		}
		version = pkg.Version
	}

	dest := opt.Dest
	if dest == "" {
		dest = name
	}

	absDest, err := filepath.Abs(dest)
	if err != nil {
		return err
	}

	if opt.DryRun {
		infof("Would clone @preview/%s:%s to %s", name, version, displayPath(absDest))
		previewNote("clone skipped, no files downloaded")
		return nil
	}

	// Check if destination exists and is not empty
	if entries, err := os.ReadDir(absDest); err == nil && len(entries) > 0 {
		if !opt.Force {
			prompt := fmt.Sprintf("Directory %q already exists and is not empty. Overwrite?", dest)
			if !util.Confirm(prompt, false) {
				return fmt.Errorf("aborted")
			}
		}
	}

	if err := os.MkdirAll(absDest, 0o755); err != nil {
		return err
	}

	infof("Downloading @preview/%s:%s...", name, version)
	body, err := util.FetchPackageArchive(name, version)
	if err != nil {
		return err
	}
	defer body.Close()

	if err := util.ExtractTarGz(body, absDest); err != nil {
		return fmt.Errorf("failed to extract package archive: %w", err)
	}

	infof("Cloned @preview/%s:%s to %s", name, version, displayPath(absDest))
	infof("Usage: #import \"@preview/%s:%s\": *", name, version)
	return nil
}
