package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/Vncntvx/typush/manifest"
	"github.com/Vncntvx/typush/util"
)

// InfoResponse is the JSON schema returned by `typush info --json`.
type InfoResponse struct {
	util.UniversePackage
	IsLatest    bool     `json:"isLatest"`
	AllVersions []string `json:"allVersions"`
}

// Info prints detailed metadata for a package in Typst Universe. The spec is
// "name" or "name:version", optionally prefixed with @preview/ or preview/.
func Info(pkgSpec string, opt UniverseOptions) error {
	name, requested, err := parsePackageSpec(pkgSpec)
	if err != nil {
		return err
	}

	idx, err := loadUniverseIndex(opt.Refresh)
	if err != nil {
		return err
	}

	releases := idx.Releases(name)
	if len(releases) == 0 {
		return fmt.Errorf("package %q not found in Typst Universe", name)
	}

	versions := idx.Versions(name)
	latest := releases[0]
	target := latest
	if requested != "" {
		found := false
		for _, r := range releases {
			if r.Version == requested {
				target, found = r, true
				break
			}
		}
		if !found {
			return fmt.Errorf("version %q not found for package %q (available: %s)",
				requested, name, strings.Join(versions, ", "))
		}
	}

	if opt.AsJSON {
		return writeJSON(InfoResponse{
			UniversePackage: target,
			IsLatest:        target.Version == latest.Version,
			AllVersions:     versions,
		})
	}
	printPackageInfo(target, latest.Version, versions)
	return nil
}

// printPackageInfo renders the human-readable package summary to stdout.
func printPackageInfo(p util.UniversePackage, latest string, versions []string) {
	fmt.Printf("Package:      %s\n", p.Name)
	if p.Version == latest {
		fmt.Printf("Version:      %s (latest)\n", p.Version)
	} else {
		fmt.Printf("Version:      %s (latest is %s)\n", p.Version, latest)
	}
	if p.Description != "" {
		fmt.Printf("Description:  %s\n", p.Description)
	}
	if p.License != "" {
		fmt.Printf("License:      %s\n", p.License)
	}
	if len(p.Authors) > 0 {
		fmt.Printf("Authors:      %s\n", strings.Join(p.Authors, ", "))
	}
	if p.Repository != "" {
		fmt.Printf("Repository:   %s\n", p.Repository)
	}
	if p.Homepage != "" {
		fmt.Printf("Homepage:     %s\n", p.Homepage)
	}
	if p.Compiler != "" {
		fmt.Printf("Compiler:     >= %s\n", p.Compiler)
	}
	if len(p.Categories) > 0 {
		fmt.Printf("Categories:   %s\n", strings.Join(p.Categories, ", "))
	}
	if len(p.Disciplines) > 0 {
		fmt.Printf("Disciplines:  %s\n", strings.Join(p.Disciplines, ", "))
	}
	if len(p.Keywords) > 0 {
		fmt.Printf("Keywords:     %s\n", strings.Join(p.Keywords, ", "))
	}
	if p.Template != nil {
		fmt.Printf("Template:     path: %s, entrypoint: %s\n", p.Template.Path, p.Template.Entrypoint)
	}
	if p.UpdatedAt > 0 {
		fmt.Printf("Updated:      %s\n", time.Unix(p.UpdatedAt, 0).Format("2006-01-02 15:04:05 MST"))
	}
	fmt.Printf("Versions:     %s (%d release(s))\n", strings.Join(versions, ", "), len(versions))
	fmt.Printf("\nUsage:\n  #import \"@preview/%s:%s\": *\n", p.Name, p.Version)
}

// parsePackageSpec splits a package spec into a name and an optional version.
// It accepts "name", "name:version", "@preview/name:version" and
// "preview/name:version" and validates both halves against the Universe rules, so
// a name or version Universe could not hold fails before the lookup.
func parsePackageSpec(spec string) (name, version string, err error) {
	s := strings.TrimSpace(spec)
	if s == "" {
		return "", "", fmt.Errorf("package name cannot be empty")
	}

	for _, prefix := range []string{"@preview/", "preview/"} {
		if rest, ok := strings.CutPrefix(s, prefix); ok {
			s = rest
			break
		}
	}

	name, version, _ = strings.Cut(s, ":")
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)

	if err := manifest.ValidateKebabName(name); err != nil {
		return "", "", err
	}
	if version != "" {
		if err := manifest.ValidateVersion(version); err != nil {
			return "", "", err
		}
	}
	return name, version, nil
}
