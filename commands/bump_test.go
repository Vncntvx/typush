package commands_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vncntvx/typush/commands"
	"github.com/Vncntvx/typush/manifest"
)

func TestBump_DoesNotTouchReadme(t *testing.T) {
	dir := t.TempDir()
	tomlContent := `[package]
name = "my-pkg"
version = "0.1.0"
entrypoint = "lib.typ"
authors = ["Test Author"]
`
	if err := os.WriteFile(filepath.Join(dir, "typst.toml"), []byte(tomlContent), 0o644); err != nil {
		t.Fatalf("failed to write typst.toml: %v", err)
	}

	readmeContent := `# my-pkg

Import via:
` + "```typst" + `
#import "@preview/my-pkg:0.1.0": *
` + "```\n"

	readmePath := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0o644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}

	if err := commands.Bump(dir, "patch"); err != nil {
		t.Fatalf("Bump failed: %v", err)
	}

	// Verify typst.toml was updated to 0.1.1
	m, err := manifest.Read(dir)
	if err != nil {
		t.Fatalf("failed to read updated manifest: %v", err)
	}
	if m.Package.Version != "0.1.1" {
		t.Errorf("expected typst.toml version '0.1.1', got %q", m.Package.Version)
	}

	// Verify README.md was NOT modified
	readmeAfter, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}
	if string(readmeAfter) != readmeContent {
		t.Errorf("README.md should not have been modified!\nExpected:\n%s\nGot:\n%s", readmeContent, string(readmeAfter))
	}
	if !strings.Contains(string(readmeAfter), "@preview/my-pkg:0.1.0") {
		t.Errorf("expected README to retain original 0.1.0 reference, but it changed")
	}
}
