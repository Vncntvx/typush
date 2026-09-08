package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Vncntvx/typush/manifest"
	"github.com/Vncntvx/typush/util"
)

// Dev creates a preview symlink for local development.
// When checkUniverse is true it warns if the name/version already exists upstream.
func Dev(packageDir string, checkUniverse bool) error {
	m, err := manifest.Read(packageDir)
	if err != nil {
		return err
	}
	absPkg, err := filepath.Abs(packageDir)
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "Cleaning up the existing symlinks...")
	if err := CleanOne(m.Package.Name); err != nil {
		return err
	}
	if checkUniverse {
		if err := WarnIfExists(m.Package.Name, m.Package.Version); err != nil {
			// network failure shouldn't block local dev; warn only
			fmt.Fprintf(os.Stderr, "WARN: universe check failed: %v\n", err)
		}
	}
	base, err := util.TypstLocalDir()
	if err != nil {
		return err
	}
	versionDir := filepath.Join(base, "preview", m.Package.Name, m.Package.Version)
	if fi, err := os.Lstat(versionDir); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("version `%s` is already a symlink", m.Package.Version)
		}
		return fmt.Errorf("version `%s` already exists", m.Package.Version)
	}
	if err := os.MkdirAll(filepath.Dir(versionDir), 0o755); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Trying to create a symlink for `%s:%s`\n", m.Package.Name, m.Package.Version)
	if err := os.Symlink(absPkg, versionDir); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}
	if fi, err := os.Lstat(versionDir); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("failed to create symlink")
	}
	fmt.Fprintln(os.Stderr, "Symlink created successfully")
	return nil
}
