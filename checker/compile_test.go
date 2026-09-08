package checker_test

import (
	"os"
	"path/filepath"
	"testing"

	checker "github.com/Vncntvx/typkg/checker"
)

func TestLookPath(t *testing.T) {
	if _, ok := checker.LookPath(); !ok {
		t.Skip("typst binary not found, skipping compile tests")
	}
}

func writePkg(t *testing.T, withTemplate bool) string {
	t.Helper()
	dir := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("typst.toml", "[package]\nname = \"smoke-pkg\"\nversion = \"0.1.0\"\nentrypoint = \"src/lib.typ\"\nauthors = [\"t\"]\n")
	write("src/lib.typ", "#let greet(name) = [Hello, #name!]\n")
	if withTemplate {
		f, err := os.OpenFile(filepath.Join(dir, "typst.toml"), os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.WriteString("\n[template]\npath = \"template\"\nentrypoint = \"main.typ\"\n"); err != nil {
			t.Fatal(err)
		}
		f.Close()
		write("template/main.typ", "#import \"@preview/smoke-pkg:0.1.0\": *\n#greet(\"world\")\n")
	}
	return dir
}

func TestCheckLibraryOK(t *testing.T) {
	bin, ok := checker.LookPath()
	if !ok {
		t.Skip("typst binary not found")
	}
	res := checker.CheckLibrary(bin, writePkg(t, false), "smoke-pkg", "0.1.0")
	if len(res.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", res.Errors)
	}
}

func TestCheckLibraryFails(t *testing.T) {
	bin, ok := checker.LookPath()
	if !ok {
		t.Skip("typst binary not found")
	}
	dir := writePkg(t, false)
	if err := os.WriteFile(filepath.Join(dir, "src", "lib.typ"), []byte("#let = broken ((("), 0o644); err != nil {
		t.Fatal(err)
	}
	res := checker.CheckLibrary(bin, dir, "smoke-pkg", "0.1.0")
	if len(res.Errors) == 0 {
		t.Fatal("expected compile errors for broken lib.typ")
	}
}

func TestCheckTemplateOK(t *testing.T) {
	bin, ok := checker.LookPath()
	if !ok {
		t.Skip("typst binary not found")
	}
	res := checker.CheckTemplate(bin, writePkg(t, true), "smoke-pkg", "0.1.0", "main.typ")
	if len(res.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", res.Errors)
	}
}
