package manifest_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vncntvx/typush/manifest"
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

func universeBase() map[string]string {
	return map[string]string{
		"name": `"demo"`, "version": `"0.1.0"`, "entrypoint": `"src/lib.typ"`,
		"authors": `["Jane <@jane-doe>"]`, "license": `"MIT"`,
		"description": `"Draw diagrams fast."`,
	}
}

func buildManifest(pkg map[string]string, template string) string {
	var sb strings.Builder
	sb.WriteString("[package]\n")
	keys := []string{"name", "version", "entrypoint", "authors", "license", "description", "homepage", "repository", "keywords", "categories", "disciplines", "compiler", "exclude", "foo"}
	for _, k := range keys {
		if v, ok := pkg[k]; ok {
			fmt.Fprintf(&sb, "%s = %s\n", k, v)
		}
	}
	sb.WriteString(template)
	return sb.String()
}

func TestValidateUniverseOK(t *testing.T) {
	dir := writeTOML(t, buildManifest(universeBase(), ""))
	m, err := manifest.Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.ValidateUniverse(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateUniverseRejects(t *testing.T) {
	cases := map[string]func(map[string]string){
		"unknown field":      func(p map[string]string) { p["foo"] = `1` },
		"bad author":         func(p map[string]string) { p["authors"] = `["X <@bad--handle>"]` },
		"too many cats":      func(p map[string]string) { p["categories"] = `["text", "layout", "fun", "cv"]` },
		"bad category":       func(p map[string]string) { p["categories"] = `["nope"]` },
		"bad discipline":     func(p map[string]string) { p["disciplines"] = `["nope"]` },
		"bad license":        func(p map[string]string) { p["license"] = `"All-Rights-Reserved"` },
		"license ref":        func(p map[string]string) { p["license"] = `"LicenseRef-Foo"` },
		"bang exclude":       func(p map[string]string) { p["exclude"] = `["!keep"]` },
		"missing license":    func(p map[string]string) { delete(p, "license") },
		"missing authors":    func(p map[string]string) { p["authors"] = `[]` },
		"missing descrip":    func(p map[string]string) { delete(p, "description") },
		"bad name":           func(p map[string]string) { p["name"] = `"9bad"` },
		"bad version":        func(p map[string]string) { p["version"] = `"x"` },
		"template no catego": func(p map[string]string) {},
	}
	templates := map[string]string{
		"template no catego": "[template]\npath = \"template\"\nentrypoint = \"main.typ\"\n",
	}
	for name, mutate := range cases {
		p := universeBase()
		mutate(p)
		tpl := templates[name]
		dir := writeTOML(t, buildManifest(p, tpl))
		m, err := manifest.Read(dir)
		if err != nil {
			continue // read-time rejection also counts as reject
		}
		if err := m.ValidateUniverse(); err == nil {
			t.Fatalf("%s: expected ValidateUniverse error", name)
		}
	}
}

func TestValidateLicense(t *testing.T) {
	for _, ok := range []string{"MIT", "Apache-2.0 OR MIT", "MIT AND (Apache-2.0 OR CC-BY-4.0)", "GPL-3.0-or-later", "CC-BY-SA-4.0", "MIT-0"} {
		if err := manifest.ValidateLicense(ok); err != nil {
			t.Fatalf("%s: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "MIT OR", "Proprietary", "LicenseRef-X", "(MIT"} {
		if err := manifest.ValidateLicense(bad); err == nil {
			t.Fatalf("%s: expected error", bad)
		}
	}
}

func TestValidateAuthor(t *testing.T) {
	for _, ok := range []string{"Martin", "Martin <@reknih>", "Martin <https://mha.ug>", "Martin <martin.haug@typst.app>"} {
		if err := manifest.ValidateAuthor(ok); err != nil {
			t.Fatalf("%s: %v", ok, err)
		}
	}
	for _, bad := range []string{"Martin <@reknih->", "Martin <@-reknih>", "Martin <@räknih>", "Martin <@reknih", "Martin <>", "Martin <martin>", "Martin <m@.de>", "Martin <http://mha ug>"} {
		if err := manifest.ValidateAuthor(bad); err == nil {
			t.Fatalf("%s: expected error", bad)
		}
	}
}

func TestIsIdent(t *testing.T) {
	for _, ok := range []string{"demo", "_x", "a-b_c9", "Ünïcode"} {
		if !manifest.IsIdent(ok) {
			t.Fatalf("%s should be ident", ok)
		}
	}
	for _, bad := range []string{"", "9a", "a b", "a/b", "a:b"} {
		if manifest.IsIdent(bad) {
			t.Fatalf("%s should not be ident", bad)
		}
	}
}
