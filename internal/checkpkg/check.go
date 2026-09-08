package checkpkg

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Vncntvx/typush-go/internal/manifest"
	"github.com/Vncntvx/typush-go/internal/walker"
)

// Run performs a real validation (the Rust original was a stub).
func Run(dir string) error {
	m, err := manifest.Read(dir)
	if err != nil {
		return err
	}
	warns := 0
	warn := func(f string, a ...any) {
		warns++
		fmt.Fprintf(os.Stderr, "WARN: "+f+"\n", a...)
	}
	if m.Package.Description == nil || *m.Package.Description == "" {
		warn("package.description is empty")
	}
	if len(m.Package.Keywords) == 0 {
		warn("package.keywords is empty")
	}
	if m.Package.License == nil || *m.Package.License == "" {
		warn("package.license is empty (Universe requires a license or README link)")
	}
	if _, err := os.Stat(filepath.Join(dir, "README.md")); err != nil {
		warn("README.md not found")
	}
	if _, err := os.Stat(filepath.Join(dir, m.Package.Entrypoint)); err != nil {
		return fmt.Errorf("entrypoint %q not found: %w", m.Package.Entrypoint, err)
	}
	if m.Template != nil {
		tp := filepath.Join(dir, m.Template.Path)
		if _, err := os.Stat(tp); err != nil {
			return fmt.Errorf("template.path %q not found: %w", m.Template.Path, err)
		}
		if _, err := os.Stat(filepath.Join(dir, m.Template.Entrypoint)); err != nil {
			return fmt.Errorf("template.entrypoint %q not found: %w", m.Template.Entrypoint, err)
		}
	}
	if _, err := walker.ListInstall(dir, m.Package.Exclude); err != nil {
		return err
	}
	if warns == 0 {
		fmt.Fprintln(os.Stderr, "No issues found")
	} else {
		fmt.Fprintf(os.Stderr, "No errors found (%d warning(s))\n", warns)
	}
	return nil
}
