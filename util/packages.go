package util

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Vncntvx/typush/manifest"
)

// PackageSource describes how a package version entry was installed.
type PackageSource string

const (
	SourceInstalled  PackageSource = "installed"   // regular directory (typush install / download)
	SourceDevLink    PackageSource = "dev-link"    // active symlink (typush dev)
	SourceBrokenLink PackageSource = "broken-link" // broken symlink
	SourceCache      PackageSource = "cache"       // Typst download cache (@preview)
)

// InstalledPackage represents one installed package version.
type InstalledPackage struct {
	Namespace string        `json:"namespace"`
	Name      string        `json:"name"`
	Version   string        `json:"version"`
	Source    PackageSource `json:"source"`
	Path      string        `json:"path"`
	Target    string        `json:"target,omitempty"` // symlink target, only for dev-link/broken-link
}

// ScanInstalledPackages discovers all packages in the given root directory.
// root is expected to be TypstLocalDir() or TypstPackageCacheDir().
// defaultSource specifies the source for non-symlink directories (SourceInstalled or SourceCache).
func ScanInstalledPackages(root string, defaultSource PackageSource) ([]InstalledPackage, error) {
	return ScanInstalledPackagesInNamespace(root, "", defaultSource)
}

// ScanInstalledPackagesInNamespace discovers packages under root, optionally restricted to a specific namespace.
// If namespace is empty, all namespaces under root are scanned.
func ScanInstalledPackagesInNamespace(root, namespace string, defaultSource PackageSource) ([]InstalledPackage, error) {
	st, err := os.Stat(root)
	if err != nil || !st.IsDir() {
		return nil, nil
	}

	var namespaces []string
	if namespace != "" {
		cleaned := strings.TrimPrefix(namespace, "@")
		if strings.Contains(cleaned, "..") || strings.ContainsAny(cleaned, "/\\") {
			return nil, fmt.Errorf("invalid namespace %q: path traversal not allowed", namespace)
		}
		nsDir := filepath.Join(root, cleaned)
		if fi, err := os.Stat(nsDir); err == nil && fi.IsDir() {
			namespaces = []string{cleaned}
		} else {
			return nil, nil
		}
	} else {
		entries, err := os.ReadDir(root)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() {
				namespaces = append(namespaces, e.Name())
			}
		}
	}

	var packages []InstalledPackage
	for _, ns := range namespaces {
		nsDir := filepath.Join(root, ns)
		pkgEntries, err := os.ReadDir(nsDir)
		if err != nil {
			continue
		}

		for _, pe := range pkgEntries {
			if !pe.IsDir() {
				continue
			}
			pkgName := pe.Name()
			pkgDir := filepath.Join(nsDir, pkgName)
			verEntries, err := os.ReadDir(pkgDir)
			if err != nil {
				continue
			}

			for _, ve := range verEntries {
				verName := ve.Name()
				fullPath := filepath.Join(pkgDir, verName)

				var src PackageSource
				var target string

				if ve.Type()&os.ModeSymlink != 0 {
					resolved, err := filepath.EvalSymlinks(fullPath)
					if err != nil {
						src = SourceBrokenLink
						target = "(broken target)"
					} else {
						src = SourceDevLink
						target = resolved
					}
				} else if ve.IsDir() {
					src = defaultSource
				} else {
					continue
				}

				packages = append(packages, InstalledPackage{
					Namespace: ns,
					Name:      pkgName,
					Version:   verName,
					Source:    src,
					Path:      fullPath,
					Target:    target,
				})
			}
		}
	}

	SortInstalledPackages(packages)
	return packages, nil
}

// SortInstalledPackages sorts packages by Namespace ASC, Name ASC, and Version DESC (newest first).
func SortInstalledPackages(packages []InstalledPackage) {
	sort.Slice(packages, func(i, j int) bool {
		if packages[i].Namespace != packages[j].Namespace {
			return packages[i].Namespace < packages[j].Namespace
		}
		if packages[i].Name != packages[j].Name {
			return packages[i].Name < packages[j].Name
		}
		cmp := manifest.CompareVersions(packages[i].Version, packages[j].Version)
		if cmp != 0 {
			return cmp > 0 // newer versions first
		}
		return packages[i].Version < packages[j].Version
	})
}
