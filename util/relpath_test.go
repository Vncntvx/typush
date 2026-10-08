package util_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Vncntvx/typush/util"
)

func TestRelEntries(t *testing.T) {
	root := t.TempDir()
	entries := []string{
		root,
		filepath.Join(root, "src"),
		filepath.Join(root, "src", "lib.typ"),
		filepath.Join(root, "typst.toml"),
	}
	got, err := util.RelEntries(entries, root)
	if err != nil {
		t.Fatalf("RelEntries failed: %v", err)
	}
	want := []util.RelEntry{
		{Abs: filepath.Join(root, "src"), Rel: "src"},
		{Abs: filepath.Join(root, "src", "lib.typ"), Rel: "src/lib.typ"},
		{Abs: filepath.Join(root, "typst.toml"), Rel: "typst.toml"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("RelEntries() = %+v, want %+v", got, want)
	}
}

// RelEntries resolves a relative root itself, so it still matches the absolute
// entries the walker returns.
func TestRelEntriesAcceptsRelativeRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "lib.typ"), []byte("// lib\n"), 0o644); err != nil {
		t.Fatalf("failed to write lib.typ: %v", err)
	}
	t.Chdir(root)

	entries, err := util.ListInstall(".", nil)
	if err != nil {
		t.Fatalf("ListInstall failed: %v", err)
	}
	if len(entries) == 0 || !filepath.IsAbs(entries[0]) {
		t.Fatalf("expected absolute entries, got %v", entries)
	}

	got, err := util.RelEntries(entries, ".")
	if err != nil {
		t.Fatalf("RelEntries failed: %v", err)
	}
	if len(got) != 1 || got[0].Rel != "lib.typ" {
		t.Fatalf("expected exactly lib.typ, got %+v", got)
	}
	if _, err := os.Stat(got[0].Abs); err != nil {
		t.Errorf("Abs should point at the walked file: %v", err)
	}
}
