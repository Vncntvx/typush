package commands_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Vncntvx/typush/commands"
	"github.com/Vncntvx/typush/util"
)

const searchIndex = `[
	{"name": "cetz", "version": "0.2.2", "description": "Drawing library for Typst", "categories": ["visualization"]},
	{"name": "cetz-plot", "version": "0.1.0", "description": "Plotting add-on for CeTZ", "keywords": ["plot", "chart"]},
	{"name": "fletcher", "version": "0.5.1", "description": "Commutative diagrams", "keywords": ["diagram", "drawing"]}
]`

func TestSearch(t *testing.T) {
	useUniverse(t, searchIndex)

	stdout, err := captureStdout(t, func() error {
		return commands.Search("cetz", 10, commands.UniverseOptions{Refresh: true})
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	for _, want := range []string{"NAME", "cetz", "cetz-plot", "Found 2 package(s)"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected %q in table output, got:\n%s", want, stdout)
		}
	}

	jsonOut, err := captureStdout(t, func() error {
		return commands.Search("cetz", 10, commands.UniverseOptions{AsJSON: true})
	})
	if err != nil {
		t.Fatalf("Search --json failed: %v", err)
	}
	var results []util.UniverseSearchResult
	if err := json.Unmarshal([]byte(jsonOut), &results); err != nil {
		t.Fatalf("failed to parse JSON: %v, raw: %s", err, jsonOut)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Name != "cetz" {
		t.Errorf("expected cetz first, got %s", results[0].Name)
	}

	if err := commands.Search("", 10, commands.UniverseOptions{}); err == nil {
		t.Error("expected an error for an empty query")
	}

	stdout, stderr, err := captureStdio(t, func() error {
		return commands.Search("nonexistent-pkg", 10, commands.UniverseOptions{})
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	// A search that matches nothing is a diagnostic, so it goes to stderr; stdout
	// stays reserved for the table and the JSON.
	if stdout != "" {
		t.Errorf("not-found notice reached stdout: %q", stdout)
	}
	if !strings.Contains(stderr, `No packages found matching "nonexistent-pkg".`) {
		t.Errorf("expected a not-found message on stderr, got: %s", stderr)
	}
}

// A non-positive limit falls back to the default limit.
func TestSearch_NonPositiveLimitUsesDefault(t *testing.T) {
	useUniverse(t, searchIndex)

	for _, limit := range []int{0, -5} {
		out, err := captureStdout(t, func() error {
			return commands.Search("cetz", limit, commands.UniverseOptions{})
		})
		if err != nil {
			t.Fatalf("Search(limit=%d) failed: %v", limit, err)
		}
		if !strings.Contains(out, "Found 2 package(s)") {
			t.Errorf("limit=%d should fall back to the default, got:\n%s", limit, out)
		}
	}
}

// A long description must be truncated on a rune boundary.
func TestSearch_TruncatesLongDescription(t *testing.T) {
	long := strings.Repeat("图", 100)
	useUniverse(t, `[{"name": "wide", "version": "1.0.0", "description": "`+long+`"}]`)

	out, err := captureStdout(t, func() error {
		return commands.Search("wide", 10, commands.UniverseOptions{Refresh: true})
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if !strings.Contains(out, "...") {
		t.Errorf("expected a truncated description, got:\n%s", out)
	}
	if strings.Contains(out, long) {
		t.Error("description was not truncated")
	}
}
