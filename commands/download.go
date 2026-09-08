package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/Vncntvx/typkg/util"
)

// Run clones repo (+ optional checkout ref) into a temp dir then installs.
func Download(repository, checkout, namespace string) error {
	tmp := util.TempSubdir(repository)
	_ = os.RemoveAll(tmp)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	fmt.Fprintln(os.Stderr, "Cloning the repository...")
	// clone into tmp directly (tmp is empty)
	clone := exec.Command("git", "clone", repository, tmp)
	clone.Stdin = os.Stdin
	clone.Stdout = os.Stderr
	clone.Stderr = os.Stderr
	// git clone refuses non-empty dir; we created it, so remove first like Rust does
	_ = os.RemoveAll(tmp)
	if err := clone.Run(); err != nil {
		return fmt.Errorf("failed to clone: %w", err)
	}
	if checkout != "" {
		fmt.Fprintf(os.Stderr, "Checking out to %s...\n", checkout)
		co := exec.Command("git", "checkout", checkout)
		co.Dir = tmp
		co.Stdin = os.Stdin
		co.Stdout = os.Stderr
		co.Stderr = os.Stderr
		if err := co.Run(); err != nil {
			return fmt.Errorf("failed to checkout: %w", err)
		}
	}
	fmt.Fprintln(os.Stderr, "Installing...")
	if err := Install(tmp, namespace); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "Done")
	return nil
}
