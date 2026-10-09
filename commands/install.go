package commands

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Vncntvx/typush/manifest"
	"github.com/Vncntvx/typush/util"
)

// Install installs the package at srcDir into namespace target (without @ prefix handling).
// If dryRun is true, it displays the destination and file list without writing to disk.
func Install(srcDir, target string, dryRun bool) error {
	for strings.HasPrefix(target, "@") {
		// The original Rust logic keeps trimming "@" until the namespace is bare;
		// we normalize the same way and say so when previewing.
		trimmed := target[1:]
		if dryRun {
			infof("Note: normalized namespace %q to %q", target, trimmed)
		} else if !util.Confirm(fmt.Sprintf("Namespace parameter should not contain `@` prefix. Do you mean `%s`?", trimmed), true) {
			return fmt.Errorf("aborted")
		}
		target = trimmed
	}
	m, err := manifest.Read(srcDir)
	if err != nil {
		return err
	}
	if target == "preview" {
		warnf("installing directly to `preview` is discouraged, since it might break the versioning.")
		if !dryRun && !util.Confirm("Are you sure you want to install directly to `preview`?", false) {
			return fmt.Errorf("aborted")
		}
	}
	base, err := util.TypstLocalDir()
	if err != nil {
		return err
	}
	versionDir := filepath.Join(base, target, m.Package.Name, m.Package.Version)
	// One ReadDir tells us both whether the target exists and whether it holds
	// anything that would be overwritten.
	if existing, err := os.ReadDir(versionDir); err == nil {
		if len(existing) > 0 {
			if dryRun {
				infof("Note: `@%s/%s:%s` already exists and would be overwritten", target, m.Package.Name, m.Package.Version)
			} else {
				if !util.Confirm(fmt.Sprintf("`@%s/%s:%s` already exists. Overwrite?", target, m.Package.Name, m.Package.Version), false) {
					return fmt.Errorf("aborted")
				}
				if err := os.RemoveAll(versionDir); err != nil {
					return err
				}
			}
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	absSrc, err := filepath.Abs(srcDir)
	if err != nil {
		return err
	}
	entries, err := util.ListInstall(absSrc, m.Package.Exclude)
	if err != nil {
		return err
	}
	// Relative paths are derived once and shared by the preview and the copy.
	relEntries, err := util.RelEntries(entries, absSrc)
	if err != nil {
		return err
	}

	if dryRun {
		relPaths := make([]string, 0, len(relEntries))
		for _, e := range relEntries {
			relPaths = append(relPaths, e.Rel)
		}
		infof("Destination directory:\n  %s", versionDir)
		previewItems("entries to install", relPaths)
		previewNote("installation skipped, no files written")
		return nil
	}

	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		return err
	}

	for _, e := range relEntries {
		abs := e.Abs
		dest := filepath.Join(versionDir, e.Rel)
		fi, err := os.Lstat(abs)
		if err != nil {
			return err
		}
		if fi.IsDir() {
			if err := os.MkdirAll(dest, 0o755); err != nil {
				return err
			}
			continue
		}
		if fi.Mode()&fs.ModeSymlink != 0 {
			// resolve and copy file content; keep it simple and portable
			target2, err := filepath.EvalSymlinks(abs)
			if err != nil {
				return err
			}
			abs = target2
			// A resolved symlink may point to a directory.
			st, err := os.Stat(abs)
			if err != nil {
				return err
			}
			if st.IsDir() {
				if err := os.MkdirAll(dest, 0o755); err != nil {
					return err
				}
				continue
			}
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := copyFile(abs, dest); err != nil {
			return err
		}
	}
	fmt.Fprintf(os.Stderr, "Installed `@%s/%s:%s`\n", target, m.Package.Name, m.Package.Version)
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if fi, err := os.Stat(src); err == nil {
		_ = os.Chmod(dst, fi.Mode().Perm())
	}
	return nil
}
