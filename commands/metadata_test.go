package commands_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vncntvx/typush/commands"
)

func TestMetadata_Overview(t *testing.T) {
	dir := t.TempDir()
	writeTestPackage(t, dir)

	out, err := captureStdout(t, func() error {
		return commands.Metadata(dir, "", false)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Name:        demo-pkg") {
		t.Errorf("expected overview to contain Name, got:\n%s", out)
	}
	if !strings.Contains(out, "Version:     1.2.3") {
		t.Errorf("expected overview to contain Version, got:\n%s", out)
	}
	if !strings.Contains(out, "Authors:     Alice, Bob") {
		t.Errorf("expected overview to contain Authors, got:\n%s", out)
	}
}

func TestMetadata_OverviewWithTemplate(t *testing.T) {
	dir := t.TempDir()
	tomlContent := `[package]
name = "template-pkg"
version = "0.1.0"
entrypoint = "lib.typ"
authors = ["Author"]

[template]
path = "template"
entrypoint = "main.typ"
thumbnail = "thumbnail.png"
`
	if err := os.WriteFile(filepath.Join(dir, "typst.toml"), []byte(tomlContent), 0o644); err != nil {
		t.Fatalf("failed to write typst.toml: %v", err)
	}

	out, err := captureStdout(t, func() error {
		return commands.Metadata(dir, "", false)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Template Path:   template") {
		t.Errorf("expected Template Path in overview, got:\n%s", out)
	}
	if !strings.Contains(out, "Template Entry:  main.typ") {
		t.Errorf("expected Template Entry in overview, got:\n%s", out)
	}
	if !strings.Contains(out, "Thumbnail:       thumbnail.png") {
		t.Errorf("expected Thumbnail in overview, got:\n%s", out)
	}
}

func TestMetadata_JSON(t *testing.T) {
	dir := t.TempDir()
	writeTestPackage(t, dir)

	out, err := captureStdout(t, func() error {
		return commands.Metadata(dir, "", true)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v\nOutput was: %s", err, out)
	}

	pkg, ok := parsed["package"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'package' object in JSON, got: %v", parsed)
	}
	if pkg["name"] != "demo-pkg" {
		t.Errorf("expected name 'demo-pkg', got %v", pkg["name"])
	}
	if pkg["version"] != "1.2.3" {
		t.Errorf("expected version '1.2.3', got %v", pkg["version"])
	}
}

func TestMetadata_Fields(t *testing.T) {
	dir := t.TempDir()
	writeTestPackage(t, dir)

	cases := []struct {
		field    string
		asJSON   bool
		expected string
	}{
		{"name", false, "demo-pkg\n"},
		{"version", false, "1.2.3\n"},
		{"entrypoint", false, "lib.typ\n"},
		{"license", false, "MIT\n"},
		{"repository", false, "https://github.com/example/demo-pkg\n"},
		{"authors", false, "Alice, Bob\n"},
		{"authors", true, "[\"Alice\",\"Bob\"]\n"},
		{"categories", false, "utility\n"},
		{"disciplines", false, "computer-science\n"},
		{"keywords", false, "test, demo\n"},
		{"compiler", false, "0.12.0\n"},
	}

	for _, tc := range cases {
		t.Run(tc.field, func(t *testing.T) {
			out, err := captureStdout(t, func() error {
				return commands.Metadata(dir, tc.field, tc.asJSON)
			})
			if err != nil {
				t.Fatalf("unexpected error for field %s: %v", tc.field, err)
			}
			if out != tc.expected {
				t.Errorf("field %s (json=%v): expected %q, got %q", tc.field, tc.asJSON, tc.expected, out)
			}
		})
	}
}

func TestMetadata_UnsetFields(t *testing.T) {
	dir := t.TempDir()
	tomlContent := `[package]
name = "minimal-pkg"
version = "0.1.0"
entrypoint = "lib.typ"
authors = ["Tester"]
`
	if err := os.WriteFile(filepath.Join(dir, "typst.toml"), []byte(tomlContent), 0o644); err != nil {
		t.Fatalf("failed to write typst.toml: %v", err)
	}

	// Plain text should output empty line
	outText, err := captureStdout(t, func() error {
		return commands.Metadata(dir, "license", false)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if outText != "\n" {
		t.Errorf("expected empty newline for unset field, got %q", outText)
	}

	// JSON should output null
	outJSON, err := captureStdout(t, func() error {
		return commands.Metadata(dir, "license", true)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if outJSON != "null\n" {
		t.Errorf("expected \"null\\n\" for unset field in JSON mode, got %q", outJSON)
	}
}

func TestMetadata_InvalidField(t *testing.T) {
	dir := t.TempDir()
	writeTestPackage(t, dir)

	_, err := captureStdout(t, func() error {
		return commands.Metadata(dir, "nonexistent", false)
	})
	if err == nil {
		t.Fatal("expected error on unknown field, got nil")
	}
	if !strings.Contains(err.Error(), "unknown metadata field") {
		t.Errorf("unexpected error message: %v", err)
	}
}
