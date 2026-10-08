package util

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// UniverseIndexURL is the official Typst Universe static CDN index.
var UniverseIndexURL = "https://packages.typst.org/preview/index.json"

type indexItem struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// FetchUniverseIndex retrieves all packages from the official Typst Universe CDN.
// It returns a map from package name to a slice of its available versions.
func FetchUniverseIndex() (map[string][]string, error) {
	client := &http.Client{Timeout: 6 * time.Second}
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

	dec := json.NewDecoder(resp.Body)
	t, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to parse Universe index: %w", err)
	}
	delim, ok := t.(json.Delim)
	if !ok || delim != '[' {
		return nil, fmt.Errorf("expected JSON array in Universe index, got %v", t)
	}

	res := make(map[string][]string)
	for dec.More() {
		var it indexItem
		if err := dec.Decode(&it); err != nil {
			return nil, fmt.Errorf("failed to parse Universe index item: %w", err)
		}
		res[it.Name] = append(res[it.Name], it.Version)
	}
	return res, nil
}

// FetchRemotePackageVersions queries the official CDN for all versions of a package.
func FetchRemotePackageVersions(pkgName string) ([]string, bool, error) {
	index, err := FetchUniverseIndex()
	if err != nil {
		return nil, false, err
	}
	vers, ok := index[pkgName]
	return vers, ok, nil
}
