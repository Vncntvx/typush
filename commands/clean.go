package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Vncntvx/typkg/util"
)

// CleanOne removes dev symlinks for one package under preview/.
func CleanOne(name string) error {
	base, err := util.TypstLocalDir()
	if err != nil {
		return err
	}
	pkgDir := filepath.Join(base, "preview", name)
	st, err := os.Lstat(pkgDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "WARN: package `%s` not found in local data dir, skipping\n", name)
			return nil
		}
		return err
	}
	if !st.IsDir() {
		return fmt.Errorf("package `%s` is not a directory", name)
	}
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		full := filepath.Join(pkgDir, e.Name())
		fi, err := os.Lstat(full)
		if err != nil {
			return err
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			continue
		}
		target, err := filepath.EvalSymlinks(full)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARN: symlink `%s` is broken, removing\n", full)
			_ = os.Remove(full)
			continue
		}
		if st2, err := os.Stat(target); err == nil && st2.IsDir() {
			if err := os.Remove(full); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Removed symlink of version `%s`\n", e.Name())
		} else {
			fmt.Fprintf(os.Stderr, "WARN: symlink `%s` is not a directory, skipping\n", full)
		}
	}
	return nil
}

// CleanAll removes dev symlinks for all packages.
func CleanAll() error {
	base, err := util.TypstLocalDir()
	if err != nil {
		return err
	}
	preview := filepath.Join(base, "preview")
	if st, err := os.Stat(preview); err != nil || !st.IsDir() {
		return fmt.Errorf("no packages found")
	}
	entries, err := os.ReadDir(preview)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			if err := CleanOne(e.Name()); err != nil {
				return err
			}
		}
	}
	return nil
}
