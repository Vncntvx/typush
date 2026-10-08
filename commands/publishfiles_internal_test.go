package commands

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// writeFile creates a file (and its parents) inside dir.
func writeFile(t *testing.T, dir, rel string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("failed to create dir for %s: %v", rel, err)
	}
	if err := os.WriteFile(p, []byte("// "+rel+"\n"), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", rel, err)
	}
}

func TestPublishableFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "typst.toml")
	writeFile(t, dir, "README.md")
	writeFile(t, dir, "src/lib.typ")
	writeFile(t, dir, "examples/demo.typ")
	writeFile(t, dir, ".gitkeep") // dotfiles are never returned
	// .gitignore is read for patterns but never published itself.
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("examples/\n"), 0o644); err != nil {
		t.Fatalf("failed to write .gitignore: %v", err)
	}

	got, err := publishableFiles(dir)
	if err != nil {
		t.Fatalf("publishableFiles failed: %v", err)
	}
	want := []string{"README.md", "src/lib.typ", "typst.toml"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("publishableFiles() = %v, want %v", got, want)
	}
}

func TestPublishableFilesKeepsPackageExcludedFiles(t *testing.T) {
	dir := t.TempDir()
	manifest := `[package]
name = "demo-pkg"
version = "1.2.3"
entrypoint = "lib.typ"
exclude = ["tests/**"]
`
	if err := os.WriteFile(filepath.Join(dir, "typst.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "lib.typ")
	writeFile(t, dir, "tests/example.typ")

	got, err := publishableFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"lib.typ", "tests/example.typ", "typst.toml"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("publishableFiles() = %v, want %v; package.exclude must not filter uploads", got, want)
	}
}

// Directories and ignore files never belong in the upload list.
func TestPublishableFilesListsFilesOnly(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "lib.typ")
	writeFile(t, dir, "docs/notes.txt")
	writeFile(t, dir, ".gitignore")

	got, err := publishableFiles(dir)
	if err != nil {
		t.Fatalf("publishableFiles failed: %v", err)
	}
	want := []string{"docs/notes.txt", "lib.typ"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("publishableFiles() = %v, want %v", got, want)
	}
}
