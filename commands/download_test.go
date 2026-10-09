package commands_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Vncntvx/typush/commands"
	"github.com/Vncntvx/typush/util"
)

func TestDownload_DryRun(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("git command stub requires /bin/sh")
	}

	fixture := installTestPackage(t)
	binDir := t.TempDir()
	gitStub := `#!/bin/sh
set -eu
printf '%s\n' "$*" >> "$TYPUSH_TEST_GIT_LOG"
case "$1" in
  clone)
    mkdir -p "$3"
    cp -R "$TYPUSH_TEST_PACKAGE/." "$3/"
    ;;
  checkout)
    test -f typst.toml
    ;;
  *) exit 1 ;;
esac
`
	if err := os.WriteFile(filepath.Join(binDir, "git"), []byte(gitStub), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("TYPUSH_TEST_PACKAGE", fixture)

	for _, tc := range []struct {
		name     string
		checkout string
		existing bool
	}{
		{name: "new install"},
		{name: "existing install with checkout", checkout: "v1.2.3", existing: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("XDG_DATA_HOME", t.TempDir())
			base, err := util.TypstLocalDir()
			if err != nil {
				t.Fatal(err)
			}
			targetDir := filepath.Join(base, "local", "demo-pkg", "1.2.3")
			marker := filepath.Join(targetDir, "lib.typ")
			if tc.existing {
				if err := os.MkdirAll(targetDir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(marker, []byte("existing install"), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			logPath := filepath.Join(t.TempDir(), "git.log")
			t.Setenv("TYPUSH_TEST_GIT_LOG", logPath)
			// A unique repository ID isolates Download's deterministic temp path.
			repository := filepath.Join(t.TempDir(), "repository")
			stdout, stderr, err := captureStdio(t, func() error {
				return commands.Download(repository, tc.checkout, "local", commands.Preview)
			})
			if err != nil {
				t.Fatalf("Download preview failed: %v", err)
			}
			if stdout != "" {
				t.Errorf("preview must not write to stdout, got %q", stdout)
			}
			for _, want := range []string{
				targetDir,
				"Dry run: entries to install",
				"  src/extra.typ\n",
				"Dry run: installation skipped, no files written",
			} {
				if !strings.Contains(stderr, want) {
					t.Errorf("stderr missing %q, got:\n%s", want, stderr)
				}
			}
			// One counted item block and at most one summary line for the whole
			// command: download's install step must not add a second summary.
			summaries := 0
			for _, line := range strings.Split(stderr, "\n") {
				if strings.HasPrefix(line, "Dry run: ") && !strings.HasSuffix(line, ":") {
					summaries++
				}
			}
			if summaries != 1 {
				t.Errorf("expected exactly one dry-run summary line, got %d:\n%s", summaries, stderr)
			}
			if tc.existing {
				data, err := os.ReadFile(marker)
				if err != nil || string(data) != "existing install" {
					t.Errorf("preview changed existing install: content=%q, err=%v", data, err)
				}
			} else if _, err := os.Stat(base); !os.IsNotExist(err) {
				t.Errorf("preview must not create the packages directory, err=%v", err)
			}
			if _, err := os.Stat(util.TempSubdir(repository)); !os.IsNotExist(err) {
				t.Errorf("download temp directory must be removed, err=%v", err)
			}
			log, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatal(err)
			}
			wantLog := "clone " + repository + " " + util.TempSubdir(repository) + "\n"
			if tc.checkout != "" {
				wantLog += "checkout " + tc.checkout + "\n"
			}
			if string(log) != wantLog {
				t.Errorf("git calls = %q, want %q", log, wantLog)
			}
		})
	}
}
