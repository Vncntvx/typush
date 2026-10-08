package commands

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Vncntvx/typush/manifest"
	"github.com/Vncntvx/typush/util"
)

//go:embed release-typst.yml
var releaseWorkflow string

// GenerateOptions configures `ci generate` and the deprecated `generate` alias.
type GenerateOptions struct {
	// Source is the typst/packages repository the workflow publishes to.
	Source string
	// PushToFork is the fork the workflow pushes the release branch to.
	PushToFork string
	// Destination is the path the package is installed to inside Source.
	Destination string
	// DryRun prints the generated workflow to stdout instead of writing a file.
	DryRun bool
}

// Generate writes .github/workflows/release-typst.yml with template substitution.
// If opts.DryRun is true, the workflow is printed to stdout and no file is written.
func Generate(dir string, opts GenerateOptions) error {
	source := opts.Source
	if source == "" {
		source = "typst/packages"
	}
	pushToFork := opts.PushToFork
	if pushToFork == "" {
		pushToFork = detectPushToFork(source)
	}
	destination := opts.Destination
	if destination == "" {
		destination = "packages/preview"
	}
	out := releaseWorkflow
	out = strings.ReplaceAll(out, `"<<"source>>""`, fmt.Sprintf("%q", source))
	out = strings.ReplaceAll(out, `"<<"destination>>""`, fmt.Sprintf("%q", destination))
	out = strings.ReplaceAll(out, `"<<"push-to-fork>>""`, fmt.Sprintf("%q", pushToFork))
	// also support bare placeholders (robust to quoting differences)
	out = strings.ReplaceAll(out, "<<source>>", source)
	out = strings.ReplaceAll(out, "<<destination>>", destination)
	out = strings.ReplaceAll(out, "<<push-to-fork>>", pushToFork)

	if opts.DryRun {
		fmt.Print(out)
		previewNote("release-typst.yml previewed to stdout, file not written")
		return nil
	}

	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(wfDir, "release-typst.yml"), []byte(out), 0o644); err != nil {
		return fmt.Errorf("failed to write release workflow: %w", err)
	}
	fmt.Fprintln(os.Stderr, "Generating project files...")
	return nil
}

func detectPushToFork(fallback string) string {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return fallback
	}
	remote := strings.TrimSpace(string(out))
	// ssh: git@github.com:owner/repo.git  | https: https://github.com/owner/repo
	owner := ""
	if strings.Contains(remote, "github.com") {
		rest := remote
		if i := strings.Index(rest, "github.com"); i >= 0 {
			rest = rest[i+len("github.com"):]
		}
		rest = strings.Trim(rest, ":/")
		parts := strings.Split(rest, "/")
		if len(parts) >= 1 && parts[0] != "" {
			owner = parts[0]
		}
	}
	if owner == "" {
		return fallback
	}
	return owner + "/packages"
}

// Plan scans the workspace for typst.toml files and prints the CI matrix JSON.
func Plan(dir string, only []string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	entries, err := util.ListPublish(abs)
	if err != nil {
		return err
	}
	onlySet := map[string]bool{}
	for _, o := range only {
		onlySet[o] = true
	}
	type pkg struct {
		Source  string `json:"source"`
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	type item struct {
		Runner    string  `json:"runner"`
		Container *string `json:"container,omitempty"`
		Package   pkg     `json:"package"`
	}
	var tasks []item
	for _, abs2 := range entries {
		if filepath.Base(abs2) != "typst.toml" {
			continue
		}
		pkgDir := filepath.Dir(abs2)
		m, err := manifest.Read(pkgDir)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", abs2, err)
		}
		if len(onlySet) > 0 && !onlySet[m.Package.Name] {
			continue
		}
		tasks = append(tasks, item{
			Runner: "ubuntu-24.04",
			Package: pkg{
				Source:  pkgDir,
				Name:    m.Package.Name,
				Version: m.Package.Version,
			},
		})
	}
	if tasks == nil {
		tasks = []item{}
	}
	output := map[string]any{
		"ci": map[string]any{
			"github": map[string]any{
				"artifacts_matrix": map[string]any{"include": tasks},
			},
		},
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		return err
	}
	fmt.Print(buf.String())
	return nil
}
