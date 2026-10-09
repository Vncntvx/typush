package util_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Vncntvx/typush/util"
)

const sampleIndex = `[
	{"name": "fletcher", "version": "0.4.0", "description": "Diagrams"},
	{"name": "fletcher", "version": "0.5.1", "description": "Diagrams v0.5.1"},
	{"name": "fletcher", "version": "0.5.0", "description": "Diagrams v0.5.0"},
	{"name": "touying", "version": "0.6.1", "description": "Presentations"}
]`

func TestLoadUniverseIndex(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	universeServer(t, sampleIndex, nil)

	idx, err := util.LoadUniverseIndex(true)
	if err != nil {
		t.Fatalf("LoadUniverseIndex failed: %v", err)
	}
	if err := idx.StaleErr(); err != nil {
		t.Errorf("fresh load must not be stale, got %v", err)
	}

	releases := idx.Releases("fletcher")
	if len(releases) != 3 {
		t.Fatalf("expected 3 fletcher releases, got %d", len(releases))
	}
	// Releases must be SemVer descending.
	if releases[0].Version != "0.5.1" || releases[1].Version != "0.5.0" || releases[2].Version != "0.4.0" {
		t.Errorf("releases not sorted descending: %v", []string{
			releases[0].Version, releases[1].Version, releases[2].Version})
	}

	latest, found := idx.Latest("fletcher")
	if !found || latest.Version != "0.5.1" {
		t.Errorf("Latest(fletcher) = (%v, %v); want 0.5.1, true", latest.Version, found)
	}

	if _, found := idx.Latest("nonexistent"); found {
		t.Errorf("Latest(nonexistent) reported found")
	}
}

func TestLoadUniverseIndex_HTTPError(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	origURL := util.UniverseIndexURL
	t.Cleanup(func() { util.UniverseIndexURL = origURL })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	util.UniverseIndexURL = server.URL

	if _, err := util.LoadUniverseIndex(true); err == nil {
		t.Fatal("expected error on HTTP 404, got nil")
	}
}

func TestLoadUniverseIndex_InvalidJSON(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	universeServer(t, `invalid-json`, nil)

	if _, err := util.LoadUniverseIndex(true); err == nil {
		t.Fatal("expected error on invalid JSON, got nil")
	}
}

// A successful fetch must be cached, so a second command run needs no network.
func TestLoadUniverseIndex_UsesCache(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	var hits int
	universeServer(t, sampleIndex, &hits)

	if _, err := util.LoadUniverseIndex(false); err != nil {
		t.Fatalf("first load failed: %v", err)
	}
	if hits != 1 {
		t.Fatalf("expected 1 request on cold cache, got %d", hits)
	}

	if _, err := util.LoadUniverseIndex(false); err != nil {
		t.Fatalf("second load failed: %v", err)
	}
	if hits != 1 {
		t.Fatalf("expected a warm cache to avoid a second request, got %d requests", hits)
	}
}

// A stale cache is used when the remote fetch fails, reporting the cause.
func TestLoadUniverseIndex_StaleFallback(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	origURL := util.UniverseIndexURL
	t.Cleanup(func() { util.UniverseIndexURL = origURL })

	fail := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(sampleIndex))
	}))
	t.Cleanup(server.Close)
	util.UniverseIndexURL = server.URL

	// Seed the cache with a successful fetch.
	fail = false
	if _, err := util.LoadUniverseIndex(true); err != nil {
		t.Fatalf("seeding fetch failed: %v", err)
	}

	// Expire the cache entry by backdating it beyond the TTL.
	fail = true
	expireCache(t, cacheDir)

	idx, err := util.LoadUniverseIndex(false)
	if err != nil {
		t.Fatalf("expected stale cache fallback, got error: %v", err)
	}
	if idx.StaleErr() == nil {
		t.Errorf("expected a reason for the fallback, got nil")
	}
	if _, found := idx.Latest("fletcher"); !found {
		t.Errorf("stale index did not carry package data")
	}
}

// LoadUniverseIndex returns an error when both network and cache are unavailable.
func TestLoadUniverseIndex_NoCacheNoNetwork(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	origURL := util.UniverseIndexURL
	t.Cleanup(func() { util.UniverseIndexURL = origURL })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	util.UniverseIndexURL = server.URL

	idx, err := util.LoadUniverseIndex(true)
	if err == nil {
		t.Fatalf("expected an error, got index %+v", idx)
	}
}

// LoadUniverseIndex writes nothing to stdout or stderr.
func TestLoadUniverseIndex_WritesNothingToStderr(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	origURL := util.UniverseIndexURL
	t.Cleanup(func() { util.UniverseIndexURL = origURL })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	util.UniverseIndexURL = server.URL

	captured, err := os.CreateTemp(t.TempDir(), "stderr-*")
	if err != nil {
		t.Fatal(err)
	}
	origStderr := os.Stderr
	os.Stderr = captured
	t.Cleanup(func() { os.Stderr = origStderr })

	// Both the no-cache failure and the stale fallback paths must stay silent.
	_, _ = util.LoadUniverseIndex(true)

	os.Stderr = origStderr
	if err := captured.Close(); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(captured.Name())
	if err != nil {
		t.Fatal(err)
	}
	if len(content) != 0 {
		t.Errorf("util.LoadUniverseIndex wrote to stderr: %q", content)
	}
}

func TestUniverseIndexSearch(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	universeServer(t, `[
		{"name": "cetz", "version": "0.2.1", "description": "Drawing library for Typst", "categories": ["visualization"]},
		{"name": "cetz", "version": "0.2.2", "description": "Drawing library for Typst", "categories": ["visualization"]},
		{"name": "cetz-plot", "version": "0.1.0", "description": "Plotting add-on for CeTZ", "keywords": ["plot", "chart"]},
		{"name": "fletcher", "version": "0.5.1", "description": "Commutative diagrams", "keywords": ["diagram", "drawing"]},
		{"name": "touying", "version": "0.6.1", "description": "Power of presentations in Typst", "categories": ["presentation"]}
	]`, nil)

	idx, err := util.LoadUniverseIndex(true)
	if err != nil {
		t.Fatalf("LoadUniverseIndex failed: %v", err)
	}

	// Exact name match wins and only the newest release is offered.
	results := idx.Search("cetz", 10)
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}
	if results[0].Name != "cetz" || results[0].Version != "0.2.2" {
		t.Errorf("expected cetz 0.2.2 first, got %s %s", results[0].Name, results[0].Version)
	}
	if results[1].Name != "cetz-plot" {
		t.Errorf("expected cetz-plot second, got %s", results[1].Name)
	}

	// Keyword match.
	if got := idx.Search("diagram", 10); len(got) == 0 || got[0].Name != "fletcher" {
		t.Errorf("expected fletcher for \"diagram\", got %+v", got)
	}

	// Limit.
	if got := idx.Search("cetz", 1); len(got) != 1 {
		t.Errorf("expected 1 result with limit 1, got %d", len(got))
	}

	// An empty query returns no matches.
	if got := idx.Search("   ", 10); len(got) != 0 {
		t.Errorf("expected no results for a blank query, got %d", len(got))
	}
}

// Multi-token queries require all tokens to match for the full score bonus.
func TestUniverseIndexSearch_MultiToken(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	universeServer(t, `[
		{"name": "cetz", "version": "0.2.2", "description": "Drawing library for Typst"}
	]`, nil)

	idx, err := util.LoadUniverseIndex(true)
	if err != nil {
		t.Fatalf("LoadUniverseIndex failed: %v", err)
	}

	all := idx.Search("library typst", 10)
	if len(all) != 1 {
		t.Fatalf("expected the all-token query to match, got %d results", len(all))
	}

	partial := idx.Search("library zebra", 10)
	if len(partial) != 1 {
		t.Fatalf("expected the partially-matching query to still match weakly, got %d results", len(partial))
	}
	if partial[0].Score >= all[0].Score {
		t.Errorf("a partial token match must score below a full one: partial=%d full=%d",
			partial[0].Score, all[0].Score)
	}

	// A query whose tokens match nothing at all must not match.
	if got := idx.Search("zebra yak", 10); len(got) != 0 {
		t.Errorf("expected no results when no token matches, got %d", len(got))
	}
}

// A failed cache write does not fail the command.
func TestLoadUniverseIndex_UnwritableCacheStillSucceeds(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	universeServer(t, sampleIndex, nil)

	// Learn where the cache goes from the file a fetch just wrote, instead of
	// deriving the name here: a renamed cache would otherwise leave this test
	// obstructing a path production never uses, and passing vacuously.
	if _, err := util.LoadUniverseIndex(true); err != nil {
		t.Fatalf("seeding fetch failed: %v", err)
	}
	cacheFile := onlyCacheFile(t, cacheDir)
	if err := os.Remove(cacheFile); err != nil {
		t.Fatal(err)
	}
	// Make the cache path a directory so the final rename must fail.
	if err := os.Mkdir(cacheFile, 0o755); err != nil {
		t.Fatal(err)
	}

	idx, err := util.LoadUniverseIndex(true)
	if err != nil {
		t.Fatalf("a failed cache write must not fail the command, got: %v", err)
	}
	if _, found := idx.Latest("fletcher"); !found {
		t.Error("index data missing after a failed cache write")
	}

	entries, err := os.ReadDir(filepath.Dir(cacheFile))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temporary file left behind after a failed cache write: %s", e.Name())
		}
	}
}

// A response body that is not a usable index must not replace the cached copy:
// with the network gone, that copy is the only source left.
func TestLoadUniverseIndex_BadBodyKeepsGoodCache(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	origURL := util.UniverseIndexURL
	t.Cleanup(func() { util.UniverseIndexURL = origURL })

	body := sampleIndex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	util.UniverseIndexURL = server.URL

	if _, err := util.LoadUniverseIndex(true); err != nil {
		t.Fatalf("seeding fetch failed: %v", err)
	}
	cached := onlyCacheFile(t, cacheDir)

	// The CDN now answers 200 with something that is not an index at all.
	body = "<html>gateway error</html>"
	expireCache(t, cacheDir)

	idx, err := util.LoadUniverseIndex(false)
	if err != nil {
		t.Fatalf("an unusable body must fall back to the cached copy, got: %v", err)
	}
	if idx.StaleErr() == nil {
		t.Error("expected the unusable body to be reported as the reason")
	}
	if _, found := idx.Latest("fletcher"); !found {
		t.Error("fallback index is missing the cached package data")
	}

	data, err := os.ReadFile(cached)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != sampleIndex {
		t.Error("the unusable body replaced the cached index")
	}

	// With the CDN gone the surviving copy still has to serve a load.
	server.Close()
	if _, err := util.LoadUniverseIndex(false); err != nil {
		t.Errorf("offline load after an unusable body failed: %v", err)
	}
}

// onlyCacheFile returns the one cache file present in dir, so tests act on the
// path production actually wrote.
func onlyCacheFile(t *testing.T, dir string) string {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(dir, "typush", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected exactly one cache file in %s, found %v", dir, matches)
	}
	return matches[0]
}

// expireCache backdates the cache file so it fails the freshness check.
func expireCache(t *testing.T, dir string) {
	t.Helper()

	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(onlyCacheFile(t, dir), old, old); err != nil {
		t.Fatal(err)
	}
}
