package checker_test

import (
	"strings"
	"testing"

	"github.com/Vncntvx/typkg/checker"

	"github.com/Vncntvx/typkg/manifest"
)

func testManifest() *manifest.Manifest {
	return &manifest.Manifest{
		Package: manifest.PackageInfo{
			Name: "demo", Version: "0.2.0", Entrypoint: "src/lib.typ",
		},
	}
}

func codes(ds []checker.Diag) []string {
	var out []string
	for _, d := range ds {
		out = append(out, string(d.Severity)+":"+d.Code)
	}
	return out
}

func has(ds []checker.Diag, want string) bool {
	for _, c := range codes(ds) {
		if c == want {
			return true
		}
	}
	return false
}

func TestReadme(t *testing.T) {
	text := "# Demo\n\n![](docs/a.png)\n\n![img](docs/b.png)\n\n[manual](docs/manual.pdf)\n\n> [!NOTE]\n> hi\n\n- [ ] todo\n\n[branch](https://github.com/u/r/blob/main/f.pdf)\n"
	// create linked files
	dir := t.TempDir()
	mustWrite := func(rel string) {
		p := dir + "/" + rel
		if err := mkfile(p); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("docs/a.png")
	mustWrite("docs/b.png")
	mustWrite("docs/manual.pdf")

	diags, linked := checker.Readme(dir, text)
	if !has(diags, "error:readme/image/missing-alt") {
		t.Fatalf("expected missing-alt error, got %v", codes(diags))
	}
	if !has(diags, "warning:readme/image/inadequate-alt") {
		t.Fatalf("expected inadequate-alt warning, got %v", codes(diags))
	}
	if !has(diags, "warning:readme/unsupported-extension/alert") {
		t.Fatalf("expected alert warning, got %v", codes(diags))
	}
	if !has(diags, "warning:readme/unsupported-extension/tasklist") {
		t.Fatalf("expected tasklist warning, got %v", codes(diags))
	}
	if !has(diags, "warning:readme/link/repository-url-permalink") {
		t.Fatalf("expected permalink warning, got %v", codes(diags))
	}
	if len(linked) != 3 {
		t.Fatalf("expected 3 linked files, got %v", linked)
	}

	// broken local link
	diags2, _ := checker.Readme(dir, "[x](docs/missing.pdf)")
	if !has(diags2, "error:readme/link/file-not-found") {
		t.Fatalf("expected file-not-found, got %v", codes(diags2))
	}
	// good alt: no findings
	diags3, _ := checker.Readme(dir, "![A bar chart of sales](docs/a.png)")
	for _, d := range diags3 {
		if strings.HasPrefix(d.Code, "readme/image") {
			t.Fatalf("unexpected image diag: %+v", d)
		}
	}
}

func TestFiles(t *testing.T) {
	all := []checker.FileEntry{
		{RelSlash: "src/lib.typ", Size: 100},
		{RelSlash: "fonts/a.ttf", Size: 100},
		{RelSlash: "examples/my-example.typ", Size: 100},
		{RelSlash: "tests/foo-test.typ", Size: 100},
		{RelSlash: "big.bin", Size: 2 * 1024 * 1024},
		{RelSlash: "huge.bin", Size: 60 * 1024 * 1024},
		{RelSlash: "docs/manual.pdf", Size: 100},
		{RelSlash: ".hidden", Size: 10},
	}
	bundled := map[string]bool{"src/lib.typ": true, "fonts/a.ttf": true, "examples/x.typ": true,
		"tests/y.typ": true, "big.bin": true, "huge.bin": true, "docs/manual.pdf": true}
	diags := checker.Files(all, bundled, map[string]bool{}, []string{"docs/manual.pdf"})
	if !has(diags, "error:files/fonts") {
		t.Fatalf("expected fonts error, got %v", codes(diags))
	}
	if !has(diags, "warning:exclude/example") || !has(diags, "warning:exclude/test") {
		t.Fatalf("expected example/test warnings, got %v", codes(diags))
	}
	if !has(diags, "warning:size/large") || !has(diags, "warning:size/extra-large") {
		t.Fatalf("expected size warnings, got %v", codes(diags))
	}
	if has(diags, "warning:files/manual/unlinked") {
		t.Fatalf("manual is linked, should not warn: %v", codes(diags))
	}
	diags2 := checker.Files(all, bundled, map[string]bool{}, nil)
	if !has(diags2, "warning:files/manual/unlinked") {
		t.Fatalf("expected unlinked manual warning, got %v", codes(diags2))
	}
	// ignored file warning
	diags3 := checker.Files(
		[]checker.FileEntry{{RelSlash: "scratch.txt", Size: 10}},
		map[string]bool{}, map[string]bool{}, nil)
	if !has(diags3, "warning:files/ignored") {
		t.Fatalf("expected ignored warning, got %v", codes(diags3))
	}
}

func TestImports(t *testing.T) {
	m := testManifest()
	srcs := map[string]string{
		"template/main.typ": "#import \"../src/lib.typ\"\n#import \"@preview/demo:0.1.0\": *",
		"docs/example.typ":  "#import \"@preview/demo:0.2.0\": *",
		"src/other.typ":     "#import \"lib.typ\"",
		"src/lib.typ":       "#let x = 1",
		"future.typ":        "#import \"@preview/demo:0.3.0\": *",
		"other-pkg.typ":     "#import \"@preview/other:0.1.0\": *",
	}
	diags := checker.Imports("/pkg", m, srcs)
	if !has(diags, "warning:import/relative") {
		t.Fatalf("expected relative warning, got %v", codes(diags))
	}
	if !has(diags, "error:import/outdated") {
		t.Fatalf("expected outdated error, got %v", codes(diags))
	}
	if !has(diags, "warning:import/outdated") {
		t.Fatalf("expected newer warning, got %v", codes(diags))
	}
}
