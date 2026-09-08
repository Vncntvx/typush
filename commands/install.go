package commands

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Vncntvx/typush/manifest"
	"github.com/Vncntvx/typush/util"
)

// Run installs the package at srcDir into namespace target (without @ prefix handling).
func Install(srcDir, target string) error {
	for strings.HasPrefix(target, "@") {
		trimmed := target[1:]
		if !util.Confirm(fmt.Sprintf("Namespace parameter should not contain `@` prefix. Do you mean `%s`?", trimmed), true) {
			return fmt.Errorf("aborted")
		}
		_ = trimmed
		// Original Rust logic is quirky here; we normalize to trimmed and continue.
		target = trimmed
	}
	m, err := manifest.Read(srcDir)
	if err != nil {
		return err
	}
	if target == "preview" {
		fmt.Fprintln(os.Stderr, "WARN: installing directly to `preview` is discouraged, since it might break the versioning.")
		if !util.Confirm("Are you sure you want to install directly to `preview`?", false) {
			return fmt.Errorf("aborted")
		}
	}
	base, err := util.TypstLocalDir()
	if err != nil {
		return err
	}
	versionDir := filepath.Join(base, target, m.Package.Name, m.Package.Version)
	if _, err := os.Stat(versionDir); err == nil {
		// non-empty?
		empty := true
		entries, err := os.ReadDir(versionDir)
		if err != nil {
			return err
		}
		if len(entries) > 0 {
			empty = false
		}
		if !empty {
			if !util.Confirm(fmt.Sprintf("`@%s/%s:%s` already exists. Overwrite?", target, m.Package.Name, m.Package.Version), false) {
				return fmt.Errorf("aborted")
			}
			if err := os.RemoveAll(versionDir); err != nil {
				return err
			}
		}
	}
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
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
	for _, abs := range entries {
		if abs == absSrc {
			continue
		}
		rel, err := filepath.Rel(absSrc, abs)
		if err != nil {
			return err
		}
		dest := filepath.Join(versionDir, rel)
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
		}
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
