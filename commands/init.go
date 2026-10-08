package commands

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Vncntvx/typush/manifest"
	"github.com/Vncntvx/typush/util"
)

// Run interactively initializes a new package in dir.
func Init(dir, nameArg string) error {
	if _, err := manifest.Read(dir); err == nil {
		if !util.Confirm("A package manifest already exists. Overwrite?", false) {
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
			v, err := util.PromptLine("Enter the package name", def, false)
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

	var author string
	for {
		v, err := util.PromptLine("Enter the package author", defaultAuthor(), false)
		if err != nil {
			return err
		}
		if err := manifest.ValidateAuthor(v); err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		author = v
		break
	}
	var version string
	for {
		v, err := util.PromptLine("Enter the package version", "0.1.0", false)
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

	catIdx, err := util.MultiSelect("Choose the package category", manifest.Categories)
	if err != nil {
		return err
	}
	var categories []string
	for _, i := range catIdx {
		categories = append(categories, manifest.Categories[i])
	}
	discIdx, err := util.MultiSelect("Choose the package discipline", manifest.Disciplines)
	if err != nil {
		return err
	}
	var disciplines []string
	for _, i := range discIdx {
		disciplines = append(disciplines, manifest.Disciplines[i])
	}

	var entrypoint string
	for {
		v, err := util.PromptLine("Enter the package entrypoint", filepath.Join("src", "lib.typ"), false)
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
	var license string
	for {
		v, err := util.PromptLine("Enter the package license (SPDX expression)", "MIT", true)
		if err != nil {
			return err
		}
		if v == "" {
			fmt.Fprintln(os.Stderr, "License is required by Typst Universe")
			continue
		}
		if err := manifest.ValidateLicense(v); err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		license = v
		break
	}
	description, err := util.PromptLine("Enter the package description", "", true)
	if err != nil {
		return err
	}
	kwLine, err := util.PromptLine("Enter package keywords (separated by commas)", "", true)
	if err != nil {
		return err
	}
	var keywords []string
	for _, k := range strings.Split(kwLine, ",") {
		keywords = append(keywords, strings.TrimSpace(k))
	}
	var homepage *string
	for {
		v, err := util.PromptLine("Enter the package homepage URL", "", true)
		if err != nil {
			return err
		}
		if v == "" {
			break
		}
		u, err := url.Parse(v)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			fmt.Fprintln(os.Stderr, "Invalid URL scheme (must start with http:// or https://)")
			continue
		}
		homepage = &v
		break
	}
	var repository *string
	defRepo := defaultRepo()
	for {
		v, err := util.PromptLine("Enter the package repository URL", defRepo, true)
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
		v, err := util.PromptLine("Enter compiler version", "", true)
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
			License:     &license,
			Description: &description, Keywords: keywords,
			Entrypoint: entrypoint, Homepage: homepage,
			Repository: repository, Compiler: compiler,
		},
	}
	if util.Confirm("Does the package have a template?", false) {
		tplPath, err := util.PromptLine("Enter the template project path", "template", false)
		if err != nil {
			return err
		}
		var tplEntry string
		for {
			// Entrypoint is relative to the template path (official spec).
			v, err := util.PromptLine("Enter the template entrypoint (relative to template path)", "main.typ", false)
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
		thumb, err := util.PromptLine("Enter the template thumbnail path(optional)", "", true)
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

	// Scaffold starter README.md if absent
	readmePath := filepath.Join(dir, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		desc := description
		if desc == "" {
			desc = "A Typst package."
		}
		readmeContent := fmt.Sprintf("# %s\n\n%s\n\n## Usage\n\n```typ\n#import \"@preview/%s:%s\": *\n```\n", name, desc, name, version)
		_ = os.WriteFile(readmePath, []byte(readmeContent), 0o644)
	}

	// Scaffold standard LICENSE if absent and license is MIT
	licensePath := filepath.Join(dir, "LICENSE")
	if _, err := os.Stat(licensePath); os.IsNotExist(err) && strings.EqualFold(license, "MIT") {
		authorName := author
		if i := strings.Index(authorName, "<"); i >= 0 {
			authorName = strings.TrimSpace(authorName[:i])
		}
		mitText := fmt.Sprintf(`MIT License

Copyright (c) %d %s

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
`, time.Now().Year(), authorName)
		_ = os.WriteFile(licensePath, []byte(mitText), 0o644)
	}

	fmt.Fprintln(os.Stderr, "Package initialized.")
	return nil
}

func defaultAuthor() string {
	name, _ := exec.Command("git", "config", "user.name").Output()
	email, _ := exec.Command("git", "config", "user.email").Output()
	n := strings.TrimSpace(string(name))
	e := strings.TrimSpace(string(email))
	if n != "" && e != "" {
		return fmt.Sprintf("%s <%s>", n, e)
	}
	if n != "" {
		return n
	}
	u := os.Getenv("USER")
	if u == "" {
		u = os.Getenv("USERNAME")
	}
	return u
}

func defaultRepo() string {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	remote := strings.TrimSpace(string(out))
	if remote == "" {
		return ""
	}
	if strings.Contains(remote, "github.com") {
		rest := remote
		if i := strings.Index(rest, "github.com"); i >= 0 {
			rest = rest[i+len("github.com"):]
		}
		rest = strings.Trim(rest, ":/")
		rest = strings.TrimSuffix(rest, ".git")
		if rest != "" {
			return "https://github.com/" + rest
		}
	}
	return ""
}
