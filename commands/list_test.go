package commands_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vncntvx/typush/commands"
	"github.com/Vncntvx/typush/util"
)

func TestListEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("TYPST_PACKAGE_PATH", tmpDir)
	t.Setenv("TYPST_PACKAGE_CACHE_PATH", filepath.Join(tmpDir, "cache"))

	stdout, stderr, err := captureStdio(t, func() error {
		return commands.List(commands.ListOptions{})
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if stdout != "" {
		t.Errorf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "No installed packages found") {
		t.Errorf("expected 'No installed packages found' in stderr, got %q", stderr)
	}
}

func TestListEmptyJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("TYPST_PACKAGE_PATH", tmpDir)
	t.Setenv("TYPST_PACKAGE_CACHE_PATH", filepath.Join(tmpDir, "cache"))

	stdout, err := captureStdout(t, func() error {
		return commands.List(commands.ListOptions{JSON: true})
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	trimmed := strings.TrimSpace(stdout)
	if trimmed != "[]" {
		t.Errorf("expected '[]', got %q", trimmed)
	}
}

func setupTestPackages(t *testing.T) (string, string) {
	t.Helper()
	localRoot := t.TempDir()
	cacheRoot := t.TempDir()

	// @local/theme/1.0.0
	p1 := filepath.Join(localRoot, "local", "theme", "1.0.0")
	if err := os.MkdirAll(p1, 0o755); err != nil {
		t.Fatal(err)
	}

	// @local/lib/0.2.0
	p2 := filepath.Join(localRoot, "local", "lib", "0.2.0")
	if err := os.MkdirAll(p2, 0o755); err != nil {
		t.Fatal(err)
	}

	// @preview/dev-pkg/0.1.0 -> symlink
	devSrc := t.TempDir()
	devLinkDir := filepath.Join(localRoot, "preview", "dev-pkg")
	if err := os.MkdirAll(devLinkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(devSrc, filepath.Join(devLinkDir, "0.1.0")); err != nil {
		t.Fatal(err)
	}

	// @preview/touying/0.5.0 in cache
	cachePkg := filepath.Join(cacheRoot, "preview", "touying", "0.5.0")
	if err := os.MkdirAll(cachePkg, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("TYPST_PACKAGE_PATH", localRoot)
	t.Setenv("TYPST_PACKAGE_CACHE_PATH", cacheRoot)

	return localRoot, cacheRoot
}

func TestListTable(t *testing.T) {
	setupTestPackages(t)

	stdout, err := captureStdout(t, func() error {
		return commands.List(commands.ListOptions{})
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	for _, want := range []string{
		"NAMESPACE", "PACKAGE", "VERSION", "SOURCE", "PATH",
		"@local", "theme", "1.0.0", "installed",
		"@local", "lib", "0.2.0", "installed",
		"@preview", "dev-pkg", "0.1.0", "dev-link",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected %q in table output, got:\n%s", want, stdout)
		}
	}

	// Without --all, cache package should not be listed
	if strings.Contains(stdout, "touying") {
		t.Errorf("cache package 'touying' should not be listed without --all")
	}
}

func TestListWithCache(t *testing.T) {
	setupTestPackages(t)

	stdout, err := captureStdout(t, func() error {
		return commands.List(commands.ListOptions{All: true})
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if !strings.Contains(stdout, "touying") || !strings.Contains(stdout, "0.5.0") {
		t.Errorf("expected cache package 'touying' with --all, got:\n%s", stdout)
	}
}

func TestListNamespaceFilter(t *testing.T) {
	setupTestPackages(t)

	stdout, err := captureStdout(t, func() error {
		return commands.List(commands.ListOptions{Namespace: "local"})
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if !strings.Contains(stdout, "theme") {
		t.Errorf("expected 'theme' in filtered output, got:\n%s", stdout)
	}
	if strings.Contains(stdout, "dev-pkg") {
		t.Errorf("did not expect 'dev-pkg' from @preview when filtering by local, got:\n%s", stdout)
	}
}

func TestListTree(t *testing.T) {
	setupTestPackages(t)

	stdout, err := captureStdout(t, func() error {
		return commands.List(commands.ListOptions{Tree: true})
	})
	if err != nil {
		t.Fatalf("List tree failed: %v", err)
	}

	for _, want := range []string{
		"@local",
		"theme",
		"1.0.0",
		"@preview",
		"dev-pkg",
		"dev-link",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected %q in tree output, got:\n%s", want, stdout)
		}
	}
}

func TestListJSON(t *testing.T) {
	setupTestPackages(t)

	stdout, err := captureStdout(t, func() error {
		return commands.List(commands.ListOptions{JSON: true})
	})
	if err != nil {
		t.Fatalf("List JSON failed: %v", err)
	}

	var pkgs []util.InstalledPackage
	if err := json.Unmarshal([]byte(stdout), &pkgs); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v, raw:\n%s", err, stdout)
	}
	if len(pkgs) != 3 {
		t.Errorf("expected 3 packages in JSON, got %d", len(pkgs))
	}
}

func TestList_SemVerOrdering(t *testing.T) {
	localRoot := t.TempDir()
	t.Setenv("TYPST_PACKAGE_PATH", localRoot)

	// Create @local/pkg/0.2.0, 0.9.0, and 0.10.0
	for _, v := range []string{"0.2.0", "0.10.0", "0.9.0"} {
		if err := os.MkdirAll(filepath.Join(localRoot, "local", "pkg", v), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	stdout, err := captureStdout(t, func() error {
		return commands.List(commands.ListOptions{JSON: true})
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	var pkgs []util.InstalledPackage
	if err := json.Unmarshal([]byte(stdout), &pkgs); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}

	if len(pkgs) != 3 {
		t.Fatalf("expected 3 packages, got %d", len(pkgs))
	}
	// Newer version first: 0.10.0, 0.9.0, 0.2.0
	expected := []string{"0.10.0", "0.9.0", "0.2.0"}
	for i, want := range expected {
		if pkgs[i].Version != want {
			t.Errorf("pkgs[%d] = %s, want %s (semver descending)", i, pkgs[i].Version, want)
		}
	}
}
