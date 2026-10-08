package util_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/Vncntvx/typush/util"
)

func TestFetchUniverseIndex(t *testing.T) {
	origURL := util.UniverseIndexURL
	defer func() { util.UniverseIndexURL = origURL }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "typush" {
			t.Errorf("expected User-Agent 'typush', got %q", r.Header.Get("User-Agent"))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[
			{"name": "fletcher", "version": "0.5.0"},
			{"name": "fletcher", "version": "0.5.1"},
			{"name": "touying", "version": "0.6.1"}
		]`)
	}))
	defer server.Close()

	util.UniverseIndexURL = server.URL

	index, err := util.FetchUniverseIndex()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedFletcher := []string{"0.5.0", "0.5.1"}
	if !reflect.DeepEqual(index["fletcher"], expectedFletcher) {
		t.Errorf("expected fletcher versions %v, got %v", expectedFletcher, index["fletcher"])
	}

	expectedTouying := []string{"0.6.1"}
	if !reflect.DeepEqual(index["touying"], expectedTouying) {
		t.Errorf("expected touying versions %v, got %v", expectedTouying, index["touying"])
	}

	vers, found, err := util.FetchRemotePackageVersions("fletcher")
	if err != nil || !found || !reflect.DeepEqual(vers, expectedFletcher) {
		t.Errorf("FetchRemotePackageVersions(fletcher) = (%v, %v, %v); expected (%v, true, nil)", vers, found, err, expectedFletcher)
	}

	vers, found, err = util.FetchRemotePackageVersions("nonexistent")
	if err != nil || found || len(vers) != 0 {
		t.Errorf("FetchRemotePackageVersions(nonexistent) = (%v, %v, %v); expected (nil, false, nil)", vers, found, err)
	}
}

func TestFetchUniverseIndex_HttpError(t *testing.T) {
	origURL := util.UniverseIndexURL
	defer func() { util.UniverseIndexURL = origURL }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	util.UniverseIndexURL = server.URL

	_, err := util.FetchUniverseIndex()
	if err == nil {
		t.Fatal("expected error on HTTP 404, got nil")
	}
}

func TestFetchUniverseIndex_InvalidJSON(t *testing.T) {
	origURL := util.UniverseIndexURL
	defer func() { util.UniverseIndexURL = origURL }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `invalid-json`)
	}))
	defer server.Close()

	util.UniverseIndexURL = server.URL

	_, err := util.FetchUniverseIndex()
	if err == nil {
		t.Fatal("expected error on invalid JSON, got nil")
	}
}
