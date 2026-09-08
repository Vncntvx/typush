package lint

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	alertRe    = regexp.MustCompile(`(?m)^\s*>\s*\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]`)
	taskRe     = regexp.MustCompile(`(?m)^\s*[-*+]\s+\[[ xX]\]`)
	mdImageRe  = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	mdLinkRe   = regexp.MustCompile(`\[([^\]]*)\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)
	imgTagRe   = regexp.MustCompile(`(?is)<img\b[^>]*>`)
	anchorRe   = regexp.MustCompile(`(?is)<a\b[^>]*>`)
	attrRe     = regexp.MustCompile(`(?i)(src|href|alt)\s*=\s*("([^"]*)"|'([^']*)'|([^\s>]+))`)
	singleWord = regexp.MustCompile(`^[\p{L}\p{N}\-_]+\w$`)
	singleChar = regexp.MustCompile(`^\w$`)
	// Matches github/gitlab/codeberg file URLs pinned to a default branch.
	defaultBranchURL = regexp.MustCompile(`^https://(github\.com/[^/]+/[^/]+/(?:blob|tree)/(?:main|master)/|raw\.githubusercontent\.com/[^/]+/[^/]+/(?:refs/heads/)?(?:main|master)/|gitlab\.com/[^/]+/[^/]+/-/(?:raw|blob|tree)/(?:main|master)/|codeberg\.org/[^/]+/[^/]+/(?:raw|src)/branch/(?:main|master)/)`)
)

// Readme checks README.md content. Returns diagnostics and the list of
// existing local files linked from it (slash-separated, relative to dir).
func Readme(dir, text string) ([]Diag, []string) {
	var out []Diag
	var linked []string

	seen := map[string]bool{}
	addLinked := func(rel string) {
		rel = filepath.ToSlash(rel)
		if !seen[rel] {
			seen[rel] = true
			linked = append(linked, rel)
		}
	}

	if alertRe.MatchString(text) {
		out = append(out, warnf("readme/unsupported-extension/alert",
			"GFM alert boxes are not supported on Typst Universe."))
	}
	if taskRe.MatchString(text) {
		out = append(out, warnf("readme/unsupported-extension/tasklist",
			"GFM task lists are not supported on Typst Universe."))
	}

	for _, m := range mdImageRe.FindAllStringSubmatch(text, -1) {
		alt, url := m[1], strings.TrimSpace(m[2])
		out = append(out, checkAlt(alt)...)
		if d, rel, ok := checkLink(dir, url); ok {
			out = append(out, d...)
			if rel != "" {
				addLinked(rel)
			}
		}
	}
	// Links: strip image matches first to avoid double counting.
	plain := mdImageRe.ReplaceAllString(text, "")
	for _, m := range mdLinkRe.FindAllStringSubmatch(plain, -1) {
		if d, rel, ok := checkLink(dir, strings.TrimSpace(m[2])); ok {
			out = append(out, d...)
			if rel != "" {
				addLinked(rel)
			}
		}
	}
	for _, tag := range imgTagRe.FindAllString(text, -1) {
		attrs := parseAttrs(tag)
		out = append(out, checkAlt(attrs["alt"])...)
		if src, ok := attrs["src"]; ok {
			if d, rel, done := checkLink(dir, strings.TrimSpace(src)); done {
				out = append(out, d...)
				if rel != "" {
					addLinked(rel)
				}
			}
		}
	}
	for _, tag := range anchorRe.FindAllString(text, -1) {
		if href, ok := parseAttrs(tag)["href"]; ok {
			if d, rel, done := checkLink(dir, strings.TrimSpace(href)); done {
				out = append(out, d...)
				if rel != "" {
					addLinked(rel)
				}
			}
		}
	}
	return out, linked
}

func checkAlt(alt string) []Diag {
	if strings.TrimSpace(alt) == "" {
		return []Diag{errf("readme/image/missing-alt",
			"Missing alternative description for image. Please add a short description to make this image more accessible.")}
	} else if singleChar.MatchString(strings.TrimSpace(alt)) || singleWord.MatchString(strings.TrimSpace(alt)) {
		return []Diag{warnf("readme/image/inadequate-alt",
			fmt.Sprintf("Possibly inadequate alternative description for image: `%s`. Please add a short description to make this image more accessible.", strings.TrimSpace(alt)))}
	}
	return nil
}

// checkLink validates one URL. Returns (diags, localRel, handled).
// Absolute http(s) URLs and #fragments need no local file.
func checkLink(dir, url string) ([]Diag, string, bool) {
	if url == "" || strings.HasPrefix(url, "#") {
		return nil, "", false
	}
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		if defaultBranchURL.MatchString(url) {
			return []Diag{warnf("readme/link/repository-url-permalink",
				fmt.Sprintf("URL links to default branch: `%s`. Consider a tag/release or commit permalink so it always matches this package version.", url))}, "", true
		}
		return nil, "", true
	}
	// mailto: and other schemes are left alone.
	if strings.Contains(url, ":") && !strings.HasPrefix(url, "./") && !strings.HasPrefix(url, "/") && !strings.HasPrefix(url, "../") {
		return nil, "", true
	}
	// Local file: strip fragment/query.
	rel := url
	if i := strings.IndexAny(rel, "#?"); i >= 0 {
		rel = rel[:i]
	}
	rel = strings.TrimPrefix(rel, "./")
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" {
		return []Diag{errf("readme/link/invalid-url", fmt.Sprintf("Invalid url: `%s`", url))}, "", true
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
		return []Diag{errf("readme/link/file-not-found",
			fmt.Sprintf("Linked file not found: `%s`. Make sure to commit all linked files and possibly add them to the `exclude` list.", rel))}, "", true
	}
	return nil, filepath.ToSlash(rel), true
}

func parseAttrs(tag string) map[string]string {
	out := map[string]string{}
	for _, m := range attrRe.FindAllStringSubmatch(tag, -1) {
		v := m[3]
		if v == "" {
			v = m[4]
		}
		if v == "" {
			v = m[5]
		}
		out[strings.ToLower(m[1])] = v
	}
	return out
}
