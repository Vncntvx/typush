package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Vncntvx/typush/checker"
	"github.com/Vncntvx/typush/manifest"
	"github.com/Vncntvx/typush/util"
)

const (
	UniverseOwner = "typst"
	UniverseRepo  = "packages"
)

// Login verifies GitHub authentication via the gh CLI, offering to run
// `gh auth login` when needed. typush performs all Universe registry
// operations through gh, so no token is stored locally anymore.
func Login() error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("the `gh` CLI is required but not found in PATH. Install it from https://cli.github.com")
	}
	if err := exec.Command("gh", "auth", "token").Run(); err == nil {
		login, err := ghCurrentUser()
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Already logged in to GitHub as %s (via gh)\n", login)
		return nil
	}
	fmt.Fprintln(os.Stderr, "typush uses the GitHub CLI (gh) for all Universe registry operations.")
	if util.Confirm("Run `gh auth login` now?", true) {
		cmd := exec.Command("gh", "auth", "login")
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
		login, err := ghCurrentUser()
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Logged in to GitHub as %s (via gh)\n", login)
		return nil
	}
	return fmt.Errorf("not logged in to GitHub; run `gh auth login` then retry")
}

// WarnIfExists warns when name/version already exist upstream (used by dev --check).
// It checks the Universe index first and falls back to the GitHub API, which the
// publish path also uses.
func WarnIfExists(name, version string) error {
	var vers []string
	found := false

	idx, err := loadUniverseIndex(false)
	if err == nil {
		vers = idx.Versions(name)
		found = len(vers) > 0
	} else {
		// Fallback to GitHub API via gh
		if err := requireGhPresence(); err != nil {
			return err
		}
		pkgs, err := ghDirNames(UniverseOwner, UniverseRepo, "packages/preview", "main")
		if err != nil {
			return err
		}
		found = slices.Contains(pkgs, name)
		if found {
			if vers, err = ghDirNames(UniverseOwner, UniverseRepo, "packages/preview/"+name, "main"); err != nil {
				return err
			}
		}
	}

	if !found {
		warnf("package `%s` is not found in the Universe", name)
		return nil
	}
	if slices.Contains(vers, version) {
		warnf("version `%s` already exists in the Universe", version)
	}
	return nil
}

type submission struct {
	name         string
	version      string
	isNewPackage bool
	description  string
	hasTemplate  bool
}

func (s submission) title() string    { return s.name + ":" + s.version }
func (s submission) branch() string   { return s.name + "-" + s.version }
func (s submission) repoPath() string { return "packages/preview/" + s.name + "/" + s.version }
func (s submission) prBody() string {
	newBox, updBox := " ", " "
	if s.isNewPackage {
		newBox = "x"
	} else {
		updBox = "x"
	}
	tpl := ""
	if s.hasTemplate {
		tpl = "\n" +
			"<!--\n" +
			"The following box only needs to be checked for **template** submissions. If you're submitting a package that isn't a template, you can delete the following section. See the guidelines section about licenses in the README for more details.\n" +
			"-->\n" +
			"- [x] ensured that my package is licensed such that users can use and distribute the contents of its template directory without restriction, after modifying them through normal use.\n"
	}
	// Mirrors typst/packages' PR template (.github/pull_request_template.md)
	// minus the name Explanation block, boxes pre-checked: byte-identical to
	// the previously accepted submission style.
	out := "<!--\n" +
		"Thanks for submitting a package! Please read and follow the submission guidelines detailed in the repository's README and check the boxes below. Please name your PR as `name:version` of the submitted package.\n" +
		"\n" +
		"If you want to make a PR for something other than a package submission, just delete all this and make a plain PR.\n" +
		"-->\n" +
		"\n" +
		"I am submitting\n" +
		"- [" + newBox + "] a new package\n" +
		"- [" + updBox + "] an update for a package\n" +
		"\n" +
		"<!--\n" +
		"Please add a brief description of your package below and explain why you think it is useful to others. If this is an update, please briefly say what changed.\n" +
		"-->\n" +
		"\n" +
		"Description: " + s.description + "\n" +
		"\n" +
		"<!--\n" +
		"These things need to be checked for a new submission to be merged. If you're just submitting an update, you can delete the following section.\n" +
		"-->\n" +
		"\n" +
		"I have read and followed the submission guidelines and, in particular, I\n" +
		"- [x] selected [a name](https://github.com/typst/packages/blob/main/docs/manifest.md#naming-rules) that isn't the most obvious or canonical name for what the package does\n" +
		"- [x] added a [`typst.toml`](https://github.com/typst/packages/blob/main/docs/manifest.md#package-metadata) file with all required keys\n" +
		"- [x] added a [`README.md`](https://github.com/typst/packages/blob/main/docs/documentation.md) with documentation for my package\n" +
		"- [x] have chosen [a license](https://github.com/typst/packages/blob/main/docs/licensing.md) and added a `LICENSE` file or linked one in my `README.md`\n" +
		"- [x] tested my package locally on my system and it worked\n" +
		"- [x] [`exclude`d](https://github.com/typst/packages/blob/main/docs/tips.md#what-to-commit-what-to-exclude) PDFs or README images, if any, but not the LICENSE\n" +
		tpl
	return strings.ReplaceAll(out, "\n", "\r\n")
}

// parseSubmissionTitle extracts the name and version from a submission PR title
// such as "name:1.2.3" or "@preview/name:1.2.3".
func parseSubmissionTitle(title string) (name, version string, ok bool) {
	name, version, err := parsePackageSpec(title)
	if err != nil || version == "" {
		return "", "", false
	}
	return name, version, true
}

// Publish implements `publish universe` (sparse-checkout only).
func Publish(packageDir string, dryRun bool) error {
	// Local Universe review first: same hard errors as official CI.
	if err := checker.Run(packageDir); err != nil {
		return fmt.Errorf("local Universe check failed: %w", err)
	}
	m, err := manifest.Read(packageDir)
	if err != nil {
		return err
	}
	name, version := m.Package.Name, m.Package.Version

	fmt.Fprintln(os.Stderr, "Checking official packages repository...")
	isNew := true
	pkgs, err := ghDirNames(UniverseOwner, UniverseRepo, "packages/preview", "main")
	if err != nil {
		return err
	}
	for _, p := range pkgs {
		if p != name {
			continue
		}
		fmt.Fprintf(os.Stderr, "Package `%s` found in official repository\n", p)
		isNew = false
		vers, err := ghDirNames(UniverseOwner, UniverseRepo, "packages/preview/"+p, "main")
		if err != nil {
			return err
		}
		var names []string
		for _, v := range vers {
			names = append(names, v)
			if v == version {
				return fmt.Errorf("package version `%s` already exists in official packages repository", version)
			}
		}
		fmt.Fprintf(os.Stderr, "Existing versions: %s\n", strings.Join(names, ", "))
	}

	fmt.Fprintln(os.Stderr, "Checking open pull requests...")
	prs, err := ghOpenPullsForPackage(name)
	if err != nil {
		return err
	}
	for _, pr := range prs {
		if pr.Title == "" {
			continue
		}
		n, v, ok := parseSubmissionTitle(pr.Title)
		if !ok || n != name {
			continue
		}
		switch manifest.CompareVersions(v, version) {
		case 1:
			return fmt.Errorf("package version `%s` (newer) is already submitted in PR #%d", v, pr.Number)
		case 0:
			return fmt.Errorf("package version `%s` (current) is already submitted in PR #%d", v, pr.Number)
		default:
			warnf("package version `%s` (older) is already submitted in PR #%d", v, pr.Number)
		}
	}

	desc := ""
	if m.Package.Description != nil {
		desc = *m.Package.Description
	}
	if strings.TrimSpace(desc) == "" {
		return fmt.Errorf("missing description")
	}
	sub := submission{name: name, version: version, isNewPackage: isNew, description: desc, hasTemplate: m.Template != nil}

	if err := requireGhAuth(); err != nil {
		return err
	}
	me, err := ghCurrentUser()
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "Checking fork of packages repository...")
	myRepo, err := ghFindUserFork()
	if err != nil {
		return err
	}
	if myRepo == "" {
		if !util.Confirm(fmt.Sprintf("No fork of %s/%s found. Create one now?", UniverseOwner, UniverseRepo), true) {
			return fmt.Errorf("aborted: fork required to create pull request")
		}
		fmt.Fprintf(os.Stderr, "Forking %s/%s to your account...\n", UniverseOwner, UniverseRepo)
		myRepo, err = ghEnsureFork()
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Fork created: %s/%s\n", me, myRepo)
	} else {
		fmt.Fprintf(os.Stderr, "Found fork: %s/%s\n", me, myRepo)
	}

	fmt.Fprintln(os.Stderr, "Creating branch in fork...")
	if !dryRun {
		mainSHA, err := ghBranchHead(UniverseOwner, UniverseRepo, "main")
		if err != nil {
			return err
		}
		exists, err := ghBranchExists(me, myRepo, sub.branch())
		if err != nil {
			return err
		}
		if exists {
			if !util.Confirm(fmt.Sprintf("Branch `%s` already exists in your fork. Do you want to overwrite it?", sub.branch()), false) {
				return fmt.Errorf("aborted")
			}
			if err := ghDeleteBranch(me, myRepo, sub.branch()); err != nil {
				return err
			}
		}
		if err := ghCreateBranch(me, myRepo, sub.branch(), mainSHA); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Branch `%s` created\n", sub.branch())
	} else {
		previewNote("branch creation skipped")
	}

	fmt.Fprintln(os.Stderr, "Uploading package files to fork...")
	absPkg, _ := filepath.Abs(packageDir)
	files, err := publishableFiles(absPkg)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Files to upload:\n\t%s\n", strings.Join(files, "\n\t"))
	if !dryRun {
		if !util.Confirm("Do you want to continue?", false) {
			return fmt.Errorf("aborted")
		}
		if !util.GitSupportsSparseCheckout() {
			return fmt.Errorf("git >= 2.25 is required for sparse-checkout upload; please upgrade git")
		}
		if err := uploadSparse(me, myRepo, sub, absPkg, files); err != nil {
			return err
		}
	} else {
		previewNote("file upload skipped")
	}

	fmt.Fprintln(os.Stderr, "Opening submission pull request...")
	if !dryRun {
		url, err := ghCreateDraftPR(me+":"+sub.branch(), "main", sub.title(), sub.prBody())
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "PR created: %s\n", url)
	} else {
		previewNote("PR creation skipped")
	}
	return nil
}

// publishableFiles lists the slash-separated relative paths of the files that
// will be uploaded for packageDir, honouring .typstignore and .gitignore.
// package.exclude is applied during installation, not when uploading files.
func publishableFiles(packageDir string) ([]string, error) {
	entries, err := util.ListPublish(packageDir)
	if err != nil {
		return nil, err
	}
	relEntries, err := util.RelEntries(entries, packageDir)
	if err != nil {
		return nil, err
	}
	var files []string // slash-separated rel paths of files only
	for _, e := range relEntries {
		if fi, err := os.Stat(e.Abs); err != nil || fi.IsDir() {
			continue
		}
		files = append(files, e.Rel)
	}
	return files, nil
}

func uploadSparse(userLogin, repoName string, sub submission, packageDir string, files []string) error {
	tmp, err := os.MkdirTemp("", "typush-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	forkURL := fmt.Sprintf("https://github.com/%s/%s.git", userLogin, repoName)
	run := func(dir string, args ...string) error {
		gitArgs := append([]string{
			"-c", "credential.helper=",
			"-c", "credential.helper=!gh auth git-credential",
		}, args...)
		cmd := exec.Command("git", gitArgs...)
		cmd.Dir = dir
		cmd.Stdin = os.Stdin
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
		}
		return nil
	}
	if err := run(tmp, "clone", "--filter=blob:none", "--no-checkout", "--single-branch", "--branch", sub.branch(), forkURL, "repo"); err != nil {
		return err
	}
	repoPath := filepath.Join(tmp, "repo")
	if err := run(repoPath, "sparse-checkout", "init", "--cone"); err != nil {
		return err
	}
	if err := run(repoPath, "sparse-checkout", "set", sub.repoPath()); err != nil {
		return err
	}
	if err := run(repoPath, "checkout"); err != nil {
		return err
	}
	localTarget := filepath.Join(repoPath, filepath.FromSlash(sub.repoPath()))
	if err := os.MkdirAll(localTarget, 0o755); err != nil {
		return err
	}
	for _, f := range files {
		src := filepath.Join(packageDir, filepath.FromSlash(f))
		dst := filepath.Join(localTarget, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return err
		}
	}
	if err := run(repoPath, "add", "."); err != nil {
		return err
	}
	// commit may report nothing to commit; tolerate it
	if err := run(repoPath, "commit", "-m", fmt.Sprintf("[typush] Add package %s:%s", sub.name, sub.version)); err != nil {
		if !strings.Contains(err.Error(), "nothing to commit") {
			return err
		}
	}
	if err := run(repoPath, "push", "origin", sub.branch()); err != nil {
		return err
	}
	return nil
}
