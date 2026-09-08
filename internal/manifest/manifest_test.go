package manifest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vncntvx/typush-go/internal/manifest"
)

func writeTOML(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "typst.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestReadValid(t *testing.T) {
	dir := writeTOML(t, `[package]
name = "demo"
version = "0.1.0"
entrypoint = "src/lib.typ"
authors = ["me"]
`)
	m, err := manifest.Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Package.Name != "demo" || m.Package.Version != "0.1.0" {
		t.Fatalf("unexpected manifest: %+v", m.Package)
	}
}

func TestReadInvalid(t *testing.T) {
	for _, body := range []string{
		`[package]
name = "0bad"
version = "0.1.0"
entrypoint = "src/lib.typ"
authors = ["me"]
`,
		`[package]
name = "demo"
version = "notaversion"
entrypoint = "src/lib.typ"
authors = ["me"]
`,
		`[package]
name = "demo"
version = "0.1.0"
entrypoint = "src/main.pdf"
authors = ["me"]
`,
	} {
		if _, err := manifest.Read(writeTOML(t, body)); err == nil {
			t.Fatalf("expected error for %q", body)
		}
	}
}

func TestRoundTripToolPreserved(t *testing.T) {
	dir := writeTOML(t, `[package]
name = "demo"
version = "0.1.0"
entrypoint = "src/lib.typ"
authors = ["me"]

[tool.typush]
hello = "world"
`)
	m, err := manifest.Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := manifest.Write(dir, m); err != nil {
		t.Fatal(err)
	}
	m2, err := manifest.Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m2.Tool == nil {
		t.Fatalf("expected tool table preserved, got nil")
	}
}

func TestCompareVersions(t *testing.T) {
	if manifest.CompareVersions("0.2.0", "0.1.0") != 1 {
		t.Fatal("0.2.0 > 0.1.0")
	}
	if manifest.CompareVersions("0.1.0", "0.1.0") != 0 {
		t.Fatal("equal")
	}
	if manifest.CompareVersions("0.1.0", "0.2.0") != -1 {
		t.Fatal("0.1.0 < 0.2.0")
	}
}
