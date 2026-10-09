package commands_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vncntvx/typush/commands"
)

const outdatedIndex = `[
	{"name": "cetz", "version": "0.2.2"},
	{"name": "fletcher", "version": "0.5.1"}
]`

func TestOutdated(t *testing.T) {
	useUniverse(t, outdatedIndex)

	tmpDir := t.TempDir()
	docPath := filepath.Join(tmpDir, "main.typ")
	content := `#import "@preview/cetz:0.2.0": *
#import "@preview/fletcher:0.5.1": *
`
	if err := os.WriteFile(docPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, err := captureStdout(t, func() error {
		return commands.Outdated(tmpDir, commands.UniverseOptions{Refresh: true})
	})
	if err != nil {
		t.Fatalf("Outdated failed: %v", err)
	}
	for _, want := range []string{"PACKAGE", "cetz", "Update available", "fletcher", "Up to date",
		"1 of 2 package dependencies have updates available."} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected %q in table output, got:\n%s", want, stdout)
		}
	}
}

// --json must expose the status as a stable string, and flag a package that is
// absent from Universe distinctly from one that is up to date.
func TestOutdated_JSON(t *testing.T) {
	useUniverse(t, outdatedIndex)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.typ"), []byte(
		`#import "@preview/cetz:0.2.2": *
#import "@preview/ghost:1.0.0": *
`), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := captureStdout(t, func() error {
		return commands.Outdated(dir, commands.UniverseOptions{AsJSON: true, Refresh: true})
	})
	if err != nil {
		t.Fatalf("Outdated --json failed: %v", err)
	}

	var deps []commands.PackageDependency
	if err := json.Unmarshal([]byte(out), &deps); err != nil {
		t.Fatalf("failed to parse JSON: %v, raw: %s", err, out)
	}
	if len(deps) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(deps))
	}

	statuses := map[string]string{}
	for _, d := range deps {
		statuses[d.Name] = d.Status.String()
	}
	if statuses["cetz"] != "up-to-date" {
		t.Errorf("cetz status = %q, want up-to-date", statuses["cetz"])
	}
	if statuses["ghost"] != "not-in-universe" {
		t.Errorf("ghost status = %q, want not-in-universe", statuses["ghost"])
	}
}

// An empty result set must encode as valid JSON.
func TestOutdated_EmptyIsValidJSON(t *testing.T) {
	useUniverse(t, outdatedIndex)

	out, err := captureStdout(t, func() error {
		return commands.Outdated(t.TempDir(), commands.UniverseOptions{AsJSON: true})
	})
	if err != nil {
		t.Fatalf("Outdated --json failed: %v", err)
	}

	var deps []commands.PackageDependency
	if err := json.Unmarshal([]byte(out), &deps); err != nil {
		t.Fatalf("empty result is not valid JSON: %v, raw: %q", err, out)
	}
	if len(deps) != 0 {
		t.Errorf("expected no dependencies, got %d", len(deps))
	}
}

// Outdated fails if the index cannot be retrieved and no cache exists.
func TestOutdated_FailsWhenIndexUnavailable(t *testing.T) {
	useFailingUniverse(t)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.typ"), []byte(
		`#import "@preview/cetz:0.1.0": *
`), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, err := captureStdout(t, func() error {
		return commands.Outdated(dir, commands.UniverseOptions{Refresh: true})
	})
	if err == nil {
		t.Fatalf("expected an error when the index is unavailable, got output:\n%s", stdout)
	}
	if strings.Contains(stdout, "up to date") {
		t.Errorf("must not claim dependencies are up to date without an index:\n%s", stdout)
	}
}

// A directory with no dependencies needs no index at all.
func TestOutdated_NoDependencies(t *testing.T) {
	useUniverse(t, outdatedIndex)

	stdout, stderr, err := captureStdio(t, func() error {
		return commands.Outdated(t.TempDir(), commands.UniverseOptions{})
	})
	if err != nil {
		t.Fatalf("Outdated failed: %v", err)
	}
	// Nothing to report is a diagnostic, so it goes to stderr with the other notes,
	// leaving stdout for the table.
	if stdout != "" {
		t.Errorf("no-dependency notice reached stdout: %q", stdout)
	}
	if !strings.Contains(stderr, "No Typst package dependencies (@preview) found.") {
		t.Errorf("expected no-dependency notice on stderr, got: %s", stderr)
	}
}

// Dependency ordering and occurrence locations must be deterministic across runs.
func TestOutdated_DeterministicOutput(t *testing.T) {
	useUniverse(t, outdatedIndex)

	dir := t.TempDir()
	for _, f := range []string{"a.typ", "b.typ", "c.typ"} {
		var b strings.Builder
		for _, name := range []string{"cetz", "fletcher", "aaa", "zzz"} {
			b.WriteString("#import \"@preview/" + name + ":0.0.1\": *\n")
			b.WriteString("#import \"@preview/" + name + ":0.0.2\": *\n")
		}
		if err := os.WriteFile(filepath.Join(dir, f), []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	var runs []string
	for i := 0; i < 5; i++ {
		out, err := captureStdout(t, func() error {
			return commands.Outdated(dir, commands.UniverseOptions{Refresh: true})
		})
		if err != nil {
			t.Fatalf("Outdated failed: %v", err)
		}
		runs = append(runs, out)
	}
	for i := 1; i < len(runs); i++ {
		if runs[i] != runs[0] {
			t.Errorf("output differs between runs:\nrun0:\n%s\nrun%d:\n%s", runs[0], i, runs[i])
		}
	}
}

// A package imported from many places must summarise its locations compactly
// and in a stable order.
func TestOutdated_CollapsesLocations(t *testing.T) {
	useUniverse(t, outdatedIndex)

	dir := t.TempDir()
	for _, f := range []string{"a.typ", "b.typ", "c.typ", "d.typ"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("#import \"@preview/cetz:0.1.0\": *\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	out, err := captureStdout(t, func() error {
		return commands.Outdated(dir, commands.UniverseOptions{Refresh: true})
	})
	if err != nil {
		t.Fatalf("Outdated failed: %v", err)
	}
	if !strings.Contains(out, "(+2 more)") {
		t.Errorf("expected the extra locations to be collapsed, got:\n%s", out)
	}
}
