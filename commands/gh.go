package commands

import (
	"bytes"
	"encoding/base64"
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
	cmd := exec.Command("gh", "auth", "status")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gh is not authenticated. Run `gh auth login` first: %s", firstLine(string(out)))
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
	cmd := exec.Command("gh", "api")
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
// Note: the ref goes into the endpoint path, not -f: gh turns -f into a
// body/query form that the contents API answers 404 to.
func ghDirNames(owner, repo, dir, ref string) ([]string, error) {
	out, err := ghAPI(fmt.Sprintf("repos/%s/%s/contents/%s?ref=%s", owner, repo, strings.TrimPrefix(dir, "/"), ref),
		"--jq", ".[].name")
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

// ghOpenPulls lists open PRs of the Universe repo (title + number).
func ghOpenPulls() ([]ghPull, error) {
	if err := requireGhPresence(); err != nil {
		return nil, err
	}
	cmd := exec.Command("gh", "pr", "list", "--repo", UniverseOwner+"/"+UniverseRepo,
		"--state", "open", "--json", "number,title", "--limit", "500")
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

// ghBranchExists reports whether a branch exists.
func ghBranchExists(owner, repo, branch string) (bool, error) {
	out, err := ghAPI(fmt.Sprintf("repos/%s/%s/branches", owner, repo), "--paginate", "--jq", ".[].name")
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == branch {
			return true, nil
		}
	}
	return false, nil
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

// ghCreateFile creates a file via the Contents API (used to initialize the
// version directory before the sparse-checkout upload).
func ghCreateFile(owner, repo, path, message, branch string, content []byte) error {
	_, err := ghAPI("--method", "PUT",
		fmt.Sprintf("repos/%s/%s/contents/%s", owner, repo, strings.TrimPrefix(path, "/")),
		"-f", "message="+message,
		"-f", "content="+base64.StdEncoding.EncodeToString(content),
		"-f", "branch="+branch)
	return err
}

// ghCreateDraftPR opens a draft PR and returns its URL.
func ghCreateDraftPR(head, base, title, body string) (string, error) {
	f, err := os.CreateTemp("", "typush-pr-*")
	if err != nil {
		return "", err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err := f.WriteString(body); err != nil {
		f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	cmd := exec.Command("gh", "pr", "create",
		"--repo", UniverseOwner+"/"+UniverseRepo,
		"--head", head, "--base", base,
		"--title", title, "--body-file", name, "--draft")
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
