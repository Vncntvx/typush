package universe

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Vncntvx/typush-go/internal/cliutil"
	"github.com/Vncntvx/typush-go/internal/config"
	gh "github.com/Vncntvx/typush-go/internal/github"
	"github.com/Vncntvx/typush-go/internal/manifest"
	"github.com/Vncntvx/typush-go/internal/walker"
)

func publicClient() *gh.Client { return gh.New("") }

func authedClient() (*gh.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if cfg.Tokens.Universe == nil || strings.TrimSpace(*cfg.Tokens.Universe) == "" {
		return nil, fmt.Errorf("you need to set up the token first. Run `typush login universe`")
	}
	return gh.New(strings.TrimSpace(*cfg.Tokens.Universe)), nil
}

// Login stores a GitHub fine-grained PAT in config.toml.
func Login() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.Tokens.Universe != nil {
		fmt.Fprintln(os.Stderr, "Already logged in to the Universe registry")
		if !cliutil.Confirm("Do you want to overwrite the existing token?", false) {
			return nil
		}
	}
	fmt.Fprintln(os.Stderr, "Please create a fine-grained token from https://github.com/settings/personal-access-tokens/new.")
	fmt.Fprintln(os.Stderr, `Grant Contents, Workflows and Pull requests permission on your typst/packages fork.`)
	token, err := cliutil.PromptPassword("Enter your GitHub personal access token")
	if err != nil {
		return err
	}
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("token must not be empty")
	}
	cfg.Tokens.Universe = &token
	if err := config.Save(cfg); err != nil {
		return err
	}
	path, _ := config.ConfigFile()
	fmt.Fprintf(os.Stderr, "Your token has been saved to %s\n", path)
	return nil
}

// WarnIfExists warns when name/version already exist upstream (used by dev --check).
func WarnIfExists(name, version string) error {
	c := publicClient()
	pkgs, err := c.GetContents(gh.UniverseOwner, gh.UniverseRepo, "packages/preview", "main")
	if err != nil {
		return err
	}
	found := false
	for _, p := range pkgs {
		if p.Name == name {
			found = true
		}
	}
	if !found {
		fmt.Fprintf(os.Stderr, "WARN: package `%s` is not available in the Universe (yet)\n", name)
		return nil
	}
	vers, err := c.GetContents(gh.UniverseOwner, gh.UniverseRepo, "packages/preview/"+name, "main")
	if err != nil {
		return err
	}
	for _, v := range vers {
		if v.Name == version {
			fmt.Fprintf(os.Stderr, "WARN: version `%s` is already available in the Universe\n", version)
			return nil
		}
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
	tpl := ""
	if s.hasTemplate {
		tpl = "\n- [x] ensured that my package is licensed such that users can use and distribute the contents of its template directory without restriction, after modifying them through normal use.\n"
	}
	newBox, updBox := " ", " "
	if s.isNewPackage {
		newBox = "x"
	} else {
		updBox = "x"
	}
	return fmt.Sprintf("I am submitting\n- [%s] a new package\n- [%s] an update for a package\n\nDescription: %s\n\nI have read and followed the submission guidelines and, in particular, I\n- [x] selected a name that isn't the most obvious or canonical name for what the package does\n- [x] added a `typst.toml` file with all required keys\n- [x] added a `README.md` with documentation for my package\n- [x] have chosen a license and added a `LICENSE` file or linked one in my `README.md`\n- [x] tested my package locally on my system and it worked\n- [x] `exclude`d PDFs or README images, if any, but not the LICENSE\n%s",
		newBox, updBox, s.description, tpl)
}

func parseSubmissionTitle(title string) (name, version string, ok bool) {
	parts := strings.Split(title, ":")
	if len(parts) != 2 {
		return "", "", false
	}
	if err := manifest.ValidateName(parts[0]); err != nil {
		return "", "", false
	}
	if err := manifest.ValidateVersion(parts[1]); err != nil {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// Publish implements `publish universe` (sparse-checkout only).
func Publish(packageDir string, dryRun bool) error {
	m, err := manifest.Read(packageDir)
	if err != nil {
		return err
	}
	name, version := m.Package.Name, m.Package.Version
	pub := publicClient()

	fmt.Fprintln(os.Stderr, "Checking the packages in the official packages repo...")
	isNew := true
	pkgs, err := pub.GetContents(gh.UniverseOwner, gh.UniverseRepo, "packages/preview", "main")
	if err != nil {
		return err
	}
	for _, p := range pkgs {
		if p.Name != name {
			continue
		}
		fmt.Fprintf(os.Stderr, "Package `%s` found in official packages repo\n", p.Name)
		isNew = false
		vers, err := pub.GetContents(gh.UniverseOwner, gh.UniverseRepo, "packages/preview/"+p.Name, "main")
		if err != nil {
			return err
		}
		var names []string
		for _, v := range vers {
			names = append(names, v.Name)
			if v.Name == version {
				return fmt.Errorf("package version `%s` already exists in the official packages repo", version)
			}
		}
		fmt.Fprintf(os.Stderr, "Existing versions: %s\n", strings.Join(names, ", "))
	}

	fmt.Fprintln(os.Stderr, "Checking the pending PRs...")
	prs, err := pub.ListOpenPulls()
	if err != nil {
		return err
	}
	for _, pr := range prs {
		if pr.Title == nil {
			continue
		}
		n, v, ok := parseSubmissionTitle(*pr.Title)
		if !ok || n != name {
			continue
		}
		switch manifest.CompareVersions(v, version) {
		case 1:
			return fmt.Errorf("package version `%s`(newer) is already submitted in PR #%d", v, pr.Number)
		case 0:
			return fmt.Errorf("package version `%s`(current) is already submitted in PR #%d", v, pr.Number)
		default:
			fmt.Fprintf(os.Stderr, "WARN: package version `%s`(older) is already submitted in PR #%d\n", v, pr.Number)
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

	client, err := authedClient()
	if err != nil {
		return err
	}
	me, err := client.CurrentUser()
	if err != nil {
		return err
	}
	myRepo, err := cliutil.PromptLine("Enter the name of your forked repository", gh.UniverseRepo, false)
	if err != nil {
		return err
	}
	fork, err := client.GetRepo(me.Login, myRepo)
	if err != nil {
		return err
	}
	if fork.Parent == nil {
		return fmt.Errorf("the given repository is not a fork")
	}
	if fork.Parent.Name != gh.UniverseRepo || fork.Parent.Owner == nil || fork.Parent.Owner.Login != gh.UniverseOwner {
		return fmt.Errorf("the given repository is not a fork of the official packages repo")
	}

	fmt.Fprintln(os.Stderr, "Creating corresponding branch in your fork...")
	if !dryRun {
		mainSHA, err := client.GetBranchHead(gh.UniverseOwner, gh.UniverseRepo, "main")
		if err != nil {
			return err
		}
		exists, err := client.BranchExists(me.Login, myRepo, sub.branch())
		if err != nil {
			return err
		}
		if exists {
			if !cliutil.Confirm(fmt.Sprintf("Branch `%s` already exists in your fork. Do you want to overwrite it?", sub.branch()), false) {
				return fmt.Errorf("aborted")
			}
			if err := client.DeleteBranch(me.Login, myRepo, sub.branch()); err != nil {
				return err
			}
		}
		if err := client.CreateBranch(me.Login, myRepo, sub.branch(), mainSHA); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Branch `%s` created\n", sub.branch())
	} else {
		fmt.Fprintln(os.Stderr, "Dry run: branch creation skipped")
	}

	fmt.Fprintln(os.Stderr, "Uploading files to personal fork...")
	absPkg, _ := filepath.Abs(packageDir)
	entries, err := walker.ListPublish(absPkg)
	if err != nil {
		return err
	}
	var files []string // slash-separated rel paths of files only
	for _, abs := range entries {
		if abs == absPkg {
			continue
		}
		fi, err := os.Stat(abs)
		if err != nil || fi.IsDir() {
			continue
		}
		rel, _ := filepath.Rel(absPkg, abs)
		files = append(files, filepath.ToSlash(rel))
	}
	fmt.Fprintf(os.Stderr, "Files to upload:\n\t%s\n", strings.Join(files, "\n\t"))
	if !dryRun {
		if !cliutil.Confirm("Do you want to continue?", false) {
			return fmt.Errorf("aborted")
		}
		if !cliutil.GitSupportsSparseCheckout() {
			return fmt.Errorf("git >= 2.25 is required for sparse-checkout upload; please upgrade git")
		}
		if err := uploadSparse(client, me.Login, myRepo, sub, absPkg, files); err != nil {
			return err
		}
	} else {
		fmt.Fprintln(os.Stderr, "Dry run: file upload skipped")
	}

	fmt.Fprintln(os.Stderr, "Generating submission PR...")
	if !dryRun {
		pr, err := client.CreatePull(sub.title(), me.Login+":"+sub.branch(), "main", sub.prBody(), true)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "PR created: %s\n", pr.HTMLURL)
	} else {
		fmt.Fprintln(os.Stderr, "Dry run: PR creation skipped")
	}
	return nil
}

func uploadSparse(client *gh.Client, userLogin, repoName string, sub submission, packageDir string, files []string) error {
	typstToml, err := os.ReadFile(filepath.Join(packageDir, "typst.toml"))
	if err != nil {
		return err
	}
	if err := client.CreateFile(userLogin, repoName, sub.repoPath()+"/typst.toml",
		"[Typship] Initialize package version directory", sub.branch(), typstToml); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "typush-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	forkURL := fmt.Sprintf("https://github.com/%s/%s.git", userLogin, repoName)
	run := func(dir string, args ...string) error {
		cmd := exec.Command("git", args...)
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
	cmd := exec.Command("git", "commit", "-m", fmt.Sprintf("[Typship] Add package %s:%s", sub.name, sub.version))
	cmd.Dir = repoPath
	if out, err := cmd.CombinedOutput(); err != nil {
		if !strings.Contains(string(out), "nothing to commit") {
			return fmt.Errorf("git commit: %v: %s", err, strings.TrimSpace(string(out)))
		}
	}
	if err := run(repoPath, "push", "origin", sub.branch()); err != nil {
		return err
	}
	return nil
}
