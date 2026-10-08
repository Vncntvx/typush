package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// This file shells out to the GitHub CLI (`gh`) for all registry operations.
// Rationale: `gh` already handles auth (keyring, GH_TOKEN, `gh auth login`)
// with a token that can actually open PRs against typst/packages — something
// a self-stored fine-grained PAT kept failing at.

// requireGhPresence ensures the `gh` binary exists.
func requireGhPresence() error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("the `gh` CLI is required but not found in PATH. Install it from https://cli.github.com")
	}
	return nil
}

// requireGhAuth ensures `gh` exists and is authenticated.
func requireGhAuth() error {
	if err := requireGhPresence(); err != nil {
		return err
	}
	cmd := exec.Command("gh", "auth", "token")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gh is not authenticated. Run `gh auth login` or set GH_TOKEN: %s", firstLine(string(out)))
	}
	return nil
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

// ghAPI runs `gh api ...` and returns stdout. API/CLI errors are surfaced
// with gh's own stderr message.
func ghAPI(args ...string) ([]byte, error) {
	cmd := exec.Command("gh", "api", "-H", "X-GitHub-Api-Version: 2022-11-28")
	cmd.Args = append(cmd.Args, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := firstLine(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("gh api %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.Bytes(), nil
}

// ghDirNames lists entry names of a directory in a repo at a ref.
// It uses the Git Trees API: the Contents API caps directory listings at 1000
// entries, and `packages/preview` holds more than that. Without the Trees API,
// entries sorting after the cap are invisible and updates look like new
// packages.
// Note: the ref and path go into the endpoint path, not -f: gh turns -f into
// a body/query form that the API answers 404 to.
func ghDirNames(owner, repo, dir, ref string) ([]string, error) {
	out, err := ghAPI(fmt.Sprintf("repos/%s/%s/git/trees/%s:%s", owner, repo, ref, strings.TrimPrefix(dir, "/")),
		"--jq", `.tree[] | select(.type == "tree") | .path`)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			names = append(names, line)
		}
	}
	return names, nil
}

type ghPull struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
}

// ghOpenPullsForPackage lists open PRs of the Universe repo for a package (title + number).
func ghOpenPullsForPackage(pkgName string) ([]ghPull, error) {
	if err := requireGhPresence(); err != nil {
		return nil, err
	}
	cmd := exec.Command("gh", "pr", "list", "--repo", UniverseOwner+"/"+UniverseRepo,
		"--state", "open", "--search", fmt.Sprintf("%s in:title", pkgName),
		"--json", "number,title")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := firstLine(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("gh pr list: %s", msg)
	}
	var pulls []ghPull
	if err := json.Unmarshal(stdout.Bytes(), &pulls); err != nil {
		return nil, fmt.Errorf("decode gh pr list: %w", err)
	}
	return pulls, nil
}

// ghCurrentUser returns the gh-authenticated login.
func ghCurrentUser() (string, error) {
	out, err := ghAPI("user", "--jq", ".login")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ghFindUserFork finds the user's fork of UniverseOwner/UniverseRepo.
// Returns the fork repo name (e.g. "packages"), or "" if no fork exists.
func ghFindUserFork() (string, error) {
	query := fmt.Sprintf(`query {
		repository(owner: %q, name: %q) {
			forks(affiliations: OWNER, first: 1) {
				nodes {
					name
				}
			}
		}
	}`, UniverseOwner, UniverseRepo)
	out, err := ghAPI("graphql", "-f", "query="+query, "--jq", ".data.repository.forks.nodes[0].name // empty")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ghEnsureFork ensures the user has a fork of UniverseOwner/UniverseRepo.
// If not found, it provisions one using `gh repo fork --clone=false`.
func ghEnsureFork() (string, error) {
	name, err := ghFindUserFork()
	if err != nil {
		return "", err
	}
	if name != "" {
		return name, nil
	}
	cmd := exec.Command("gh", "repo", "fork", UniverseOwner+"/"+UniverseRepo, "--clone=false")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := firstLine(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("gh repo fork: %s", msg)
	}
	name, err = ghFindUserFork()
	if err != nil || name == "" {
		return UniverseRepo, nil
	}
	return name, nil
}

// ghForkParentFull returns the parent's "owner/repo" of a repo, or "" if none.
func ghForkParentFull(owner, repo string) (string, error) {
	out, err := ghAPI(fmt.Sprintf("repos/%s/%s", owner, repo), "--jq", `.parent.full_name // "none"`)
	if err != nil {
		return "", err
	}
	if s := strings.TrimSpace(string(out)); s != "" && s != "none" {
		return s, nil
	}
	return "", nil
}

// ghBranchHead returns the HEAD SHA of a branch.
func ghBranchHead(owner, repo, branch string) (string, error) {
	out, err := ghAPI(fmt.Sprintf("repos/%s/%s/git/ref/heads/%s", owner, repo, branch), "--jq", ".object.sha")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ghBranchExists reports whether a branch exists in a repo using O(1) git ref query.
func ghBranchExists(owner, repo, branch string) (bool, error) {
	_, err := ghAPI(fmt.Sprintf("repos/%s/%s/git/ref/heads/%s", owner, repo, branch))
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "Not Found") || strings.Contains(errMsg, "404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ghDeleteBranch deletes a branch via the git-refs API.
func ghDeleteBranch(owner, repo, branch string) error {
	_, err := ghAPI("--method", "DELETE",
		fmt.Sprintf("repos/%s/%s/git/refs/heads/%s", owner, repo, branch))
	return err
}

// ghCreateBranch creates a branch at sha via the git-refs API.
func ghCreateBranch(owner, repo, branch, sha string) error {
	_, err := ghAPI("--method", "POST", fmt.Sprintf("repos/%s/%s/git/refs", owner, repo),
		"-f", "ref=refs/heads/"+branch, "-f", "sha="+sha)
	return err
}

// ghCreateDraftPR opens a draft PR using stdin streaming and returns its URL.
func ghCreateDraftPR(head, base, title, body string) (string, error) {
	cmd := exec.Command("gh", "pr", "create",
		"--repo", UniverseOwner+"/"+UniverseRepo,
		"--head", head, "--base", base,
		"--title", title, "--body-file", "-", "--draft")
	cmd.Stdin = strings.NewReader(body)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := firstLine(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("gh pr create: %s", msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// ghPRView views a PR on the official packages repository.
func ghPRView(prRef string) error {
	if err := requireGhPresence(); err != nil {
		return err
	}
	args := []string{"pr", "view", "--repo", UniverseOwner + "/" + UniverseRepo}
	if prRef != "" {
		args = append(args, prRef)
	}
	cmd := exec.Command("gh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ghPRChecks views CI checks on a PR on the official packages repository.
func ghPRChecks(prRef string, watch bool) error {
	if err := requireGhPresence(); err != nil {
		return err
	}
	args := []string{"pr", "checks", "--repo", UniverseOwner + "/" + UniverseRepo}
	if prRef != "" {
		args = append(args, prRef)
	}
	if watch {
		args = append(args, "--watch")
	}
	cmd := exec.Command("gh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
