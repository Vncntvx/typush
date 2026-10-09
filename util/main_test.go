package util_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Vncntvx/typush/util"
)

// TestMain isolates the cache directory to a temporary location for tests.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "typush-util-test-cache-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp cache dir: %v\n", err)
		os.Exit(1)
	}
	if err := os.Setenv("XDG_CACHE_HOME", dir); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set XDG_CACHE_HOME: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// universeServer starts an httptest server serving body and points
// UniverseIndexURL at it for the duration of the test.
func universeServer(t *testing.T, body string, hits *int) *httptest.Server {
	t.Helper()

	origURL := util.UniverseIndexURL
	t.Cleanup(func() { util.UniverseIndexURL = origURL })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits != nil {
			*hits++
		}
		if r.Header.Get("User-Agent") != util.AppName {
			t.Errorf("expected User-Agent %q, got %q", util.AppName, r.Header.Get("User-Agent"))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	t.Cleanup(server.Close)

	util.UniverseIndexURL = server.URL
	return server
}
