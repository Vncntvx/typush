package commands_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Vncntvx/typush/commands"
	"github.com/Vncntvx/typush/util"
)

func TestWarnIfExists(t *testing.T) {
	origURL := util.UniverseIndexURL
	defer func() { util.UniverseIndexURL = origURL }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[
			{"name": "fletcher", "version": "0.5.0"},
			{"name": "fletcher", "version": "0.5.1"}
		]`)
	}))
	defer server.Close()

	util.UniverseIndexURL = server.URL

	// 1. Package not found
	stderr, err := captureStderr(t, func() error {
		return commands.WarnIfExists("brand-new-pkg", "0.1.0")
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr, "WARN: package `brand-new-pkg` is not found in the Universe") {
		t.Errorf("expected package not found warning, got: %q", stderr)
	}

	// 2. Version already exists
	stderr, err = captureStderr(t, func() error {
		return commands.WarnIfExists("fletcher", "0.5.0")
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr, "WARN: version `0.5.0` already exists in the Universe") {
		t.Errorf("expected version already exists warning, got: %q", stderr)
	}

	// 3. Package exists, new version -> no warning
	stderr, err = captureStderr(t, func() error {
		return commands.WarnIfExists("fletcher", "0.6.0")
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Errorf("expected no warning for new version of existing package, got: %q", stderr)
	}
}
