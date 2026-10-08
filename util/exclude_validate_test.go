package util_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Vncntvx/typush/util"
)

func TestValidateExcludePatterns(t *testing.T) {
	if err := util.ValidateExcludePatterns([]string{"*.log", "docs/drafts", "tests/"}); err != nil {
		t.Errorf("valid patterns should pass, got %v", err)
	}
	if err := util.ValidateExcludePatterns([]string{"a", ""}); err == nil {
		t.Error("expected an empty pattern to be rejected")
	}
	if err := util.ValidateExcludePatterns([]string{"["}); err == nil {
		t.Error("expected a malformed glob to be rejected")
	}
}

func TestListInstallDirectoryExcludePatterns(t *testing.T) {
	root := t.TempDir()
	testFile := filepath.Join(root, "tests", "nested", "example.typ")
	if err := os.MkdirAll(filepath.Dir(testFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testFile, []byte("// test fixture"), 0o644); err != nil {
		t.Fatal(err)
	}
	libFile := filepath.Join(root, "lib.typ")
	if err := os.WriteFile(libFile, []byte("// entrypoint"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		pattern      string
		wantTestFile bool
	}{
		{pattern: "tests/", wantTestFile: true},
		{pattern: "tests/**", wantTestFile: false},
	} {
		t.Run(tc.pattern, func(t *testing.T) {
			entries, err := util.ListInstall(root, []string{tc.pattern})
			if err != nil {
				t.Fatal(err)
			}
			if got := slices.Contains(entries, testFile); got != tc.wantTestFile {
				t.Errorf("test file present = %v, want %v", got, tc.wantTestFile)
			}
			if !slices.Contains(entries, libFile) {
				t.Error("directory exclusion must preserve the entrypoint")
			}
		})
	}
}

// ListInstall must keep rejecting invalid patterns without touching the disk.
func TestListInstallExcludePatternValidation(t *testing.T) {
	for _, patterns := range [][]string{{""}, {"["}, {"ok", "["}} {
		if _, err := util.ListInstall(t.TempDir(), patterns); err == nil {
			t.Errorf("expected ListInstall to reject %q", patterns)
		}
	}
	if _, err := util.ListInstall(t.TempDir(), []string{"*.log"}); err != nil {
		t.Errorf("valid patterns should still work, got %v", err)
	}
}
