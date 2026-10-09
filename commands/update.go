package commands

import (
	"fmt"
	"sort"
)

// UpdateOptions configures Update. A struct rather than positional arguments so
// the two booleans cannot be transposed at a call site, mirroring
// checker.Options and GenerateOptions.
type UpdateOptions struct {
	// Dir is the directory scanned when File is empty.
	Dir string
	// File limits the update to a single .typ file.
	File string
	// Packages limits the update to these package names. Empty updates everything.
	Packages []string
	// DryRun previews the line changes without writing files.
	DryRun bool
	// Refresh ignores the cached Universe index and fetches a fresh one.
	Refresh bool
}

// Update rewrites @preview package versions in .typ files to the latest versions
// on Typst Universe.
func Update(opt UpdateOptions) error {
	target := opt.Dir
	if opt.File != "" {
		target = opt.File
	}
	if target == "" {
		target = "."
	}

	occurrences, err := scanDependencies(target)
	if err != nil {
		return err
	}
	if len(occurrences) == 0 {
		if opt.DryRun {
			previewNote("no package dependencies found to update")
		} else {
			infof("No package dependencies found to update.")
		}
		return nil
	}

	idx, err := loadUniverseIndex(opt.Refresh)
	if err != nil {
		return err
	}
	deps := groupDependencies(occurrences, idx)

	plan, warnings := selectUpdates(deps, opt.Packages)
	for _, warning := range warnings {
		warnf("%s", warning)
	}

	if len(plan) == 0 {
		if opt.DryRun {
			previewNote("all dependencies are already up to date")
		} else {
			infof("All package dependencies are already up to date.")
		}
		return nil
	}

	targets := planTargets(plan)
	files := plannedFiles(plan)

	if opt.DryRun {
		rewritten, err := planRewrites(files, targets)
		if err != nil {
			return err
		}
		if rewritten.Occurrences() == 0 {
			previewNote("all package dependencies are already up to date")
			return nil
		}
		items := make([]string, 0, len(rewritten.Edits))
		for _, e := range rewritten.Edits {
			items = append(items, fmt.Sprintf("%s:%d:\n- %s\n+ %s",
				displayPath(e.FilePath), e.LineNumber, e.OldLine, e.NewLine))
		}
		previewItems("dependency updates", items)
		previewNote("%d occurrence(s) in %d file(s) would be updated",
			rewritten.Occurrences(), len(rewritten.Files))
		return nil
	}

	rewritten, err := applyRewrites(files, targets)
	if err != nil {
		return err
	}
	if rewritten.Occurrences() == 0 {
		infof("All package dependencies are already up to date.")
		return nil
	}
	for _, u := range plan {
		if rewritten.Hits[u.Name] > 0 {
			infof("✓ Updated %s to %s", u.Name, u.Version)
		}
	}
	infof("Updated %d occurrence(s) across %d file(s).",
		rewritten.Occurrences(), len(rewritten.Files))
	return nil
}

// planTargets maps every package in the plan to the version to write for it.
func planTargets(plan []plannedUpdate) map[string]string {
	targets := make(map[string]string, len(plan))
	for _, u := range plan {
		targets[u.Name] = u.Version
	}
	return targets
}

// plannedFiles returns every distinct file the plan rewrites, sorted.
func plannedFiles(plan []plannedUpdate) []string {
	seen := make(map[string]struct{})
	var files []string
	for _, u := range plan {
		for _, path := range u.Files {
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			files = append(files, path)
		}
	}
	sort.Strings(files)
	return files
}

// selectUpdates picks the dependencies that need rewriting. It returns the plan,
// ordered by package name, plus any warnings to print, in the order the user
// should read them.
func selectUpdates(deps []PackageDependency, only []string) ([]plannedUpdate, []string) {
	filter := make(map[string]struct{}, len(only))
	for _, name := range only {
		filter[name] = struct{}{}
	}

	plan := make([]plannedUpdate, 0, len(deps))
	var warnings []string
	found := make(map[string]bool, len(filter))

	for _, dep := range deps {
		if len(filter) > 0 {
			if _, ok := filter[dep.Name]; !ok {
				continue
			}
			found[dep.Name] = true
		}

		switch dep.Status {
		case StatusOutdated:
			plan = append(plan, plannedUpdate{
				Name:    dep.Name,
				Version: dep.LatestVersion,
				Files:   occurrenceFiles(dep.Occurrences),
			})
		case StatusNotInUniverse:
			warnings = append(warnings, fmt.Sprintf("package %q was not found in Typst Universe, skipping", dep.Name))
		}
	}
	sort.Slice(plan, func(i, j int) bool { return plan[i].Name < plan[j].Name })

	// Report requested packages the scan never saw, in the user's own order.
	reported := make(map[string]bool, len(only))
	for _, name := range only {
		if reported[name] {
			continue
		}
		reported[name] = true
		if !found[name] {
			warnings = append(warnings, fmt.Sprintf("package %q not found in the scanned files, skipping", name))
		}
	}
	return plan, warnings
}

// occurrenceFiles returns the distinct files of these occurrences, sorted.
func occurrenceFiles(occurrences []ImportOccurrence) []string {
	seen := make(map[string]struct{}, len(occurrences))
	files := make([]string, 0, len(occurrences))
	for _, occ := range occurrences {
		if _, ok := seen[occ.FilePath]; ok {
			continue
		}
		seen[occ.FilePath] = struct{}{}
		files = append(files, occ.FilePath)
	}
	sort.Strings(files)
	return files
}
