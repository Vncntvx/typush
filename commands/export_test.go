package commands

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Vncntvx/typush/util"
)

// UseUniverseServerForTest points the Universe index at handler and isolates the
// cache directory for the duration of the test. It is declared in the package
// under test so the internal and the external test packages share one bootstrap
// instead of each rebuilding the same isolation.
func UseUniverseServerForTest(t *testing.T, handler http.HandlerFunc) {
	t.Helper()

	origURL := util.UniverseIndexURL
	t.Cleanup(func() { util.UniverseIndexURL = origURL })

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	util.UniverseIndexURL = server.URL
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
}
