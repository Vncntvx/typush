package commands_test

import (
	"strings"
	"testing"

	"github.com/Vncntvx/typush/commands"
	"github.com/Vncntvx/typush/manifest"
)

func TestExclude_DryRun(t *testing.T) {
	dir := t.TempDir()
	writeTestPackage(t, dir)

	// 1. Dry run
	stderr, err := captureStderr(t, func() error {
		return commands.Add(dir, []string{"*.log", "docs/drafts"}, commands.Preview)
	})
	if err != nil {
		t.Fatalf("Add dryRun failed: %v", err)
	}
	if !strings.Contains(stderr, "Dry run: patterns to exclude (2 item(s)):") {
		t.Errorf("expected patterns header in stderr, got: %s", stderr)
	}
	if !strings.Contains(stderr, "*.log") || !strings.Contains(stderr, "docs/drafts") {
		t.Errorf("expected patterns in stderr, got: %s", stderr)
	}
	if !strings.Contains(stderr, "Dry run: typst.toml unchanged") {
		t.Errorf("expected dry-run completed message in stderr, got: %s", stderr)
	}

	// Verify typst.toml was not changed
	m, err := manifest.Read(dir)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}
	for _, e := range m.Package.Exclude {
		if e == "*.log" || e == "docs/drafts" {
			t.Errorf("typst.toml exclude should not have changed in dry-run, found %q", e)
		}
	}

	// 2. Real add
	out, err := captureStdout(t, func() error {
		return commands.Add(dir, []string{"*.log"}, commands.Execute)
	})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if !strings.Contains(out, "Excluded 1 pattern(s)") {
		t.Errorf("expected success message in stdout, got: %s", out)
	}

	// Verify typst.toml was updated
	mAfter, err := manifest.Read(dir)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}
	found := false
	for _, e := range mAfter.Package.Exclude {
		if e == "*.log" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected *.log in exclude list after real Add")
	}
}

// Adding an already excluded pattern is a no-op: the reported count must match
// what the dry run previewed, and typst.toml must be left byte-identical.
func TestExclude_DuplicatePatternIsNoOp(t *testing.T) {
	dir := t.TempDir()
	writeTestPackage(t, dir)
	before, err := manifest.Read(dir)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}

	stderr, err := captureStderr(t, func() error {
		return commands.Add(dir, []string{"tests"}, commands.Preview)
	})
	if err != nil {
		t.Fatalf("Add dryRun failed: %v", err)
	}
	if !strings.Contains(stderr, "Dry run: no new exclude patterns") {
		t.Errorf("expected no-op preview message, got: %s", stderr)
	}

	out, err := captureStdout(t, func() error {
		return commands.Add(dir, []string{"tests"}, commands.Execute)
	})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if !strings.Contains(out, "Excluded 0 pattern(s)") {
		t.Errorf("expected zero count on stdout, got: %s", out)
	}

	after, err := manifest.Read(dir)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}
	if len(after.Package.Exclude) != len(before.Package.Exclude) {
		t.Errorf("exclude list changed on a no-op add: %v -> %v", before.Package.Exclude, after.Package.Exclude)
	}
}

// A malformed glob must fail identically with and without a preview.
func TestExclude_InvalidPattern(t *testing.T) {
	dir := t.TempDir()
	writeTestPackage(t, dir)

	_, err := captureStderr(t, func() error {
		return commands.Add(dir, []string{"["}, commands.Preview)
	})
	if err == nil {
		t.Error("expected dry run to reject an invalid glob")
	}
	if _, err := captureStderr(t, func() error {
		return commands.Add(dir, []string{"["}, commands.Execute)
	}); err == nil {
		t.Error("expected real run to reject an invalid glob")
	}
}
