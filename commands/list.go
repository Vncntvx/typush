package commands

import (
	"fmt"
	"os"
	"sort"

	"github.com/Vncntvx/typush/util"
)

// ListOptions configures the list command.
type ListOptions struct {
	Namespace string // filter by namespace; empty = all
	All       bool   // include Typst package cache
	Tree      bool   // ASCII tree output
	JSON      bool   // JSON output
}

// List displays installed packages.
func List(opt ListOptions) error {
	localDir, err := util.TypstLocalDir()
	if err != nil {
		return err
	}

	pkgs, err := util.ScanInstalledPackagesInNamespace(localDir, opt.Namespace, util.SourceInstalled)
	if err != nil {
		return err
	}

	if opt.All {
		cacheDir, err := util.TypstPackageCacheDir()
		if err == nil {
			cachePkgs, err := util.ScanInstalledPackagesInNamespace(cacheDir, opt.Namespace, util.SourceCache)
			if err == nil {
				pkgs = append(pkgs, cachePkgs...)
				util.SortInstalledPackages(pkgs)
			}
		}
	}

	if len(pkgs) == 0 {
		if opt.JSON {
			return writeJSON([]util.InstalledPackage{})
		}
		infof("No installed packages found.")
		return nil
	}

	if opt.JSON {
		return writeJSON(pkgs)
	}

	if opt.Tree {
		printPackageTree(pkgs)
		return nil
	}

	w, flush := newTable()
	fmt.Fprintln(w, "NAMESPACE\tPACKAGE\tVERSION\tSOURCE\tPATH")
	for _, p := range pkgs {
		loc := displayPath(p.Path)
		if p.Target != "" {
			loc = fmt.Sprintf("→ %s", displayPath(p.Target))
		}
		fmt.Fprintf(w, "@%s\t%s\t%s\t%s\t%s\n", p.Namespace, p.Name, p.Version, p.Source, loc)
	}
	return flush()
}

func printPackageTree(pkgs []util.InstalledPackage) {
	// Group packages by namespace -> package name -> []InstalledPackage
	nsMap := make(map[string]map[string][]util.InstalledPackage)
	var nsOrder []string
	for _, p := range pkgs {
		if _, ok := nsMap[p.Namespace]; !ok {
			nsMap[p.Namespace] = make(map[string][]util.InstalledPackage)
			nsOrder = append(nsOrder, p.Namespace)
		}
		nsMap[p.Namespace][p.Name] = append(nsMap[p.Namespace][p.Name], p)
	}
	sort.Strings(nsOrder)

	for _, ns := range nsOrder {
		fmt.Fprintf(os.Stdout, "@%s\n", ns)
		pkgMap := nsMap[ns]
		var pkgNames []string
		for name := range pkgMap {
			pkgNames = append(pkgNames, name)
		}
		sort.Strings(pkgNames)

		for i, pkgName := range pkgNames {
			isLastPkg := i == len(pkgNames)-1
			pkgPrefix := "├── "
			childIndent := "│   "
			if isLastPkg {
				pkgPrefix = "└── "
				childIndent = "    "
			}
			fmt.Fprintf(os.Stdout, "%s%s\n", pkgPrefix, pkgName)

			versions := pkgMap[pkgName]
			for j, v := range versions {
				isLastVer := j == len(versions)-1
				verPrefix := "├── "
				if isLastVer {
					verPrefix = "└── "
				}

				suffix := fmt.Sprintf("(%s)", v.Source)
				if v.Source == util.SourceDevLink {
					suffix = fmt.Sprintf("→ %s (%s)", displayPath(v.Target), v.Source)
				} else if v.Source == util.SourceBrokenLink {
					suffix = fmt.Sprintf("→ (broken target) (%s)", v.Source)
				}

				fmt.Fprintf(os.Stdout, "%s%s%s %s\n", childIndent, verPrefix, v.Version, suffix)
			}
		}
	}
}
