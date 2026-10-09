package commands

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Vncntvx/typush/util"
)

// UninstallOptions configures the uninstall command.
type UninstallOptions struct {
	Target string
	Force  bool
	DryRun bool
}

// Uninstall removes installed packages from the local package directory.
// Target syntax:
//   - "@ns/pkg:ver" or "ns/pkg:ver" -> remove specific version
//   - "@ns/pkg" or "ns/pkg"         -> remove all versions of pkg in ns
//   - "@ns"                         -> remove entire namespace
//   - "pkg:ver"                     -> remove version from @local
//   - "pkg"                         -> remove all versions from @local
func Uninstall(opt UninstallOptions) error {
	ns, pkg, ver, err := parseUninstallTarget(opt.Target)
	if err != nil {
		return err
	}

	root, err := util.TypstLocalDir()
	if err != nil {
		return err
	}

	var targetPath string
	var desc string

	switch {
	case ver != "":
		targetPath = filepath.Join(root, ns, pkg, ver)
		desc = fmt.Sprintf("@%s/%s:%s", ns, pkg, ver)
	case pkg != "":
		targetPath = filepath.Join(root, ns, pkg)
		desc = fmt.Sprintf("@%s/%s (all versions)", ns, pkg)
	default:
		targetPath = filepath.Join(root, ns)
		desc = fmt.Sprintf("@%s (entire namespace)", ns)
	}

	fi, err := os.Lstat(targetPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("target %s not found in local package directory", desc)
		}
		return err
	}

	if fi.Mode()&os.ModeSymlink != 0 {
		warnf("%s is a development symlink; you can also use 'typush clean' to manage dev links", desc)
	}

	if opt.DryRun {
		infof("Target to remove:\n  %s", targetPath)
		previewNote("would remove %s (no files deleted)", desc)
		return nil
	}

	if !opt.Force {
		prompt := fmt.Sprintf("Are you sure you want to remove %s?", desc)
		if !util.Confirm(prompt, false) {
			return fmt.Errorf("aborted")
		}
	}

	if err := os.RemoveAll(targetPath); err != nil {
		return fmt.Errorf("failed to remove %s: %w", targetPath, err)
	}

	// Remove parent directories if they are now empty.
	if ver != "" {
		_ = os.Remove(filepath.Join(root, ns, pkg))
	}
	_ = os.Remove(filepath.Join(root, ns))

	infof("Removed %s", desc)
	return nil
}

func parseUninstallTarget(target string) (ns, pkg, ver string, err error) {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" {
		return "", "", "", fmt.Errorf("uninstall target cannot be empty")
	}

	if strings.Contains(trimmed, "..") {
		return "", "", "", fmt.Errorf("invalid target %q: path traversal not allowed", target)
	}

	isExplicitNs := strings.HasPrefix(trimmed, "@") && !strings.Contains(trimmed, "/")
	raw := strings.TrimPrefix(trimmed, "@")
	if isExplicitNs {
		if strings.Contains(raw, ":") {
			return "", "", "", fmt.Errorf("invalid target %q: namespace cannot contain ':'", target)
		}
		return raw, "", "", nil
	}

	if strings.Contains(raw, "/") {
		ns, rest, _ := strings.Cut(raw, "/")
		if rest == "" {
			return ns, "", "", nil
		}
		pkg, ver, hasVer := strings.Cut(rest, ":")
		if hasVer {
			return ns, pkg, ver, nil
		}
		return ns, pkg, "", nil
	}

	if pkg, ver, hasVer := strings.Cut(raw, ":"); hasVer {
		return "local", pkg, ver, nil
	}
	return "local", raw, "", nil
}
