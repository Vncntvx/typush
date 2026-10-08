package commands_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vncntvx/typush/commands"
	"github.com/Vncntvx/typush/util"
)

func TestClean_DryRun(t *testing.T) {
	// Isolate TypstLocalDir by setting XDG_DATA_HOME
	tmpData := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpData)

	base, err := util.TypstLocalDir()
	if err != nil {
		t.Fatalf("failed to get local dir: %v", err)
	}

	pkgDir := filepath.Join(base, "preview", "demo-clean")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("failed to create preview dir: %v", err)
	}

	// Create a target package directory and symlink it
	targetDir := t.TempDir()
	link010 := filepath.Join(pkgDir, "0.1.0")
	if err := os.Symlink(targetDir, link010); err != nil {
		t.Skipf("symlink not supported on this platform: %v", err)
	}
	// A broken symlink is still a dev link, so it must be previewed as one.
	link099 := filepath.Join(pkgDir, "0.9.9")
	if err := os.Symlink(filepath.Join(pkgDir, "missing-target"), link099); err != nil {
		t.Skipf("symlink not supported on this platform: %v", err)
	}

	// 1. Dry run clean of one package
	stderr, err := captureStderr(t, func() error {
		return commands.CleanOne("demo-clean", commands.Preview)
	})
	if err != nil {
		t.Fatalf("CleanOne dryRun failed: %v", err)
	}
	if !strings.Contains(stderr, "Dry run: symlinks to remove (2 item(s)):") {
		t.Errorf("expected preview count in stderr, got: %s", stderr)
	}
	if !strings.Contains(stderr, "@preview/demo-clean/0.1.0 -> ") {
		t.Errorf("expected symlink target in stderr, got: %s", stderr)
	}
	if !strings.Contains(stderr, "@preview/demo-clean/0.9.9 -> (broken)") {
		t.Errorf("expected broken symlink in stderr, got: %s", stderr)
	}
	if !strings.Contains(stderr, "Dry run: 2 symlink(s) would be removed for package `demo-clean`") {
		t.Errorf("expected per-package summary in stderr, got: %s", stderr)
	}

	// Verify symlinks still exist
	for _, l := range []string{link010, link099} {
		if fi, err := os.Lstat(l); err != nil || fi.Mode()&os.ModeSymlink == 0 {
			t.Errorf("symlink %s should still exist after dry-run, err=%v", l, err)
		}
	}

	// 2. Dry run clean all reports one aggregate summary instead of a trailer
	// per package followed by another global one.
	stderrAll, err := captureStderr(t, func() error {
		return commands.CleanAll(commands.Preview)
	})
	if err != nil {
		t.Fatalf("CleanAll dryRun failed: %v", err)
	}
	if !strings.Contains(stderrAll, "@preview/demo-clean/0.1.0 -> ") {
		t.Errorf("expected preview message in stderrAll, got: %s", stderrAll)
	}
	if !strings.Contains(stderrAll, "Dry run: 2 symlink(s) would be removed across 1 package(s)") {
		t.Errorf("expected aggregate summary in stderrAll, got: %s", stderrAll)
	}
	if n := strings.Count(stderrAll, "Dry run: "); n != 2 {
		t.Errorf("expected one item block plus one summary, got %d dry-run lines:\n%s", n, stderrAll)
	}

	// Verify symlinks still exist
	for _, l := range []string{link010, link099} {
		if fi, err := os.Lstat(l); err != nil || fi.Mode()&os.ModeSymlink == 0 {
			t.Errorf("symlink %s should still exist after CleanAll dry-run, err=%v", l, err)
		}
	}

	// 3. Real clean
	if err := commands.CleanOne("demo-clean", commands.Execute); err != nil {
		t.Fatalf("CleanOne failed: %v", err)
	}
	for _, l := range []string{link010, link099} {
		if _, err := os.Lstat(l); !os.IsNotExist(err) {
			t.Errorf("symlink %s should have been deleted, err=%v", l, err)
		}
	}
}

// A preview with nothing to do stays silent instead of reassuring the user.
func TestClean_DryRunWithoutLinks(t *testing.T) {
	tmpData := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpData)

	base, err := util.TypstLocalDir()
	if err != nil {
		t.Fatalf("failed to get local dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(base, "preview", "empty-pkg"), 0o755); err != nil {
		t.Fatalf("failed to create preview dir: %v", err)
	}

	stderr, err := captureStderr(t, func() error {
		return commands.CleanAll(commands.Preview)
	})
	if err != nil {
		t.Fatalf("CleanAll dryRun failed: %v", err)
	}
	if !strings.Contains(stderr, "Dry run: no dev symlinks found to remove") {
		t.Errorf("expected empty preview message, got: %s", stderr)
	}
}
