package util_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vncntvx/typush/util"
)

func createTestTarGz(t *testing.T, entries map[string][]byte) []byte {
	t.Helper()
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
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("WriteHeader failed: %v", err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("tw.Close failed: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("gzw.Close failed: %v", err)
	}
	return buf.Bytes()
}

func TestExtractTarGz(t *testing.T) {
	archiveData := createTestTarGz(t, map[string][]byte{
		"typst.toml":  []byte(`[package]\nname = "test"\nversion = "0.1.0"`),
		"src/lib.typ": []byte(`#let hello() = "world"`),
		"README.md":   []byte("# Test Package"),
	})

	dest := t.TempDir()
	if err := util.ExtractTarGz(bytes.NewReader(archiveData), dest); err != nil {
		t.Fatalf("ExtractTarGz failed: %v", err)
	}

	manifestPath := filepath.Join(dest, "typst.toml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("failed to read extracted manifest: %v", err)
	}
	if !bytes.Contains(data, []byte("test")) {
		t.Errorf("extracted content mismatch, got %s", string(data))
	}

	libPath := filepath.Join(dest, "src", "lib.typ")
	if _, err := os.Stat(libPath); err != nil {
		t.Fatalf("src/lib.typ does not exist: %v", err)
	}
}

func TestExtractTarGzSlipProtection(t *testing.T) {
	// Archive with directory traversal attempt
	archiveData := createTestTarGz(t, map[string][]byte{
		"../../evil.txt": []byte("malicious content"),
	})

	dest := t.TempDir()
	err := util.ExtractTarGz(bytes.NewReader(archiveData), dest)
	if err == nil {
		t.Fatal("expected ExtractTarGz to fail on path traversal attempt, but it succeeded")
	}

	// Verify evil.txt was NOT written outside dest
	parentEvil := filepath.Join(dest, "..", "evil.txt")
	if _, statErr := os.Stat(parentEvil); statErr == nil {
		t.Fatal("evil.txt was written outside target directory!")
	}
}

func TestExtractTarGzInvalidGzip(t *testing.T) {
	dest := t.TempDir()
	err := util.ExtractTarGz(bytes.NewReader([]byte("not a gzip stream")), dest)
	if err == nil {
		t.Fatal("expected error on invalid gzip stream, got nil")
	}
}
