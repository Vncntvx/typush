package walker_test

import (
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/Vncntvx/typkg/internal/walker"
)

func testRoot() string {
	_, f, _, _ := runtime.Caller(0)
	// internal/walker/walker_test.go -> ../../testdata/walker_test
	return filepath.Join(filepath.Dir(f), "..", "..", "testdata", "walker_test")
}

func rels(t *testing.T, abs []string) []string {
	t.Helper()
	root := testRoot()
	absRoot, _ := filepath.Abs(root)
	var out []string
	for _, a := range abs {
		rel, err := filepath.Rel(absRoot, a)
		if err != nil {
			t.Fatal(err)
		}
		if rel == "." {
			continue // skip root itself for comparison
		}
		out = append(out, filepath.ToSlash(rel))
	}
	sort.Strings(out)
	return out
}

func equalStr(t *testing.T, got, want []string) {
	t.Helper()
	sort.Strings(got)
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestPublish(t *testing.T) {
	got, err := walker.ListPublish(testRoot())
	if err != nil {
		t.Fatal(err)
	}
	// .typstignore removes ignore_test.txt; .typstignore itself never listed
	equalStr(t, rels(t, got), []string{"excludes_test.txt", "src", "src/lib.typ", "typst.toml"})
}

func TestInstall(t *testing.T) {
	got, err := walker.ListInstall(testRoot(), []string{"excludes_test.txt"})
	if err != nil {
		t.Fatal(err)
	}
	equalStr(t, rels(t, got), []string{"src", "src/lib.typ", "typst.toml"})
}
