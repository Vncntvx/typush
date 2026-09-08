package initpkg

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/Vncntvx/typkg/internal/cliutil"
	"github.com/Vncntvx/typkg/internal/manifest"
)

// Run interactively initializes a new package in dir.
func Run(dir, nameArg string) error {
	if _, err := manifest.Read(dir); err == nil {
		if !cliutil.Confirm("A package manifest already exists. Overwrite?", false) {
			return fmt.Errorf("aborted")
		}
	}
	fmt.Fprintln(os.Stderr, "Initializing a new package...")

	var name string
	if nameArg != "" {
		if err := manifest.ValidateName(nameArg); err != nil {
			return err
		}
		fmt.Printf("Package name: %s\n", nameArg)
		name = nameArg
	} else {
		def := ""
		if abs, err := filepath.Abs(dir); err == nil {
			def = filepath.Base(abs)
			if err := manifest.ValidateName(def); err != nil {
				def = ""
			}
		}
		for {
			v, err := cliutil.PromptLine("Enter the package name", def, false)
			if err != nil {
				return err
			}
			if err := manifest.ValidateName(v); err != nil {
				fmt.Fprintln(os.Stderr, "Invalid package name")
				continue
			}
			name = v
			break
		}
	}

	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}
	author, err := cliutil.PromptLine("Enter the package author", username, false)
	if err != nil {
		return err
	}
	var version string
	for {
		v, err := cliutil.PromptLine("Enter the package version", "0.1.0", false)
		if err != nil {
			return err
		}
		if err := manifest.ValidateVersion(v); err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		version = v
		break
	}

	catIdx, err := cliutil.MultiSelect("Choose the package category", manifest.Categories)
	if err != nil {
		return err
	}
	var categories []string
	for _, i := range catIdx {
		categories = append(categories, manifest.Categories[i])
	}
	discIdx, err := cliutil.MultiSelect("Choose the package discipline", manifest.Disciplines)
	if err != nil {
		return err
	}
	var disciplines []string
	for _, i := range discIdx {
		disciplines = append(disciplines, manifest.Disciplines[i])
	}

	var entrypoint string
	for {
		v, err := cliutil.PromptLine("Enter the package entrypoint", filepath.Join("src", "lib.typ"), false)
		if err != nil {
			return err
		}
		if err := manifest.ValidateEntrypoint(v); err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		entrypoint = v
		break
	}
	description, err := cliutil.PromptLine("Enter the package description", "", true)
	if err != nil {
		return err
	}
	kwLine, err := cliutil.PromptLine("Enter the package keywords(separated by comma)", "", true)
	if err != nil {
		return err
	}
	var keywords []string
	for _, k := range strings.Split(kwLine, ",") {
		keywords = append(keywords, strings.TrimSpace(k))
	}
	var homepage *string
	for {
		v, err := cliutil.PromptLine("Enter the package homepage URL", "", true)
		if err != nil {
			return err
		}
		if v == "" {
			break
		}
		u, err := url.Parse(v)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			fmt.Fprintln(os.Stderr, "Invalid URL scheme")
			continue
		}
		homepage = &v
		break
	}
	var repository *string
	for {
		v, err := cliutil.PromptLine("Enter the package repository URL", "", true)
		if err != nil {
			return err
		}
		if v == "" {
			break
		}
		u, err := url.Parse(v)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "git") {
			// allow git@ / ssh style too
			if !strings.Contains(v, "@") && !strings.HasPrefix(v, "git") {
				fmt.Fprintln(os.Stderr, "Invalid URL scheme")
				continue
			}
		}
		_ = u
		repository = &v
		break
	}
	var compiler *string
	for {
		v, err := cliutil.PromptLine("Enter compiler version", "", true)
		if err != nil {
			return err
		}
		if v == "" {
			break
		}
		if err := manifest.ValidateCompiler(v); err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		compiler = &v
		break
	}

	m := &manifest.Manifest{
		Package: manifest.PackageInfo{
			Name: name, Authors: []string{author}, Version: version,
			Categories: categories, Disciplines: disciplines,
			Description: &description, Keywords: keywords,
			Entrypoint: entrypoint, Homepage: homepage,
			Repository: repository, Compiler: compiler,
		},
	}
	if cliutil.Confirm("Does the package have a template?", false) {
		tplPath, err := cliutil.PromptLine("Enter the template project path", "template", false)
		if err != nil {
			return err
		}
		var tplEntry string
		for {
			// Entrypoint is relative to the template path (official spec).
			v, err := cliutil.PromptLine("Enter the template entrypoint (relative to template path)", "main.typ", false)
			if err != nil {
				return err
			}
			if err := manifest.ValidateEntrypoint(v); err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}
			tplEntry = v
			break
		}
		thumb, err := cliutil.PromptLine("Enter the template thumbnail path(optional)", "", true)
		if err != nil {
			return err
		}
		ti := &manifest.TemplateInfo{Path: tplPath, Entrypoint: tplEntry}
		if thumb != "" {
			ti.Thumbnail = &thumb
		}
		m.Template = ti
	}
	if err := manifest.Write(dir, m); err != nil {
		return err
	}
	ep := filepath.Join(dir, filepath.FromSlash(entrypoint))
	if err := os.MkdirAll(filepath.Dir(ep), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(ep); os.IsNotExist(err) {
		if err := os.WriteFile(ep, []byte("// "+name+"\n"), 0o644); err != nil {
			return err
		}
	}
	fmt.Fprintln(os.Stderr, "Initialized.")
	return nil
}
