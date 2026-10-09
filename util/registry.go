package util

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Vncntvx/typush/manifest"
)

// UniverseIndexURL is the official Typst Universe static CDN index.
//
// It is a variable so tests can point the client at an httptest server.
var UniverseIndexURL = "https://packages.typst.org/preview/index.json"

// UniversePackageDownloadURL returns the archive download URL for a preview package.
func UniversePackageDownloadURL(name, version string) string {
	baseURL := strings.TrimSuffix(UniverseIndexURL, "/index.json")
	return fmt.Sprintf("%s/%s-%s.tar.gz", baseURL, name, version)
}

// FetchPackageArchive downloads the .tar.gz archive for the requested package version.
// The caller is responsible for closing the returned ReadCloser.
func FetchPackageArchive(name, version string) (io.ReadCloser, error) {
	downloadURL := UniversePackageDownloadURL(name, version)
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", AppName)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download package archive: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("failed to download package archive (HTTP %d)", resp.StatusCode)
	}
	return resp.Body, nil
}

const (
	universeCacheTTL    = 15 * time.Minute
	universeHTTPTimeout = 15 * time.Second
	// anyCacheAge accepts a cached index of any age for offline fallback.
	anyCacheAge = 0
)

// maxIndexBytes caps how much of the index response is read. A response that hits
// the cap may be truncated, so fetchUniverseIndex fails it rather than returning a
// partial body. It is a variable so tests can exercise the cap without producing a
// 64 MiB body.
var maxIndexBytes = 64 << 20

// Scoring weights for Search, from strongest to weakest signal.
const (
	scoreNameExact      = 1000
	scoreNamePrefix     = 150
	scoreNameSubstring  = 60
	scoreKeywordExact   = 40
	scoreCategory       = 30
	scoreAllTokens      = 25
	scoreKeywordPartial = 20
	scoreDescription    = 20
	scoreAuthor         = 10
	scorePerToken       = 10
)

// UniversePackage represents a single package release from the Typst Universe CDN.
type UniversePackage struct {
	Name        string                `json:"name"`
	Version     string                `json:"version"`
	Entrypoint  string                `json:"entrypoint"`
	Authors     []string              `json:"authors"`
	License     string                `json:"license"`
	Description string                `json:"description"`
	Repository  string                `json:"repository,omitempty"`
	Homepage    string                `json:"homepage,omitempty"`
	Keywords    []string              `json:"keywords,omitempty"`
	Categories  []string              `json:"categories,omitempty"`
	Disciplines []string              `json:"disciplines,omitempty"`
	Compiler    string                `json:"compiler,omitempty"`
	Exclude     []string              `json:"exclude,omitempty"`
	Template    *UniverseTemplateInfo `json:"template,omitempty"`
	UpdatedAt   int64                 `json:"updatedAt"`
}

// UniverseTemplateInfo represents template metadata when a package is also a template.
type UniverseTemplateInfo struct {
	Path       string `json:"path"`
	Entrypoint string `json:"entrypoint"`
	Thumbnail  string `json:"thumbnail,omitempty"`
}

// UniverseSearchResult bundles a package with its search match score.
type UniverseSearchResult struct {
	UniversePackage
	Score int `json:"score"`
}

// UniverseIndex is an in-memory view of the Universe index, grouped by package
// name with each group's releases sorted newest first.
//
// Releases, Latest and Versions are in-memory map lookups on the loaded index.
type UniverseIndex struct {
	byName map[string][]UniversePackage

	// staleErr holds the reason the index in use came from an outdated cache. It is
	// nil when the index is up to date.
	staleErr error
}

// StaleErr returns the error that forced a fallback to an outdated cache copy,
// or nil when the index is up to date.
func (idx *UniverseIndex) StaleErr() error { return idx.staleErr }

// LoadUniverseIndex returns the Universe index, using the disk cache when valid
// and falling back to the CDN.
//
// When no fresh index can be produced (the request failed, or the response body
// did not decode), the cached copy is returned at any age and StaleErr reports
// the cause. A body that does not decode never reaches the cache, so the offline
// fallback survives one bad response.
func LoadUniverseIndex(refresh bool) (*UniverseIndex, error) {
	cachePath, err := universeCachePath()
	if err != nil {
		cachePath = "" // no cache location: the CDN is the only source
	}

	if !refresh && cachePath != "" {
		if data, ok := readIndexCache(cachePath, universeCacheTTL); ok {
			if idx, err := decodeUniverseIndex(data); err == nil {
				return idx, nil
			}
		}
	}

	data, fetchErr := fetchUniverseIndex()
	if fetchErr != nil {
		if idx, ok := staleIndexCache(cachePath, fetchErr); ok {
			return idx, nil
		}
		return nil, fetchErr
	}

	idx, decodeErr := decodeUniverseIndex(data)
	if decodeErr != nil {
		if stale, ok := staleIndexCache(cachePath, decodeErr); ok {
			return stale, nil
		}
		return nil, decodeErr
	}

	if cachePath != "" {
		// A failed cache write must not fail the command.
		_ = writeIndexCache(cachePath, data)
	}
	return idx, nil
}

// staleIndexCache decodes the cached index of any age, recording cause as the
// reason a fresh copy was unusable.
func staleIndexCache(cachePath string, cause error) (*UniverseIndex, bool) {
	if cachePath == "" {
		return nil, false
	}
	data, ok := readIndexCache(cachePath, anyCacheAge)
	if !ok {
		return nil, false
	}
	idx, err := decodeUniverseIndex(data)
	if err != nil {
		return nil, false
	}
	idx.staleErr = cause
	return idx, true
}

// Releases returns every release of a package, newest first. The result is the
// index's own slice and must not be modified.
func (idx *UniverseIndex) Releases(name string) []UniversePackage {
	return idx.byName[name]
}

// Latest returns the newest release of a package.
func (idx *UniverseIndex) Latest(name string) (UniversePackage, bool) {
	releases := idx.byName[name]
	if len(releases) == 0 {
		return UniversePackage{}, false
	}
	return releases[0], true
}

// Versions returns every version of a package, newest first.
func (idx *UniverseIndex) Versions(name string) []string {
	releases := idx.byName[name]
	versions := make([]string, 0, len(releases))
	for _, r := range releases {
		versions = append(versions, r.Version)
	}
	return versions
}

// Search scores every package against query and returns the best matches,
// strongest first and ties broken by name. Only the newest release of each
// package is considered.
func (idx *UniverseIndex) Search(query string, limit int) []UniverseSearchResult {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	tokens := strings.Fields(q)

	results := make([]UniverseSearchResult, 0, len(idx.byName))
	for _, releases := range idx.byName {
		latest := releases[0] // groups are sorted newest first
		if score := scorePackage(latest, q, tokens); score > 0 {
			results = append(results, UniverseSearchResult{UniversePackage: latest, Score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].Name < results[j].Name
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

// scorePackage weighs a single package against a lowercased query.
func scorePackage(p UniversePackage, query string, tokens []string) int {
	name := strings.ToLower(p.Name)

	score := 0
	switch {
	case name == query:
		score += scoreNameExact
	case strings.HasPrefix(name, query):
		score += scoreNamePrefix
	case strings.Contains(name, query):
		score += scoreNameSubstring
	}

	score += keywordScore(p.Keywords, query)
	if matchesAny(p.Categories, query) || matchesAny(p.Disciplines, query) {
		score += scoreCategory
	}
	if strings.Contains(strings.ToLower(p.Description), query) {
		score += scoreDescription
	}
	if matchesAny(p.Authors, query) {
		score += scoreAuthor
	}

	// Fall back to token matching if whole-query matching produced no hits.
	if score == 0 && len(tokens) > 1 {
		score += tokenScore(p, tokens)
	}
	return score
}

// keywordScore returns the strongest keyword hit, preferring an exact match.
func keywordScore(keywords []string, query string) int {
	best := 0
	for _, kw := range keywords {
		k := strings.ToLower(kw)
		switch {
		case k == query:
			return scoreKeywordExact
		case strings.Contains(k, query):
			best = scoreKeywordPartial
		}
	}
	return best
}

// tokenScore counts how many tokens match at least one field.
func tokenScore(p UniversePackage, tokens []string) int {
	name := strings.ToLower(p.Name)
	description := strings.ToLower(p.Description)

	matched := 0
	for _, token := range tokens {
		if strings.Contains(name, token) ||
			strings.Contains(description, token) ||
			matchesAny(p.Keywords, token) {
			matched++
		}
	}
	if matched == len(tokens) {
		return scoreAllTokens
	}
	return scorePerToken * matched
}

// matchesAny reports whether any value contains query.
func matchesAny(values []string, query string) bool {
	for _, v := range values {
		if strings.Contains(strings.ToLower(v), query) {
			return true
		}
	}
	return false
}

// universeCachePath returns the cache file for the current UniverseIndexURL.
func universeCachePath() (string, error) {
	dir, err := TypstCacheDir()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(UniverseIndexURL))
	return filepath.Join(dir, fmt.Sprintf("universe-index-%x.json", sum[:8])), nil
}

// readIndexCache returns the cached index body if it exists and is younger than
// maxAge. A maxAge <= 0 disables the expiration check.
func readIndexCache(path string, maxAge time.Duration) ([]byte, bool) {
	if maxAge > 0 {
		fi, err := os.Stat(path)
		if err != nil || time.Since(fi.ModTime()) >= maxAge {
			return nil, false
		}
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return nil, false
	}
	return data, true
}

// writeIndexCache persists data at the cache path, creating the cache directory
// if needed. The write itself is delegated to WriteFileAtomic so the atomic
// replace lives in exactly one place.
func writeIndexCache(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return WriteFileAtomic(path, data)
}

// fetchUniverseIndex downloads the raw index body from the CDN.
func fetchUniverseIndex() ([]byte, error) {
	client := &http.Client{Timeout: universeHTTPTimeout}
	req, err := http.NewRequest(http.MethodGet, UniverseIndexURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", AppName)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Universe index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Universe index returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxIndexBytes)+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read Universe index response: %w", err)
	}
	if len(body) > maxIndexBytes {
		return nil, fmt.Errorf("Universe index is larger than %d bytes", maxIndexBytes)
	}
	return body, nil
}

// decodeUniverseIndex parses an index body and groups its releases by name,
// newest first.
func decodeUniverseIndex(data []byte) (*UniverseIndex, error) {
	var pkgs []UniversePackage
	if err := json.Unmarshal(data, &pkgs); err != nil {
		return nil, fmt.Errorf("failed to parse Universe index: %w", err)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("Universe index is empty")
	}

	byName := make(map[string][]UniversePackage, len(pkgs))
	for _, p := range pkgs {
		byName[p.Name] = append(byName[p.Name], p)
	}
	for name, releases := range byName {
		sort.Slice(releases, func(i, j int) bool {
			return manifest.CompareVersions(releases[i].Version, releases[j].Version) > 0
		})
		byName[name] = releases
	}
	return &UniverseIndex{byName: byName}, nil
}
