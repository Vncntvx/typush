package commands_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vncntvx/typush/commands"
)

func TestCIGenerate_DryRun(t *testing.T) {
	dir := t.TempDir()
	wfFile := filepath.Join(dir, ".github", "workflows", "release-typst.yml")
	opts := commands.GenerateOptions{
		Source:      "my-org/my-packages",
		PushToFork:  "my-fork/packages",
		Destination: "packages/preview",
	}

	// 1. Dry run prints to stdout and writes nothing.
	stdout, stderr, err := captureStdio(t, func() error {
		preview := opts
		preview.DryRun = true
		return commands.Generate(dir, preview)
	})
	if err != nil {
		t.Fatalf("Generate dryRun failed: %v", err)
	}
	if !strings.Contains(stdout, "my-org/my-packages") {
		t.Errorf("expected source repository in dryRun stdout, got:\n%s", stdout)
	}
	if !strings.Contains(stderr, "Dry run: release-typst.yml previewed to stdout, file not written") {
		t.Errorf("expected dryRun message in stderr, got: %s", stderr)
	}

	// Verify file was NOT created
	if _, err := os.Stat(wfFile); !os.IsNotExist(err) {
		t.Errorf("workflow file should not exist after dryRun")
	}

	// 2. Real generate
	if err := commands.Generate(dir, opts); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if _, err := os.Stat(wfFile); err != nil {
		t.Errorf("workflow file should exist after real Generate, err=%v", err)
	}
}
