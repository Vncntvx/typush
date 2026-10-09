package commands_test

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/Vncntvx/typush/commands"
)

// TestMain isolates the cache directory to a temporary location for tests.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "typush-commands-test-cache-")
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

// useUniverse serves body as the Universe index and counts server requests.
func useUniverse(t *testing.T, body string) *int {
	t.Helper()

	hits := new(int)
	commands.UseUniverseServerForTest(t, func(w http.ResponseWriter, r *http.Request) {
		*hits++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	})
	return hits
}

// useFailingUniverse points the index URL at a failing server with an empty cache.
func useFailingUniverse(t *testing.T) {
	t.Helper()

	commands.UseUniverseServerForTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
}
