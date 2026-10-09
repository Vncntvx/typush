package commands_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vncntvx/typush/commands"
)

func setupUninstallTestDirs(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("TYPST_PACKAGE_PATH", root)

	// @local/lib/0.1.0
	// @local/lib/0.2.0
	// @local/other/1.0.0
	// @custom/tool/0.5.0
	for _, p := range []string{
		filepath.Join(root, "local", "lib", "0.1.0"),
		filepath.Join(root, "local", "lib", "0.2.0"),
		filepath.Join(root, "local", "other", "1.0.0"),
		filepath.Join(root, "custom", "tool", "0.5.0"),
	} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p, "typst.toml"), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

func TestUninstallVersion(t *testing.T) {
	root := setupUninstallTestDirs(t)

	// Uninstall @local/lib:0.1.0 with force = true
	_, err := captureStderr(t, func() error {
		return commands.Uninstall(commands.UninstallOptions{
			Target: "@local/lib:0.1.0",
			Force:  true,
			DryRun: commands.Execute,
		})
	})
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	// 0.1.0 should be gone
	if _, err := os.Stat(filepath.Join(root, "local", "lib", "0.1.0")); !os.IsNotExist(err) {
		t.Errorf("expected 0.1.0 to be deleted")
	}

	// 0.2.0 and package dir should still exist
	if _, err := os.Stat(filepath.Join(root, "local", "lib", "0.2.0")); err != nil {
		t.Errorf("expected 0.2.0 to still exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "local", "lib")); err != nil {
		t.Errorf("expected package dir 'lib' to still exist: %v", err)
	}

	// Now uninstall 0.2.0, which should cascade delete the empty 'lib' dir
	_, err = captureStderr(t, func() error {
		return commands.Uninstall(commands.UninstallOptions{
			Target: "lib:0.2.0",
			Force:  true,
			DryRun: commands.Execute,
		})
	})
	if err != nil {
		t.Fatalf("Uninstall 0.2.0 failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "local", "lib")); !os.IsNotExist(err) {
		t.Errorf("expected empty 'lib' dir to be cascade cleaned")
	}

	// 'local' should still exist because 'other' is there
	if _, err := os.Stat(filepath.Join(root, "local")); err != nil {
		t.Errorf("expected 'local' namespace to still exist: %v", err)
	}
}

func TestUninstallPackage(t *testing.T) {
	root := setupUninstallTestDirs(t)

	// Uninstall all versions of 'lib' in @local
	_, err := captureStderr(t, func() error {
		return commands.Uninstall(commands.UninstallOptions{
			Target: "@local/lib",
			Force:  true,
			DryRun: commands.Execute,
		})
	})
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "local", "lib")); !os.IsNotExist(err) {
		t.Errorf("expected 'lib' package dir to be deleted")
	}
}

func TestUninstallNamespace(t *testing.T) {
	root := setupUninstallTestDirs(t)

	// Uninstall entire @custom namespace
	_, err := captureStderr(t, func() error {
		return commands.Uninstall(commands.UninstallOptions{
			Target: "@custom",
			Force:  true,
			DryRun: commands.Execute,
		})
	})
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "custom")); !os.IsNotExist(err) {
		t.Errorf("expected 'custom' namespace dir to be deleted")
	}
}

func TestUninstallDryRun(t *testing.T) {
	root := setupUninstallTestDirs(t)

	stderr, err := captureStderr(t, func() error {
		return commands.Uninstall(commands.UninstallOptions{
			Target: "@local/lib:0.1.0",
			DryRun: commands.Preview,
		})
	})
	if err != nil {
		t.Fatalf("Uninstall dry-run failed: %v", err)
	}

	if !strings.Contains(stderr, "Dry run: would remove @local/lib:0.1.0") {
		t.Errorf("expected dry-run note in stderr, got:\n%s", stderr)
	}

	// Verify file was NOT deleted
	if _, err := os.Stat(filepath.Join(root, "local", "lib", "0.1.0")); err != nil {
		t.Errorf("expected file to remain intact in dry-run mode: %v", err)
	}
}

func TestUninstallNotFound(t *testing.T) {
	setupUninstallTestDirs(t)

	err := commands.Uninstall(commands.UninstallOptions{
		Target: "@local/non-existent:1.0.0",
		Force:  true,
		DryRun: commands.Execute,
	})
	if err == nil {
		t.Fatal("expected error for non-existent package, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got %v", err)
	}
}

func TestUninstallPathTraversalRejected(t *testing.T) {
	setupUninstallTestDirs(t)

	err := commands.Uninstall(commands.UninstallOptions{
		Target: "../evil",
		Force:  true,
		DryRun: commands.Execute,
	})
	if err == nil {
		t.Fatal("expected error for path traversal attempt, got nil")
	}
	if !strings.Contains(err.Error(), "path traversal") {
		t.Errorf("expected 'path traversal' error, got %v", err)
	}
}
