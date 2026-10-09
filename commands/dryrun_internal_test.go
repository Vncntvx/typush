package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// streamCapture redirects stdout and stderr to temporary files during fn.
func streamCapture(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()
	dir := t.TempDir()
	outPath, errPath := filepath.Join(dir, "stdout"), filepath.Join(dir, "stderr")
	outF, err := os.Create(outPath)
	if err != nil {
		t.Fatalf("failed to create stdout file: %v", err)
	}
	errF, err := os.Create(errPath)
	if err != nil {
		t.Fatalf("failed to create stderr file: %v", err)
	}
	origStdout, origStderr := os.Stdout, os.Stderr
	t.Cleanup(func() { os.Stdout, os.Stderr = origStdout, origStderr })
	os.Stdout, os.Stderr = outF, errF
	fn()
	os.Stdout, os.Stderr = origStdout, origStderr
	_ = outF.Close()
	_ = errF.Close()
	read := func(p string) string {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("failed to read captured stream: %v", err)
		}
		return string(b)
	}
	return read(outPath), read(errPath)
}

// Informational lines and preview framing must never reach stdout.
func TestStderrHelpersOnlyWriteToStderr(t *testing.T) {
	stdout, stderr := streamCapture(t, func() {
		infof("Destination directory:\n  %s", "/tmp/demo")
		previewNote("would bump version from %s to %s", "0.1.0", "0.2.0")
		previewItems("patterns to exclude", []string{"*.log", "docs/drafts"})
	})
	if stdout != "" {
		t.Errorf("preview helpers must never write to stdout, got %q", stdout)
	}
	for _, want := range []string{
		"Destination directory:\n  /tmp/demo",
		"Dry run: would bump version from 0.1.0 to 0.2.0",
		"Dry run: patterns to exclude (2 item(s)):\n  *.log\n  docs/drafts",
	} {
		if !strings.Contains(stderr, want) {
			t.Errorf("expected %q in stderr, got:\n%s", want, stderr)
		}
	}
}

func TestPreviewItemsEmptyList(t *testing.T) {
	_, stderr := streamCapture(t, func() {
		previewItems("symlinks to remove", nil)
	})
	if want := "Dry run: symlinks to remove (0 item(s)):"; !strings.Contains(stderr, want) {
		t.Errorf("expected %q in stderr, got:\n%s", want, stderr)
	}
}

// Multi-line items must be indented on every line.
func TestPreviewItemsIndentsMultiLineItems(t *testing.T) {
	_, stderr := streamCapture(t, func() {
		previewItems("dependency updates", []string{"main.typ:3:\n- old\n+ new"})
	})
	want := "Dry run: dependency updates (1 item(s)):\n  main.typ:3:\n  - old\n  + new\n"
	if stderr != want {
		t.Errorf("multi-line preview item:\n got: %q\nwant: %q", stderr, want)
	}
}

// Execute and Preview define the dry-run modes used across commands.
func TestDryRunModeConstants(t *testing.T) {
	if Execute != false {
		t.Errorf("Execute = %v, want false", Execute)
	}
	if Preview != true {
		t.Errorf("Preview = %v, want true", Preview)
	}
}
