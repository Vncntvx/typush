package commands_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vncntvx/typush/commands"
	"github.com/Vncntvx/typush/manifest"
)

// writeBumpPackage creates a minimal package at version 0.1.0.
func writeBumpPackage(t *testing.T) string {
	t.Helper()
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
	return dir
}

func TestBump_DoesNotTouchReadme(t *testing.T) {
	dir := writeBumpPackage(t)

	readmeContent := `# my-pkg

Import via:
` + "```typst" + `
#import "@preview/my-pkg:0.1.0": *
` + "```\n"

	readmePath := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0o644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}

	if err := commands.Bump(dir, "patch", commands.Execute); err != nil {
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

func TestBump_DryRun(t *testing.T) {
	dir := writeBumpPackage(t)

	// 1. Dry run with explicit target
	stderr, err := captureStderr(t, func() error {
		return commands.Bump(dir, "minor", commands.Preview)
	})
	if err != nil {
		t.Fatalf("Bump dry-run failed: %v", err)
	}
	if !strings.Contains(stderr, "Dry run: would bump version from 0.1.0 to 0.2.0 in typst.toml") {
		t.Errorf("expected dry-run preview message, got: %s", stderr)
	}

	// Verify typst.toml was NOT updated
	m, err := manifest.Read(dir)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}
	if m.Package.Version != "0.1.0" {
		t.Errorf("typst.toml should not be updated in dry-run mode, got %q", m.Package.Version)
	}

	// 2. Dry run with default (empty target defaults to patch)
	stderr, err = captureStderr(t, func() error {
		return commands.Bump(dir, "", commands.Preview)
	})
	if err != nil {
		t.Fatalf("Bump dry-run without target failed: %v", err)
	}
	if !strings.Contains(stderr, "Dry run: would bump version from 0.1.0 to 0.1.1 in typst.toml") {
		t.Errorf("expected default patch preview message, got: %s", stderr)
	}
}
