package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

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
	if err := CleanOne(m.Package.Name, Execute); err != nil {
		return err
	}
	if checkUniverse {
		if err := WarnIfExists(m.Package.Name, m.Package.Version); err != nil {
			// network failure shouldn't block local dev; warn only
			warnf("universe check failed: %v", err)
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
	fmt.Fprintf(os.Stderr, "Creating symlink for `%s:%s`...\n", m.Package.Name, m.Package.Version)
	if err := os.Symlink(absPkg, versionDir); err != nil {
		if runtime.GOOS == "windows" {
			return fmt.Errorf("failed to create symlink: %w\nTIP: Windows requires Developer Mode or Administrator privileges to create symlinks. You can also use 'typush install local' instead", err)
		}
		return fmt.Errorf("failed to create symlink: %w", err)
	}
	if fi, err := os.Lstat(versionDir); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("failed to create symlink")
	}
	fmt.Fprintln(os.Stderr, "Symlink created.")
	return nil
}

type DevLink struct {
	Package string
	Version string
	Status  string
	Target  string
}

// DevList lists all active development symlinks under @preview.
func DevList() error {
	base, err := util.TypstLocalDir()
	if err != nil {
		return err
	}
	preview := filepath.Join(base, "preview")
	st, err := os.Stat(preview)
	if err != nil || !st.IsDir() {
		fmt.Fprintln(os.Stderr, "No packages found in local data directory.")
		return nil
	}
	pkgs, err := os.ReadDir(preview)
	if err != nil {
		return err
	}
	var links []DevLink
	for _, pkg := range pkgs {
		if !pkg.IsDir() {
			continue
		}
		pkgDir := filepath.Join(preview, pkg.Name())
		versions, err := os.ReadDir(pkgDir)
		if err != nil {
			continue
		}
		for _, v := range versions {
			full := filepath.Join(pkgDir, v.Name())
			fi, err := os.Lstat(full)
			if err != nil || fi.Mode()&os.ModeSymlink == 0 {
				continue
			}
			target, err := filepath.EvalSymlinks(full)
			status := "active"
			if err != nil {
				status = "broken"
				target = "(broken target)"
			}
			links = append(links, DevLink{
				Package: pkg.Name(),
				Version: v.Name(),
				Status:  status,
				Target:  target,
			})
		}
	}

	if len(links) == 0 {
		fmt.Fprintln(os.Stderr, "No active dev symlinks found in @preview.")
		return nil
	}

	fmt.Fprintf(os.Stdout, "%-24s %-12s %-10s %s\n", "PACKAGE", "VERSION", "STATUS", "TARGET")
	for _, l := range links {
		fmt.Fprintf(os.Stdout, "%-24s %-12s %-10s %s\n", l.Package, l.Version, l.Status, l.Target)
	}
	return nil
}
