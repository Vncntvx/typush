package commands

import (
	"fmt"
	"strings"
)

// Search queries Typst Universe for packages matching query and prints results.
func Search(query string, limit int, opt UniverseOptions) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return fmt.Errorf("search query cannot be empty")
	}
	if limit <= 0 {
		limit = DefaultSearchLimit
	}

	idx, err := loadUniverseIndex(opt.Refresh)
	if err != nil {
		return err
	}

	results := idx.Search(query, limit)
	if opt.AsJSON {
		return writeJSON(results)
	}

	if len(results) == 0 {
		infof("No packages found matching %q.", query)
		return nil
	}

	w, flush := newTable()
	fmt.Fprintln(w, "NAME\tVERSION\tCATEGORIES\tDESCRIPTION")
	for _, r := range results {
		categories := strings.Join(r.Categories, ", ")
		if categories == "" {
			categories = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			r.Name, r.Version, categories, truncateRunes(r.Description, maxDescriptionRunes))
	}
	if err := flush(); err != nil {
		return err
	}

	fmt.Printf("\nFound %d package(s)\n", len(results))
	return nil
}
