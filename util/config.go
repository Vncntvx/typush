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
// Respects TYPST_PACKAGE_PATH, then XDG_DATA_HOME, then OS default.
func TypstLocalDir() (string, error) {
	if env := os.Getenv("TYPST_PACKAGE_PATH"); env != "" {
		return env, nil
	}
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

// userCacheDir resolves the base cache directory (XDG_CACHE_HOME with OS fallback).
func userCacheDir() (string, error) {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return xdg, nil
	}
	return os.UserCacheDir()
}

// TypstPackageCacheDir returns the directory where Typst caches downloaded
// remote packages (e.g. @preview packages fetched during compilation).
// Respects TYPST_PACKAGE_CACHE_PATH, then XDG_CACHE_HOME, then OS default.
func TypstPackageCacheDir() (string, error) {
	if env := os.Getenv("TYPST_PACKAGE_CACHE_PATH"); env != "" {
		return env, nil
	}
	base, err := userCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, defaultPackagesSubdir), nil
}

// TypstCacheDir returns the directory used to cache remote index files.
func TypstCacheDir() (string, error) {
	base, err := userCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, AppName), nil
}

func TempSubdir(id string) string {
	sum := sha256.Sum256([]byte(id))
	return filepath.Join(os.TempDir(), fmt.Sprintf("%s-%x", AppName, sum[:8]))
}
