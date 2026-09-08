package util

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

// AppName is the binary name.
const AppName = "typush"

const defaultPackagesSubdir = "typst/packages" // from typst-kit

// TypstLocalDir is the data dir (not cache) where packages are installed.
func TypstLocalDir() (string, error) {
	var base string
	// Prefer XDG_DATA_HOME on all platforms for predictability, fall back to OS default.
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		base = xdg
	} else {
		var err error
		base, err = dataDir()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(base, defaultPackagesSubdir), nil
}

func TempSubdir(id string) string {
	sum := sha256.Sum256([]byte(id))
	return filepath.Join(os.TempDir(), fmt.Sprintf("%s-%x", AppName, sum[:8]))
}
