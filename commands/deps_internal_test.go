package commands

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vncntvx/typush/util"
)

// isolatedIndex points the index URL at a server serving body and isolates the
// cache directory, so these internal tests never touch the user's real cache.
func isolatedIndex(t *testing.T, body string) *util.UniverseIndex {
	t.Helper()

	UseUniverseServerForTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})

	idx, err := util.LoadUniverseIndex(true)
	if err != nil {
		t.Fatalf("failed to load test index: %v", err)
	}
	return idx
}

func TestScanDependencies(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "main.typ")
	if err := os.WriteFile(main, []byte(`#import "@preview/cetz:0.2.0": *
#import '@preview/fletcher:0.5.0': diagram, node
#let x = 1
#import "@preview/cetz:0.2.0": canvas
`), 0o644); err != nil {
		t.Fatal(err)
	}

	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "chap.typ"), []byte(
		`// Chapter 1
#import "@preview/touying:0.6.0": *
`), 0o644); err != nil {
		t.Fatal(err)
	}
	// Non-.typ files and nested packages must be ignored.
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte(`@preview/ignored:1.0.0`), 0o644); err != nil {
		t.Fatal(err)
	}

	occs, err := scanDependencies(dir)
	if err != nil {
		t.Fatalf("scanDependencies failed: %v", err)
	}
	if len(occs) != 4 {
		t.Fatalf("expected 4 occurrences, got %d: %+v", len(occs), occs)
	}

	// Paths must be absolute so display never depends on the working directory.
	for _, occ := range occs {
		if !filepath.IsAbs(occ.FilePath) {
			t.Errorf("occurrence path is not absolute: %q", occ.FilePath)
		}
	}

	if occs[0].Package != "cetz" || occs[0].Version != "0.2.0" || occs[0].LineNumber != 1 {
		t.Errorf("unexpected first occurrence: %+v", occs[0])
	}
	if occs[1].Package != "fletcher" || occs[1].Version != "0.5.0" {
		t.Errorf("unexpected second occurrence: %+v", occs[1])
	}
	if occs[3].Package != "touying" || occs[3].LineNumber != 2 {
		t.Errorf("unexpected fourth occurrence: %+v", occs[3])
	}
}

func TestScanDependencies_SingleFile(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "main.typ")
	if err := os.WriteFile(doc, []byte("#import \"@preview/cetz:0.2.0\": *\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	occs, err := scanDependencies(doc)
	if err != nil {
		t.Fatalf("scanDependencies failed: %v", err)
	}
	if len(occs) != 1 {
		t.Fatalf("expected 1 occurrence, got %d", len(occs))
	}
}

// Files the project ignores must not be scanned.
func TestScanDependencies_RespectsIgnoreFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".typstignore"), []byte("skipme.typ\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "keep.typ"), []byte("#import \"@preview/keep:1.0.0\": *\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skipme.typ"), []byte("#import \"@preview/skipped:1.0.0\": *\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	occs, err := scanDependencies(dir)
	if err != nil {
		t.Fatalf("scanDependencies failed: %v", err)
	}
	for _, occ := range occs {
		if occ.Package == "skipped" {
			t.Errorf("an ignored file was scanned: %+v", occ)
		}
	}
	if len(occs) != 1 {
		t.Errorf("expected 1 occurrence, got %d", len(occs))
	}
}

// A directory named like a source file must not be treated as one.
func TestScanDependencies_DirectoryNamedTypIsSkipped(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "weird.typ"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "real.typ"), []byte("#import \"@preview/real:1.0.0\": *\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	occs, err := scanDependencies(dir)
	if err != nil {
		t.Fatalf("scanDependencies failed: %v", err)
	}
	if len(occs) != 1 || occs[0].Package != "real" {
		t.Errorf("expected only the real file to be scanned, got %+v", occs)
	}
}

func TestGroupDependencies(t *testing.T) {
	idx := isolatedIndex(t, `[
		{"name": "cetz", "version": "0.2.2"},
		{"name": "fletcher", "version": "0.5.0"}
	]`)

	occs := []ImportOccurrence{
		{FilePath: "main.typ", LineNumber: 1, Package: "cetz", Version: "0.2.0"},
		{FilePath: "main.typ", LineNumber: 4, Package: "cetz", Version: "0.2.1"},
		{FilePath: "chap.typ", LineNumber: 2, Package: "fletcher", Version: "0.5.0"},
		{FilePath: "chap.typ", LineNumber: 9, Package: "ghost", Version: "1.0.0"},
	}

	deps := groupDependencies(occs, idx)
	if len(deps) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(deps))
	}

	byName := map[string]PackageDependency{}
	for _, d := range deps {
		byName[d.Name] = d
	}

	if got := byName["cetz"]; got.Status != StatusOutdated || got.LatestVersion != "0.2.2" {
		t.Errorf("cetz = %+v, want outdated at 0.2.2", got)
	}
	if got := byName["fletcher"]; got.Status != StatusUpToDate {
		t.Errorf("fletcher = %+v, want up to date", got)
	}
	// A package absent from the index has StatusNotInUniverse.
	if got := byName["ghost"]; got.Status != StatusNotInUniverse {
		t.Errorf("ghost = %+v, want not-in-universe", got)
	}
}

// The status enum must survive a JSON round trip, including its zero value.
func TestDependencyStatusJSONRoundTrip(t *testing.T) {
	statuses := []DependencyStatus{StatusUnknown, StatusUpToDate, StatusOutdated, StatusNotInUniverse}
	for _, status := range statuses {
		data, err := json.Marshal(status)
		if err != nil {
			t.Fatalf("marshal %v: %v", status, err)
		}
		var got DependencyStatus
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("unmarshal %s: %v", data, err)
		}
		if got != status {
			t.Errorf("round trip %v -> %s -> %v", status, data, got)
		}
	}
}

// Distinct versions are collected newest first.
func TestUniqueVersions(t *testing.T) {
	got := uniqueVersions([]ImportOccurrence{
		{Version: "0.2.0"},
		{Version: "0.10.0"},
		{Version: "0.2.0"},
		{Version: "0.9.9"},
	})
	want := []string{"0.10.0", "0.9.9", "0.2.0"}
	if len(got) != len(want) {
		t.Fatalf("uniqueVersions = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("uniqueVersions = %v, want %v", got, want)
			break
		}
	}
}

// planRewrites computes diffs without modifying files.
func TestPlanRewritesIsPure(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "main.typ")
	initial := "#import \"@preview/cetz:0.2.0\": *\n"
	if err := os.WriteFile(doc, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := planRewrites([]string{doc}, map[string]string{"cetz": "0.2.2"})
	if err != nil {
		t.Fatalf("planRewrites failed: %v", err)
	}
	if len(plan.Edits) != 1 {
		t.Fatalf("expected 1 edit, got %d", len(plan.Edits))
	}
	if plan.Edits[0].OldLine != strings.TrimSuffix(initial, "\n") || !strings.Contains(plan.Edits[0].NewLine, "cetz:0.2.2") {
		t.Errorf("unexpected edit: %+v", plan.Edits[0])
	}
	if len(plan.Files) != 1 || plan.Files[0] != doc {
		t.Errorf("plan.Files = %v, want [%s]", plan.Files, doc)
	}

	after, err := os.ReadFile(doc)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != initial {
		t.Errorf("planRewrites modified the file:\n%s", after)
	}
}

// Applying the plan must write exactly what the plan described and report the same
// number of occurrences. --dry-run and a real run share one rewrite, so a preview
// shows what the run will write.
func TestApplyRewritesMatchesPlan(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "main.typ")
	if err := os.WriteFile(doc, []byte("#import \"@preview/cetz:0.2.0\": *\r\n#let x = 1\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	targets := map[string]string{"cetz": "0.2.2"}
	planned, err := planRewrites([]string{doc}, targets)
	if err != nil {
		t.Fatalf("planRewrites failed: %v", err)
	}
	if len(planned.Edits) != 1 || planned.Occurrences() != 1 {
		t.Fatalf("plan = %+v, want exactly one edit and one occurrence", planned)
	}

	applied, err := applyRewrites([]string{doc}, targets)
	if err != nil {
		t.Fatalf("applyRewrites failed: %v", err)
	}
	if applied.Occurrences() != planned.Occurrences() || len(applied.Edits) != len(planned.Edits) {
		t.Errorf("apply = %+v, plan = %+v", applied, planned)
	}
	if applied.Edits[0].NewLine != planned.Edits[0].NewLine {
		t.Errorf("apply wrote %q, plan promised %q", applied.Edits[0].NewLine, planned.Edits[0].NewLine)
	}

	after, err := os.ReadFile(doc)
	if err != nil {
		t.Fatal(err)
	}
	if want := "#import \"@preview/cetz:0.2.2\": *\r\n#let x = 1\r\n"; string(after) != want {
		t.Errorf("apply result:\n got: %q\nwant: %q", after, want)
	}
}

// A version that moved on after the index was read must not be downgraded: the
// rewrite only moves versions forward.
func TestApplyRewritesNeverDowngrades(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "main.typ")
	initial := "#import \"@preview/cetz:0.3.0\": *\n"
	if err := os.WriteFile(doc, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := applyRewrites([]string{doc}, map[string]string{"cetz": "0.2.2"})
	if err != nil {
		t.Fatalf("applyRewrites failed: %v", err)
	}
	if plan.Occurrences() != 0 {
		t.Errorf("rewrote %d occurrence(s) of a version newer than the target", plan.Occurrences())
	}

	after, err := os.ReadFile(doc)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != initial {
		t.Errorf("a newer version was rewritten:\n got: %q\nwant: %q", after, initial)
	}
}

// Occurrences counts rewritten version strings rather than edited lines.
func TestRewritePlanCountsOccurrences(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "main.typ")
	content := "#import \"@preview/cetz:0.1.0\": * #import \"@preview/cetz:0.1.1\": *\n"
	if err := os.WriteFile(doc, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := applyRewrites([]string{doc}, map[string]string{"cetz": "0.2.2"})
	if err != nil {
		t.Fatalf("applyRewrites failed: %v", err)
	}
	if plan.Occurrences() != 2 {
		t.Errorf("Occurrences() = %d, want 2", plan.Occurrences())
	}
	if len(plan.Edits) != 1 {
		t.Errorf("Edits = %d, want one edited line", len(plan.Edits))
	}
	if plan.Hits["cetz"] != 2 {
		t.Errorf("Hits[cetz] = %d, want 2", plan.Hits["cetz"])
	}
}

// The rewrite must leave alone any spec that is already current or newer, along
// with text that is not a spec.
func TestRewriteSpecs(t *testing.T) {
	targets := map[string]string{"cetz": "0.2.2"}

	tests := []struct {
		name      string
		line      string
		wantLine  string
		wantCount int
	}{
		{"updates older", `#import "@preview/cetz:0.2.0": *`, `#import "@preview/cetz:0.2.2": *`, 1},
		{"already current", `#import "@preview/cetz:0.2.2": *`, `#import "@preview/cetz:0.2.2": *`, 0},
		{"newer untouched", `#import "@preview/cetz:0.9.0": *`, `#import "@preview/cetz:0.9.0": *`, 0},
		{"prerelease is behind a release", `#import "@preview/cetz:0.2.2-rc1": *`, `#import "@preview/cetz:0.2.2": *`, 1},
		{"single quotes", `#import '@preview/cetz:0.1.0': *`, `#import '@preview/cetz:0.2.2': *`, 1},
		{"other package", `#import "@preview/other:0.1.0": *`, `#import "@preview/other:0.1.0": *`, 0},
		{"two on one line", `#import "@preview/cetz:0.1.0": * #import "@preview/cetz:0.1.1": *`,
			`#import "@preview/cetz:0.2.2": * #import "@preview/cetz:0.2.2": *`, 2},
		{"unquoted spec untouched", `#import @preview/cetz:0.1.0: *`, `#import @preview/cetz:0.1.0: *`, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotLine, gotCount := rewriteSpecs(tc.line, targets, nil)
			if gotLine != tc.wantLine {
				t.Errorf("line:\n got: %q\nwant: %q", gotLine, tc.wantLine)
			}
			if gotCount != tc.wantCount {
				t.Errorf("count = %d, want %d", gotCount, tc.wantCount)
			}
		})
	}
}

// Split and join must round-trip every line-ending convention untouched.
func TestSplitJoinContentPreservesEndings(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
	}{
		{"lf", "a\nb\n"},
		{"crlf", "a\r\nb\r\n"},
		{"no trailing newline", "a\nb"},
		{"empty", ""},
		{"single line", "a"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lines, trailing := splitContent([]byte(tc.input))
			got := string(joinContent(lines, trailing))
			if got != tc.input {
				t.Errorf("round trip changed content: got %q, want %q", got, tc.input)
			}
		})
	}
}
