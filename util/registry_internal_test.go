package util

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A response larger than the cap is rejected rather than silently truncated into
// a body that cannot decode. This test package is internal only to reach the cap.
func TestFetchUniverseIndexRejectsOversizedBody(t *testing.T) {
	limit := maxIndexBytes
	maxIndexBytes = 512
	t.Cleanup(func() { maxIndexBytes = limit })

	origURL := UniverseIndexURL
	t.Cleanup(func() { UniverseIndexURL = origURL })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("[" + strings.Repeat(" ", 1000) + "]"))
	}))
	t.Cleanup(server.Close)
	UniverseIndexURL = server.URL

	if _, err := fetchUniverseIndex(); err == nil {
		t.Error("expected a body at the cap to be reported as an error")
	}

	maxIndexBytes = 2048
	body, err := fetchUniverseIndex()
	if err != nil {
		t.Fatalf("a body within the cap must be accepted, got: %v", err)
	}
	if len(body) == 0 {
		t.Error("accepted body is empty")
	}
}
