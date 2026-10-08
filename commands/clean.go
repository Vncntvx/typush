package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Vncntvx/typush/util"
)

// CleanOne removes dev symlinks for one package under preview/.
// If dryRun is true, it previews the symlinks that would be removed without deleting them.
func CleanOne(name string, dryRun bool) error {
	base, err := util.TypstLocalDir()
	if err != nil {
		return err
	}
	return cleanOneAt(base, name, dryRun)
}

// cleanOneAt cleans one package inside an already resolved data dir.
func cleanOneAt(base, name string, dryRun bool) error {
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
	count, err := cleanSymlinks(pkgDir, name, dryRun)
	if err != nil {
		return err
	}
	if dryRun && count > 0 {
		previewNote("%d symlink(s) would be removed for package `%s`", count, name)
	}
	return nil
}

// CleanAll removes dev symlinks for all packages.
// If dryRun is true, it previews the symlinks that would be removed without deleting them.
func CleanAll(dryRun bool) error {
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
	var total, packages int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		count, err := cleanSymlinks(filepath.Join(preview, e.Name()), e.Name(), dryRun)
		if err != nil {
			return err
		}
		if count > 0 {
			packages++
			total += count
		}
	}
	if dryRun {
		if total == 0 {
			previewNote("no dev symlinks found to remove")
		} else {
			// One aggregate summary, instead of a trailer per package.
			previewNote("%d symlink(s) would be removed across %d package(s)", total, packages)
		}
	}
	return nil
}

// cleanSymlinks removes (or previews) the dev symlinks of a single package
// directory and reports how many it acted on.
func cleanSymlinks(pkgDir, name string, dryRun bool) (int, error) {
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return 0, err
	}
	var wouldRemove []string
	count := 0
	for _, e := range entries {
		full := filepath.Join(pkgDir, e.Name())
		fi, err := os.Lstat(full)
		if err != nil {
			return count, err
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			continue
		}
		target, err := filepath.EvalSymlinks(full)
		if err != nil {
			// A broken symlink is still a dev link, so it still goes away.
			if dryRun {
				wouldRemove = append(wouldRemove, fmt.Sprintf("@preview/%s/%s -> (broken)", name, e.Name()))
				count++
				continue
			}
			fmt.Fprintf(os.Stderr, "WARN: symlink `%s` is broken, removing\n", full)
			_ = os.Remove(full)
			count++
			continue
		}
		if st2, err := os.Stat(target); err == nil && st2.IsDir() {
			if dryRun {
				wouldRemove = append(wouldRemove, fmt.Sprintf("@preview/%s/%s -> %s", name, e.Name(), target))
				count++
				continue
			}
			if err := os.Remove(full); err != nil {
				return count, err
			}
			fmt.Fprintf(os.Stderr, "Removed symlink of version `%s`\n", e.Name())
			count++
		} else {
			fmt.Fprintf(os.Stderr, "WARN: symlink `%s` is not a directory, skipping\n", full)
		}
	}
	if dryRun && count > 0 {
		previewItems("symlinks to remove", wouldRemove)
	}
	return count, nil
}
