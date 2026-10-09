package commands_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vncntvx/typush/commands"
)

const updateIndex = `[
	{"name": "cetz", "version": "0.2.2"},
	{"name": "fletcher", "version": "0.5.1"},
	{"name": "touying", "version": "0.6.1"}
]`

func TestUpdate(t *testing.T) {
	useUniverse(t, updateIndex)

	dir := t.TempDir()
	docPath := filepath.Join(dir, "main.typ")
	initial := "#import \"@preview/cetz:0.2.0\": *\n#import \"@preview/fletcher:0.5.0\": *\n"
	if err := os.WriteFile(docPath, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	stderr, err := captureStderr(t, func() error {
		return commands.Update(commands.UpdateOptions{Dir: dir, DryRun: true})
	})
	if err != nil {
		t.Fatalf("dry run failed: %v", err)
	}
	if !strings.Contains(stderr, "Dry run: dependency updates (2 item(s)):") {
		t.Errorf("expected a counted preview item block, got:\n%s", stderr)
	}
	if !strings.Contains(stderr, "- #import \"@preview/cetz:0.2.0\": *") ||
		!strings.Contains(stderr, "+ #import \"@preview/cetz:0.2.2\": *") {
		t.Errorf("expected a line diff in the preview, got:\n%s", stderr)
	}

	// A dry run must not touch the file.
	after, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != initial {
		t.Errorf("dry run modified the file:\n%s", after)
	}

	// The real run must apply exactly the previewed changes.
	stderr, err = captureStderr(t, func() error {
		return commands.Update(commands.UpdateOptions{Dir: dir, Refresh: true})
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if !strings.Contains(stderr, "Updated 2 occurrence(s) across 1 file(s).") {
		t.Errorf("expected update summary, got:\n%s", stderr)
	}

	after, err = os.ReadFile(docPath)
	if err != nil {
		t.Fatal(err)
	}
	want := "#import \"@preview/cetz:0.2.2\": *\n#import \"@preview/fletcher:0.5.1\": *\n"
	if string(after) != want {
		t.Errorf("update result:\n%s\nwant:\n%s", after, want)
	}
}

// TestUpdate_PreviewMatchesWrite verifies that the dry run matches the actual write line by line.
func TestUpdate_PreviewMatchesWrite(t *testing.T) {
	useUniverse(t, updateIndex)

	dir := t.TempDir()
	docPath := filepath.Join(dir, "main.typ")
	initial := "#import \"@preview/cetz:0.1.0\": *\r\n#import '@preview/fletcher:0.5.0': diagram\r\n"
	if err := os.WriteFile(docPath, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	// Capture the "+ new" lines the preview promises.
	stderr, err := captureStderr(t, func() error {
		return commands.Update(commands.UpdateOptions{Dir: dir, DryRun: true})
	})
	if err != nil {
		t.Fatalf("dry run failed: %v", err)
	}
	var promised []string
	for _, line := range strings.Split(stderr, "\n") {
		if trimmed, ok := strings.CutPrefix(strings.TrimSpace(line), "+ "); ok {
			promised = append(promised, trimmed)
		}
	}
	if len(promised) != 2 {
		t.Fatalf("expected 2 previewed lines, got %d:\n%s", len(promised), stderr)
	}

	if _, err := captureStderr(t, func() error {
		return commands.Update(commands.UpdateOptions{Dir: dir, Refresh: true})
	}); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	after, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatal(err)
	}
	// CRLF line endings must survive, since only the version string changes.
	if !strings.Contains(string(after), "\r\n") {
		t.Errorf("CRLF line endings were not preserved:\n%q", after)
	}
	for _, line := range promised {
		if !strings.Contains(string(after), line) {
			t.Errorf("preview promised %q but the file does not contain it:\n%s", line, after)
		}
	}
}

// TestUpdate_FetchesIndexOnce verifies that the index is downloaded only once per command.
func TestUpdate_FetchesIndexOnce(t *testing.T) {
	hits := useUniverse(t, `[
		{"name": "pkg-a", "version": "2.0.0"},
		{"name": "pkg-b", "version": "2.0.0"},
		{"name": "pkg-c", "version": "2.0.0"},
		{"name": "pkg-d", "version": "2.0.0"},
		{"name": "pkg-e", "version": "2.0.0"}
	]`)

	dir := t.TempDir()
	var b strings.Builder
	for _, name := range []string{"pkg-a", "pkg-b", "pkg-c", "pkg-d", "pkg-e"} {
		b.WriteString("#import \"@preview/" + name + ":1.0.0\": *\n")
	}
	if err := os.WriteFile(filepath.Join(dir, "main.typ"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := captureStderr(t, func() error {
		return commands.Update(commands.UpdateOptions{Dir: dir, Refresh: true})
	}); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	if *hits != 1 {
		t.Errorf("expected the index to be fetched once for 5 dependencies, got %d requests", *hits)
	}
}

// The same guarantee holds for the read-only path.
func TestOutdated_FetchesIndexOnce(t *testing.T) {
	hits := useUniverse(t, `[
		{"name": "pkg-a", "version": "2.0.0"},
		{"name": "pkg-b", "version": "2.0.0"},
		{"name": "pkg-c", "version": "2.0.0"}
	]`)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.typ"), []byte(
		"#import \"@preview/pkg-a:1.0.0\": *\n#import \"@preview/pkg-b:1.0.0\": *\n#import \"@preview/pkg-c:1.0.0\": *\n"),
		0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := captureStdout(t, func() error {
		return commands.Outdated(dir, commands.UniverseOptions{Refresh: true})
	}); err != nil {
		t.Fatalf("outdated failed: %v", err)
	}

	if *hits != 1 {
		t.Errorf("expected the index to be fetched once for 3 dependencies, got %d requests", *hits)
	}
}

// A single file may hold many packages; it must be read once and rewritten once.
func TestUpdate_MultiplePackagesOneFile(t *testing.T) {
	useUniverse(t, updateIndex)

	dir := t.TempDir()
	docPath := filepath.Join(dir, "main.typ")
	if err := os.WriteFile(docPath, []byte(
		"#import \"@preview/cetz:0.2.0\": *\n#import \"@preview/fletcher:0.1.0\": *\n#import \"@preview/touying:0.1.0\": *\n"),
		0o644); err != nil {
		t.Fatal(err)
	}

	stderr, err := captureStderr(t, func() error {
		return commands.Update(commands.UpdateOptions{Dir: dir, Refresh: true})
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if !strings.Contains(stderr, "Updated 3 occurrence(s) across 1 file(s).") {
		t.Errorf("expected 3 updates in 1 file, got:\n%s", stderr)
	}

	after, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(after), "0.2.0") || strings.Contains(string(after), "0.1.0") {
		t.Errorf("stale versions remain:\n%s", after)
	}
}

// The reported count is occurrences, not lines: one line can carry two specs.
func TestUpdate_CountsOccurrencesNotLines(t *testing.T) {
	useUniverse(t, updateIndex)

	dir := t.TempDir()
	docPath := filepath.Join(dir, "main.typ")
	if err := os.WriteFile(docPath, []byte(
		"#import \"@preview/cetz:0.1.0\": * #import \"@preview/cetz:0.2.0\": *\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stderr, err := captureStderr(t, func() error {
		return commands.Update(commands.UpdateOptions{Dir: dir, Refresh: true})
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if !strings.Contains(stderr, "Updated 2 occurrence(s) across 1 file(s).") {
		t.Errorf("expected both specs on the line to be counted, got:\n%s", stderr)
	}

	after, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatal(err)
	}
	if want := "#import \"@preview/cetz:0.2.2\": * #import \"@preview/cetz:0.2.2\": *\n"; string(after) != want {
		t.Errorf("rewrite result:\n got: %q\nwant: %q", after, want)
	}
}

// An import nested inside a longer line must keep its quotes and surroundings.
func TestUpdate_PreservesFormatting(t *testing.T) {
	useUniverse(t, updateIndex)

	dir := t.TempDir()
	docPath := filepath.Join(dir, "main.typ")
	initial := "#let c = (import \"@preview/cetz:0.2.0\": canvas).canvas\n#import '@preview/fletcher:0.5.0': diagram, node"
	if err := os.WriteFile(docPath, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := captureStderr(t, func() error {
		return commands.Update(commands.UpdateOptions{Dir: dir, Refresh: true})
	}); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	after, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatal(err)
	}
	// Preserve missing trailing newline.
	want := "#let c = (import \"@preview/cetz:0.2.2\": canvas).canvas\n#import '@preview/fletcher:0.5.1': diagram, node"
	if string(after) != want {
		t.Errorf("formatting not preserved:\n got: %q\nwant: %q", after, want)
	}
}

// TestUpdate_PackageFilter verifies that only selected packages are updated.
func TestUpdate_PackageFilter(t *testing.T) {
	useUniverse(t, updateIndex)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.typ"), []byte(
		"#import \"@preview/cetz:0.2.0\": *\n#import \"@preview/fletcher:0.5.0\": *\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stderr, err := captureStderr(t, func() error {
		return commands.Update(commands.UpdateOptions{Dir: dir, Packages: []string{"cetz"}, Refresh: true})
	})
	if err != nil {
		t.Fatalf("filtered update failed: %v", err)
	}
	if !strings.Contains(stderr, "Updated 1 occurrence(s) across 1 file(s).") {
		t.Errorf("expected only cetz to be updated, got:\n%s", stderr)
	}

	after, err := os.ReadFile(filepath.Join(dir, "main.typ"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "cetz:0.2.2") {
		t.Errorf("cetz was not updated:\n%s", after)
	}
	if !strings.Contains(string(after), "fletcher:0.5.0") {
		t.Errorf("fletcher should have been left alone:\n%s", after)
	}
}

// TestUpdate_UnknownPackageIsReported verifies that unknown requested packages are reported.
func TestUpdate_UnknownPackageIsReported(t *testing.T) {
	useUniverse(t, updateIndex)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.typ"), []byte("#import \"@preview/cetz:0.2.0\": *\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stderr, err := captureStderr(t, func() error {
		return commands.Update(commands.UpdateOptions{Dir: dir, Packages: []string{"nosuchpkg"}, Refresh: true})
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if !strings.Contains(stderr, `"nosuchpkg"`) {
		t.Errorf("expected a warning about the unknown package, got:\n%s", stderr)
	}
}

// A missing target and a non-.typ target must fail clearly.
func TestUpdate_TargetErrors(t *testing.T) {
	useUniverse(t, updateIndex)

	if err := commands.Update(commands.UpdateOptions{File: filepath.Join(t.TempDir(), "missing.typ")}); err == nil {
		t.Error("expected an error for a missing target file")
	}

	txt := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(txt, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := commands.Update(commands.UpdateOptions{File: txt}); err == nil {
		t.Error("expected an error for a non-.typ target file")
	}
}

// Files the project ignores must not be scanned or rewritten.
func TestUpdate_RespectsGitignore(t *testing.T) {
	useUniverse(t, updateIndex)

	dir := t.TempDir()
	generated := filepath.Join(dir, "generated")
	if err := os.MkdirAll(generated, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("generated/\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tracked := filepath.Join(dir, "main.typ")
	if err := os.WriteFile(tracked, []byte("#import \"@preview/cetz:0.2.0\": *\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ignored := filepath.Join(generated, "out.typ")
	if err := os.WriteFile(ignored, []byte("#import \"@preview/cetz:0.2.0\": *\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := captureStderr(t, func() error {
		return commands.Update(commands.UpdateOptions{Dir: dir, Refresh: true})
	}); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	after, err := os.ReadFile(ignored)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != "#import \"@preview/cetz:0.2.0\": *\n" {
		t.Errorf("a gitignored file was rewritten:\n%s", after)
	}

	// The tracked file must still have been updated.
	trackedAfter, err := os.ReadFile(tracked)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(trackedAfter), "cetz:0.2.2") {
		t.Errorf("the tracked file was not updated:\n%s", trackedAfter)
	}
}

// TestUpdate_LongLine verifies that lines longer than 64 KiB do not abort the scan.
func TestUpdate_LongLine(t *testing.T) {
	useUniverse(t, updateIndex)

	dir := t.TempDir()
	docPath := filepath.Join(dir, "long.typ")
	long := "#let pad = \"" + strings.Repeat("x", 200_000) + "\""
	if err := os.WriteFile(docPath, []byte(long+"\n#import \"@preview/cetz:0.2.0\": *\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stderr, err := captureStderr(t, func() error {
		return commands.Update(commands.UpdateOptions{Dir: dir, Refresh: true})
	})
	if err != nil {
		t.Fatalf("update failed on a long line: %v", err)
	}
	if !strings.Contains(stderr, "Updated 1 occurrence(s) across 1 file(s).") {
		t.Errorf("expected the import after the long line to be updated, got:\n%s", stderr)
	}
}

// TestUpdate_NothingToDo verifies that having nothing to update succeeds.
func TestUpdate_NothingToDo(t *testing.T) {
	useUniverse(t, updateIndex)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.typ"), []byte("#import \"@preview/cetz:0.2.2\": *\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stderr, err := captureStderr(t, func() error {
		return commands.Update(commands.UpdateOptions{Dir: dir, DryRun: true})
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if !strings.Contains(stderr, "already up to date") {
		t.Errorf("expected an up-to-date notice, got:\n%s", stderr)
	}
}

// TestUpdate_FailsWhenIndexUnavailable verifies that Update fails when the index cannot be loaded.
func TestUpdate_FailsWhenIndexUnavailable(t *testing.T) {
	useFailingUniverse(t)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.typ"), []byte("#import \"@preview/cetz:0.1.0\": *\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stderr, err := captureStderr(t, func() error {
		return commands.Update(commands.UpdateOptions{Dir: dir, Refresh: true})
	})
	if err == nil {
		t.Fatalf("expected an error when the index is unavailable, got:\n%s", stderr)
	}
	if strings.Contains(stderr, "already up to date") {
		t.Errorf("must not claim everything is current without an index:\n%s", stderr)
	}
}
