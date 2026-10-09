package util_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vncntvx/typush/util"
)

func TestScanInstalledPackagesEmpty(t *testing.T) {
	emptyDir := t.TempDir()
	pkgs, err := util.ScanInstalledPackages(emptyDir, util.SourceInstalled)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 0 {
		t.Fatalf("expected 0 packages, got %d", len(pkgs))
	}
}

func TestScanInstalledPackages(t *testing.T) {
	root := t.TempDir()

	// 1. Regular installed package: @local/my-lib/0.1.0
	localPkg := filepath.Join(root, "local", "my-lib", "0.1.0")
	if err := os.MkdirAll(localPkg, 0o755); err != nil {
		t.Fatal(err)
	}

	// 2. Another version: @local/my-lib/0.2.0
	localPkg2 := filepath.Join(root, "local", "my-lib", "0.2.0")
	if err := os.MkdirAll(localPkg2, 0o755); err != nil {
		t.Fatal(err)
	}

	// 3. Dev symlink: @preview/dev-pkg/0.3.0 -> sourceDir
	devSource := t.TempDir()
	previewDevDir := filepath.Join(root, "preview", "dev-pkg")
	if err := os.MkdirAll(previewDevDir, 0o755); err != nil {
		t.Fatal(err)
	}
	devLink := filepath.Join(previewDevDir, "0.3.0")
	if err := os.Symlink(devSource, devLink); err != nil {
		t.Fatal(err)
	}

	// 4. Broken symlink: @preview/broken-pkg/0.4.0 -> non-existent
	previewBrokenDir := filepath.Join(root, "preview", "broken-pkg")
	if err := os.MkdirAll(previewBrokenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	brokenLink := filepath.Join(previewBrokenDir, "0.4.0")
	if err := os.Symlink(filepath.Join(devSource, "non-existent"), brokenLink); err != nil {
		t.Fatal(err)
	}

	// Scan all packages
	pkgs, err := util.ScanInstalledPackages(root, util.SourceInstalled)
	if err != nil {
		t.Fatalf("ScanInstalledPackages failed: %v", err)
	}

	if len(pkgs) != 4 {
		t.Fatalf("expected 4 packages, got %d", len(pkgs))
	}

	// Verify sources
	byPath := make(map[string]util.InstalledPackage)
	for _, p := range pkgs {
		byPath[p.Path] = p
	}

	if p, ok := byPath[localPkg]; !ok || p.Source != util.SourceInstalled {
		t.Errorf("expected %s to have SourceInstalled, got %+v", localPkg, p)
	}
	if p, ok := byPath[devLink]; !ok || p.Source != util.SourceDevLink {
		t.Errorf("expected %s to have SourceDevLink, got %+v", devLink, p)
	}
	if p, ok := byPath[brokenLink]; !ok || p.Source != util.SourceBrokenLink {
		t.Errorf("expected %s to have SourceBrokenLink, got %+v", brokenLink, p)
	}

	// Test namespace filter
	localOnly, err := util.ScanInstalledPackagesInNamespace(root, "local", util.SourceInstalled)
	if err != nil {
		t.Fatalf("ScanInstalledPackagesInNamespace failed: %v", err)
	}
	if len(localOnly) != 2 {
		t.Errorf("expected 2 packages in @local, got %d", len(localOnly))
	}

	// Test namespace filter with @ prefix
	localWithAt, err := util.ScanInstalledPackagesInNamespace(root, "@local", util.SourceInstalled)
	if err != nil {
		t.Fatalf("ScanInstalledPackagesInNamespace failed: %v", err)
	}
	if len(localWithAt) != 2 {
		t.Errorf("expected 2 packages in @local, got %d", len(localWithAt))
	}
}

func TestScanInstalledPackages_InvalidNamespace(t *testing.T) {
	root := t.TempDir()
	for _, invalid := range []string{"../escape", "sub/dir", "a\\b", ".."} {
		_, err := util.ScanInstalledPackagesInNamespace(root, invalid, util.SourceInstalled)
		if err == nil {
			t.Errorf("expected error for invalid namespace %q, got nil", invalid)
		}
	}
}

func TestSortInstalledPackages(t *testing.T) {
	pkgs := []util.InstalledPackage{
		{Namespace: "local", Name: "pkg", Version: "0.2.0"},
		{Namespace: "local", Name: "pkg", Version: "0.10.0"},
		{Namespace: "local", Name: "pkg", Version: "1.0.0"},
		{Namespace: "alpha", Name: "z", Version: "0.1.0"},
	}

	util.SortInstalledPackages(pkgs)

	// alpha comes before local
	if pkgs[0].Namespace != "alpha" {
		t.Errorf("expected alpha namespace first, got %s", pkgs[0].Namespace)
	}

	// For local/pkg: versions should be descending (1.0.0, 0.10.0, 0.2.0)
	expectedVersions := []string{"1.0.0", "0.10.0", "0.2.0"}
	for i, want := range expectedVersions {
		got := pkgs[i+1].Version
		if got != want {
			t.Errorf("expected local/pkg[%d] to be %s, got %s", i, want, got)
		}
	}
}
