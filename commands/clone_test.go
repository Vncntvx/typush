package commands_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vncntvx/typush/commands"
	"github.com/Vncntvx/typush/util"
)

func makeTestTarGz(entries map[string]string) []byte {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	for name, content := range entries {
		hdr := &tar.Header{
			Name:     name,
			Mode:     0o644,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}
		_ = tw.WriteHeader(hdr)
		_, _ = tw.Write([]byte(content))
	}

	_ = tw.Close()
	_ = gzw.Close()
	return buf.Bytes()
}

func setupCloneServer(t *testing.T) {
	t.Helper()

	indexJSON := `[
		{
			"name": "mock-pkg",
			"version": "0.1.0",
			"description": "Mock package v0.1.0",
			"entrypoint": "lib.typ",
			"authors": ["Alice"],
			"license": "MIT"
		},
		{
			"name": "mock-pkg",
			"version": "0.2.0",
			"description": "Mock package v0.2.0",
			"entrypoint": "lib.typ",
			"authors": ["Alice"],
			"license": "MIT"
		}
	]`

	tarGzV1 := makeTestTarGz(map[string]string{
		"typst.toml": "[package]\nname = \"mock-pkg\"\nversion = \"0.1.0\"",
		"lib.typ":    "#let v = 1",
	})
	tarGzV2 := makeTestTarGz(map[string]string{
		"typst.toml": "[package]\nname = \"mock-pkg\"\nversion = \"0.2.0\"",
		"lib.typ":    "#let v = 2",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/preview/index.json":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, indexJSON)
		case "/preview/mock-pkg-0.1.0.tar.gz":
			w.Header().Set("Content-Type", "application/gzip")
			_, _ = w.Write(tarGzV1)
		case "/preview/mock-pkg-0.2.0.tar.gz":
			w.Header().Set("Content-Type", "application/gzip")
			_, _ = w.Write(tarGzV2)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	origURL := util.UniverseIndexURL
	t.Cleanup(func() { util.UniverseIndexURL = origURL })
	util.UniverseIndexURL = server.URL + "/preview/index.json"
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
}

func TestCloneSpecificVersion(t *testing.T) {
	setupCloneServer(t)

	dest := filepath.Join(t.TempDir(), "output")
	_, err := captureStderr(t, func() error {
		return commands.Clone(commands.CloneOptions{
			Spec:   "@preview/mock-pkg:0.1.0",
			Dest:   dest,
			DryRun: commands.Execute,
		})
	})
	if err != nil {
		t.Fatalf("Clone failed: %v", err)
	}

	manifestData, err := os.ReadFile(filepath.Join(dest, "typst.toml"))
	if err != nil {
		t.Fatalf("failed to read typst.toml: %v", err)
	}
	if !strings.Contains(string(manifestData), "0.1.0") {
		t.Errorf("expected v0.1.0, got %s", string(manifestData))
	}
}

func TestCloneLatestVersion(t *testing.T) {
	setupCloneServer(t)

	dest := filepath.Join(t.TempDir(), "output-latest")
	_, err := captureStderr(t, func() error {
		return commands.Clone(commands.CloneOptions{
			Spec:   "mock-pkg",
			Dest:   dest,
			DryRun: commands.Execute,
		})
	})
	if err != nil {
		t.Fatalf("Clone latest failed: %v", err)
	}

	manifestData, err := os.ReadFile(filepath.Join(dest, "typst.toml"))
	if err != nil {
		t.Fatalf("failed to read typst.toml: %v", err)
	}
	if !strings.Contains(string(manifestData), "0.2.0") {
		t.Errorf("expected latest v0.2.0, got %s", string(manifestData))
	}
}

func TestCloneDryRun(t *testing.T) {
	setupCloneServer(t)

	dest := filepath.Join(t.TempDir(), "output-dryrun")
	stderr, err := captureStderr(t, func() error {
		return commands.Clone(commands.CloneOptions{
			Spec:   "mock-pkg:0.1.0",
			Dest:   dest,
			DryRun: commands.Preview,
		})
	})
	if err != nil {
		t.Fatalf("Clone dry-run failed: %v", err)
	}

	if !strings.Contains(stderr, "Dry run: clone skipped, no files downloaded") {
		t.Errorf("expected dry-run note in stderr, got:\n%s", stderr)
	}

	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Errorf("expected destination to not be created in dry-run mode")
	}
}
