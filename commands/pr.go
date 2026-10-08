package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/Vncntvx/typush/manifest"
)

// resolvePR resolves the PR reference from argument or local package manifest.
func resolvePR(dir, prRef string) (string, error) {
	if prRef != "" {
		return prRef, nil
	}
	m, err := manifest.Read(dir)
	if err != nil {
		return "", fmt.Errorf("no pull request specified and failed to read typst.toml in %s: %w", dir, err)
	}
	pkgName := m.Package.Name
	prs, err := ghOpenPullsForPackage(pkgName)
	if err != nil {
		return "", err
	}
	var matched []ghPull
	for _, pr := range prs {
		n, _, ok := parseSubmissionTitle(pr.Title)
		if ok && n == pkgName {
			matched = append(matched, pr)
		}
	}
	if len(matched) == 0 {
		return "", fmt.Errorf("no open PR found for package %q in %s/%s", pkgName, UniverseOwner, UniverseRepo)
	}
	// Prefer the PR matching the current manifest version
	currTitle := pkgName + ":" + m.Package.Version
	for _, pr := range matched {
		if pr.Title == currTitle {
			return strconv.Itoa(pr.Number), nil
		}
	}
	// Otherwise use the most recent matched PR
	return strconv.Itoa(matched[0].Number), nil
}

// PRStatus displays the PR overview and conversation comments.
func PRStatus(dir, prRef string) error {
	ref, err := resolvePR(dir, prRef)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Viewing PR #%s in %s/%s...\n", ref, UniverseOwner, UniverseRepo)
	return ghPRView(ref)
}

// PRChecks displays or watches the CI check runs on the PR.
func PRChecks(dir, prRef string, watch bool) error {
	ref, err := resolvePR(dir, prRef)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Checking CI status for PR #%s in %s/%s...\n", ref, UniverseOwner, UniverseRepo)
	if err := ghPRChecks(ref, watch); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("some CI checks were not successful")
		}
		return err
	}
	return nil
}
