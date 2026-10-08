package commands_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vncntvx/typush/commands"
	"github.com/Vncntvx/typush/util"
)

// installTestPackage creates the fixture package and returns its directory.
func installTestPackage(t *testing.T) string {
	t.Helper()
	pkgDir := t.TempDir()
	writeTestPackage(t, pkgDir)
	for _, rel := range []string{"lib.typ", filepath.Join("src", "extra.typ")} {
		p := filepath.Join(pkgDir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("failed to create dir for %s: %v", rel, err)
		}
		if err := os.WriteFile(p, []byte("// "+rel), 0o644); err != nil {
			t.Fatalf("failed to write %s: %v", rel, err)
		}
	}
	return pkgDir
}

func TestInstall_DryRun(t *testing.T) {
	tmpData := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpData)

	base, err := util.TypstLocalDir()
	if err != nil {
		t.Fatalf("failed to get local dir: %v", err)
	}

	pkgDir := installTestPackage(t)
	targetDir := filepath.Join(base, "local", "demo-pkg", "1.2.3")

	// 1. Dry run install: the walker returns directories too, so they belong in
	// the preview instead of being silently counted as "files".
	stderr, err := captureStderr(t, func() error {
		return commands.Install(pkgDir, "local", commands.Preview)
	})
	if err != nil {
		t.Fatalf("Install dryRun failed: %v", err)
	}
	if !strings.Contains(stderr, "Destination directory:") {
		t.Errorf("expected destination directory in stderr, got: %s", stderr)
	}
	if !strings.Contains(stderr, "Dry run: entries to install (4 item(s)):") {
		t.Errorf("expected entry count in stderr, got: %s", stderr)
	}
	for _, want := range []string{"lib.typ", "src", "src/extra.typ", "typst.toml"} {
		if !strings.Contains(stderr, "  "+want+"\n") {
			t.Errorf("expected %q in the file list, got: %s", want, stderr)
		}
	}
	if !strings.Contains(stderr, "Dry run: installation skipped, no files written") {
		t.Errorf("expected dry-run skipped message in stderr, got: %s", stderr)
	}

	// Verify target directory was NOT created
	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		t.Errorf("targetDir %s should not exist after dry-run", targetDir)
	}

	// 2. Real install
	if err := commands.Install(pkgDir, "local", commands.Execute); err != nil {
		t.Fatalf("Install failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "lib.typ")); err != nil {
		t.Errorf("expected lib.typ installed at %s, err=%v", targetDir, err)
	}
	if fi, err := os.Stat(filepath.Join(targetDir, "src")); err != nil || !fi.IsDir() {
		t.Errorf("expected src directory installed at %s, err=%v", targetDir, err)
	}

	// 3. Dry run again: should detect that it already exists and would be overwritten
	stderrOverwrite, err := captureStderr(t, func() error {
		return commands.Install(pkgDir, "local", commands.Preview)
	})
	if err != nil {
		t.Fatalf("Install dryRun on existing failed: %v", err)
	}
	if !strings.Contains(stderrOverwrite, "already exists and would be overwritten") {
		t.Errorf("expected overwrite warning in dry-run mode, got: %s", stderrOverwrite)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "lib.typ")); err != nil {
		t.Errorf("dry run must not delete the existing install, err=%v", err)
	}
}

// A preview is fully non-interactive: the @ prefix is normalized silently to the
// user's eyes but always reported, and never prompts.
func TestInstall_DryRunNormalizesNamespace(t *testing.T) {
	tmpData := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpData)

	pkgDir := installTestPackage(t)

	stderr, err := captureStderr(t, func() error {
		return commands.Install(pkgDir, "@local", commands.Preview)
	})
	if err != nil {
		t.Fatalf("Install dryRun failed: %v", err)
	}
	if !strings.Contains(stderr, `Note: normalized namespace "@local" to "local"`) {
		t.Errorf("expected normalization note in stderr, got: %s", stderr)
	}
	if !strings.Contains(stderr, "Dry run: entries to install") {
		t.Errorf("expected the preview to continue after normalization, got: %s", stderr)
	}

	base, err := util.TypstLocalDir()
	if err != nil {
		t.Fatalf("failed to get local dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, "local", "demo-pkg")); !os.IsNotExist(err) {
		t.Errorf("dry run must not create the target, err=%v", err)
	}
}

// The `preview` guard is reported but not confirmed during a dry run.
func TestInstall_DryRunPreviewNamespace(t *testing.T) {
	tmpData := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpData)

	pkgDir := installTestPackage(t)

	stderr, err := captureStderr(t, func() error {
		return commands.Install(pkgDir, "preview", commands.Preview)
	})
	if err != nil {
		t.Fatalf("Install dryRun failed: %v", err)
	}
	if !strings.Contains(stderr, "WARN: installing directly to `preview`") {
		t.Errorf("expected preview namespace warning, got: %s", stderr)
	}
	if !strings.Contains(stderr, "Dry run: installation skipped, no files written") {
		t.Errorf("expected dry-run skipped message, got: %s", stderr)
	}
}
