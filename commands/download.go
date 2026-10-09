package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Vncntvx/typush/util"
)

// DownloadOptions configures the download command.
type DownloadOptions struct {
	Repository string
	Checkout   string
	Namespace  string
	Subdir     string
	DryRun     bool
}

// Download clones repo (+ optional checkout ref) into a temp dir then installs.
// If subdir is non-empty, it installs only that subdirectory of the repository.
// If dryRun is true, it previews the installation without copying files to the local packages directory.
func Download(opt DownloadOptions) error {
	tmp := util.TempSubdir(opt.Repository)
	_ = os.RemoveAll(tmp)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	infof("Cloning the repository...")
	// clone into tmp directly (tmp is empty)
	clone := exec.Command("git", "clone", opt.Repository, tmp)
	clone.Stdin = os.Stdin
	clone.Stdout = os.Stderr
	clone.Stderr = os.Stderr
	// git clone refuses non-empty dir; we created it, so remove first like Rust does
	_ = os.RemoveAll(tmp)
	if err := clone.Run(); err != nil {
		return fmt.Errorf("failed to clone: %w", err)
	}
	if opt.Checkout != "" {
		infof("Checking out to %s...", opt.Checkout)
		co := exec.Command("git", "checkout", opt.Checkout)
		co.Dir = tmp
		co.Stdin = os.Stdin
		co.Stdout = os.Stderr
		co.Stderr = os.Stderr
		if err := co.Run(); err != nil {
			return fmt.Errorf("failed to checkout: %w", err)
		}
	}

	installSrc := tmp
	if opt.Subdir != "" {
		cleanSub := filepath.Clean(opt.Subdir)
		if filepath.IsAbs(cleanSub) || strings.HasPrefix(cleanSub, "..") {
			return fmt.Errorf("invalid subdir %q: path traversal not allowed", opt.Subdir)
		}
		installSrc = filepath.Join(tmp, cleanSub)
		if fi, err := os.Stat(installSrc); err != nil || !fi.IsDir() {
			return fmt.Errorf("subdirectory %q does not exist in repository", opt.Subdir)
		}
	}

	ns := opt.Namespace
	if ns == "" {
		ns = "local"
	}

	if !opt.DryRun {
		infof("Installing...")
	}
	if err := Install(installSrc, ns, opt.DryRun); err != nil {
		return err
	}
	if opt.DryRun {
		// Install already closed the preview with its own summary.
		return nil
	}
	infof("Done")
	return nil
}
