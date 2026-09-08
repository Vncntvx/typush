package github_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gh "github.com/Vncntvx/typkg/internal/github"
)

func mockServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/typst/packages/contents/packages/preview", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]gh.ContentItem{{Name: "demo", Path: "packages/preview/demo", Type: "dir"}})
	})
	mux.HandleFunc("/repos/typst/packages/pulls", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]gh.Pull{})
	})
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(gh.User{Login: "tester"})
	})
	return httptest.NewServer(mux)
}

func TestPublicEndpoints(t *testing.T) {
	srv := mockServer()
	defer srv.Close()
	c := &gh.Client{HTTP: srv.Client(), BaseURL: srv.URL}
	items, err := c.GetContents(gh.UniverseOwner, gh.UniverseRepo, "packages/preview", "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Name != "demo" {
		t.Fatalf("unexpected items: %+v", items)
	}
	pulls, err := c.ListOpenPulls()
	if err != nil {
		t.Fatal(err)
	}
	if len(pulls) != 0 {
		t.Fatalf("unexpected pulls: %+v", pulls)
	}
}

func TestAuthRequired(t *testing.T) {
	srv := mockServer()
	defer srv.Close()
	anon := &gh.Client{HTTP: srv.Client(), BaseURL: srv.URL}
	if _, err := anon.CurrentUser(); err == nil {
		t.Fatal("expected auth error")
	}
	authed := &gh.Client{HTTP: srv.Client(), BaseURL: srv.URL, Token: "x"}
	u, err := authed.CurrentUser()
	if err != nil {
		t.Fatal(err)
	}
	if u.Login != "tester" {
		t.Fatalf("unexpected user: %+v", u)
	}
}
