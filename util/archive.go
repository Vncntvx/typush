package util

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractTarGz extracts a .tar.gz stream into destDir.
// It checks paths and symlink targets to reject directory traversal outside destDir.
func ExtractTarGz(r io.Reader, destDir string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("invalid gzip stream: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	destClean := filepath.Clean(destDir)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed reading tar stream: %w", err)
		}

		target := filepath.Join(destClean, header.Name)
		cleanTarget := filepath.Clean(target)

		// Reject paths that escape destClean.
		if cleanTarget != destClean && !strings.HasPrefix(cleanTarget, destClean+string(filepath.Separator)) {
			return fmt.Errorf("illegal file path in archive (path traversal): %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(cleanTarget, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(cleanTarget), 0o755); err != nil {
				return err
			}
			perm := header.FileInfo().Mode().Perm()
			if perm == 0 {
				perm = 0o644
			}
			outFile, err := os.OpenFile(cleanTarget, os.O_CREATE|os.O_RDWR|os.O_TRUNC, perm)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			if err := outFile.Close(); err != nil {
				return err
			}
		case tar.TypeSymlink:
			// Ensure symlink target is relative and stays within destClean.
			linkTarget := header.Linkname
			if filepath.IsAbs(linkTarget) {
				return fmt.Errorf("absolute symlink in archive rejected: %s -> %s", header.Name, linkTarget)
			}
			resolved := filepath.Clean(filepath.Join(filepath.Dir(cleanTarget), linkTarget))
			if resolved != destClean && !strings.HasPrefix(resolved, destClean+string(filepath.Separator)) {
				return fmt.Errorf("symlink target escapes destination: %s -> %s", header.Name, linkTarget)
			}
			if err := os.MkdirAll(filepath.Dir(cleanTarget), 0o755); err != nil {
				return err
			}
			_ = os.Remove(cleanTarget)
			if err := os.Symlink(linkTarget, cleanTarget); err != nil {
				return err
			}
		}
	}

	return nil
}
