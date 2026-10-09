package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Vncntvx/typush/manifest"
)

// Metadata extracts and prints package metadata from typst.toml in dir.
// If field is non-empty, it prints that single field.
// If asJSON is true, it outputs in JSON format.
func Metadata(dir, field string, asJSON bool) error {
	m, err := manifest.Read(dir)
	if err != nil {
		return err
	}

	field = strings.ToLower(strings.TrimSpace(field))

	if field == "" {
		if asJSON {
			return writeJSON(m)
		}

		// Formatted text overview
		p := m.Package
		fmt.Printf("Name:        %s\n", p.Name)
		fmt.Printf("Version:     %s\n", p.Version)
		fmt.Printf("Entrypoint:  %s\n", p.Entrypoint)
		fmt.Printf("Authors:     %s\n", strings.Join(p.Authors, ", "))
		if p.License != nil {
			fmt.Printf("License:     %s\n", *p.License)
		}
		if p.Description != nil {
			fmt.Printf("Description: %s\n", *p.Description)
		}
		if p.Repository != nil {
			fmt.Printf("Repository:  %s\n", *p.Repository)
		}
		if p.Homepage != nil {
			fmt.Printf("Homepage:    %s\n", *p.Homepage)
		}
		if p.Compiler != nil {
			fmt.Printf("Compiler:    %s\n", *p.Compiler)
		}
		if len(p.Categories) > 0 {
			fmt.Printf("Categories:  %s\n", strings.Join(p.Categories, ", "))
		}
		if len(p.Disciplines) > 0 {
			fmt.Printf("Disciplines: %s\n", strings.Join(p.Disciplines, ", "))
		}
		if len(p.Keywords) > 0 {
			fmt.Printf("Keywords:    %s\n", strings.Join(p.Keywords, ", "))
		}
		if len(p.Exclude) > 0 {
			fmt.Printf("Exclude:     %s\n", strings.Join(p.Exclude, ", "))
		}
		if m.Template != nil {
			fmt.Printf("Template Path:   %s\n", m.Template.Path)
			fmt.Printf("Template Entry:  %s\n", m.Template.Entrypoint)
			if m.Template.Thumbnail != nil {
				fmt.Printf("Thumbnail:       %s\n", *m.Template.Thumbnail)
			}
		}
		return nil
	}

	optStr := func(ptr *string) (any, string) {
		if ptr == nil {
			return nil, ""
		}
		return *ptr, *ptr
	}

	sliceVal := func(s []string) (any, string) {
		if asJSON {
			return s, ""
		}
		return s, strings.Join(s, ", ")
	}

	// Single field extraction
	var val any
	var strVal string

	switch field {
	case "name":
		val, strVal = m.Package.Name, m.Package.Name
	case "version":
		val, strVal = m.Package.Version, m.Package.Version
	case "entrypoint":
		val, strVal = m.Package.Entrypoint, m.Package.Entrypoint
	case "authors":
		val, strVal = sliceVal(m.Package.Authors)
	case "license":
		val, strVal = optStr(m.Package.License)
	case "description":
		val, strVal = optStr(m.Package.Description)
	case "homepage":
		val, strVal = optStr(m.Package.Homepage)
	case "repository":
		val, strVal = optStr(m.Package.Repository)
	case "compiler":
		val, strVal = optStr(m.Package.Compiler)
	case "categories":
		val, strVal = sliceVal(m.Package.Categories)
	case "disciplines":
		val, strVal = sliceVal(m.Package.Disciplines)
	case "keywords":
		val, strVal = sliceVal(m.Package.Keywords)
	case "exclude":
		val, strVal = sliceVal(m.Package.Exclude)
	default:
		return fmt.Errorf("unknown metadata field %q (supported: name, version, entrypoint, authors, license, description, homepage, repository, compiler, categories, disciplines, keywords, exclude)", field)
	}

	if asJSON {
		data, err := json.Marshal(val)
		if err != nil {
			return fmt.Errorf("failed to encode field %q as JSON: %w", field, err)
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Println(strVal)
	return nil
}
