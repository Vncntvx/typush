package commands

import (
	"fmt"
	"sort"
	"strings"
)

// maxLocationSuffixes is how many occurrence paths the table prints before
// collapsing the rest into "(+N more)".
const maxLocationSuffixes = 2

// Outdated checks @preview dependencies in .typ files against Typst Universe.
func Outdated(path string, opt UniverseOptions) error {
	if path == "" {
		path = "."
	}

	occurrences, err := scanDependencies(path)
	if err != nil {
		return err
	}

	if len(occurrences) == 0 {
		if opt.AsJSON {
			return writeJSON([]PackageDependency{})
		}
		infof("No Typst package dependencies (@preview) found.")
		return nil
	}

	idx, err := loadUniverseIndex(opt.Refresh)
	if err != nil {
		return err
	}
	deps := groupDependencies(occurrences, idx)

	if opt.AsJSON {
		return writeJSON(deps)
	}
	return printOutdatedTable(deps)
}

// printOutdatedTable renders the dependency table and its summary.
func printOutdatedTable(deps []PackageDependency) error {
	w, flush := newTable()
	fmt.Fprintln(w, "PACKAGE\tCURRENT\tLATEST\tSTATUS\tLOCATIONS")

	var outdated, unknown int
	for _, dep := range deps {
		// Anything but a known status renders as unknown, so a missing case can
		// never be presented as up to date.
		var status string
		switch dep.Status {
		case StatusUpToDate:
			status = "Up to date"
		case StatusOutdated:
			status = "Update available"
			outdated++
		case StatusNotInUniverse:
			status = "Unknown (not in Universe)"
			unknown++
		default:
			status = "Unknown"
		}

		latest := dep.LatestVersion
		if latest == "" {
			latest = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			dep.Name,
			strings.Join(dep.CurrentVersions, ", "),
			latest,
			status,
			formatLocations(dep.Occurrences),
		)
	}
	if err := flush(); err != nil {
		return err
	}

	fmt.Println()
	switch {
	case outdated > 0:
		fmt.Printf("%d of %d package dependencies have updates available.\nRun 'typush update' to update them.\n", outdated, len(deps))
	case unknown > 0:
		fmt.Printf("All %d remaining package dependencies are up to date; %d could not be found in Typst Universe.\n", len(deps)-unknown, unknown)
	default:
		fmt.Printf("All %d package dependencies are up to date.\n", len(deps))
	}
	return nil
}

// formatLocations renders occurrence paths as "file:line" entries, truncating
// after maxLocationSuffixes.
func formatLocations(occurrences []ImportOccurrence) string {
	if len(occurrences) == 0 {
		return "-"
	}

	// Sort occurrences by file and line.
	sorted := make([]ImportOccurrence, len(occurrences))
	copy(sorted, occurrences)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].FilePath != sorted[j].FilePath {
			return sorted[i].FilePath < sorted[j].FilePath
		}
		return sorted[i].LineNumber < sorted[j].LineNumber
	})

	shown := min(len(sorted), maxLocationSuffixes)
	locs := make([]string, 0, shown)
	for _, occ := range sorted[:shown] {
		locs = append(locs, fmt.Sprintf("%s:%d", displayPath(occ.FilePath), occ.LineNumber))
	}

	out := strings.Join(locs, ", ")
	if rest := len(sorted) - shown; rest > 0 {
		out += fmt.Sprintf(" (+%d more)", rest)
	}
	return out
}
