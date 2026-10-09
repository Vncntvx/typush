package commands_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Vncntvx/typush/commands"
)

const infoIndex = `[
	{
		"name": "fletcher",
		"version": "0.5.0",
		"description": "Diagrams library v0.5.0",
		"license": "MIT",
		"authors": ["Alice <alice@example.com>"],
		"repository": "https://github.com/example/fletcher",
		"compiler": "0.11.0",
		"categories": ["visualization"],
		"updatedAt": 1710000000
	},
	{
		"name": "fletcher",
		"version": "0.5.1",
		"description": "Diagrams library v0.5.1",
		"license": "MIT",
		"authors": ["Alice <alice@example.com>"],
		"repository": "https://github.com/example/fletcher",
		"compiler": "0.12.0",
		"categories": ["visualization"],
		"updatedAt": 1720000000
	}
]`

func TestInfo(t *testing.T) {
	useUniverse(t, infoIndex)

	stdout, err := captureStdout(t, func() error {
		return commands.Info("fletcher", commands.UniverseOptions{Refresh: true})
	})
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}
	for _, want := range []string{
		"Package:      fletcher",
		"Version:      0.5.1 (latest)",
		"Versions:     0.5.1, 0.5.0 (2 release(s))",
		`#import "@preview/fletcher:0.5.1": *`,
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected %q in output, got:\n%s", want, stdout)
		}
	}

	stdout, err = captureStdout(t, func() error {
		return commands.Info("fletcher:0.5.0", commands.UniverseOptions{})
	})
	if err != nil {
		t.Fatalf("Info with version failed: %v", err)
	}
	if !strings.Contains(stdout, "Version:      0.5.0 (latest is 0.5.1)") {
		t.Errorf("expected the older-version indicator, got:\n%s", stdout)
	}
}

// Every accepted spec form must resolve to the same package.
func TestInfo_AcceptsSpecForms(t *testing.T) {
	useUniverse(t, infoIndex)

	for _, spec := range []string{"fletcher", "@preview/fletcher", "preview/fletcher"} {
		stdout, err := captureStdout(t, func() error {
			return commands.Info(spec, commands.UniverseOptions{})
		})
		if err != nil {
			t.Fatalf("Info(%q) failed: %v", spec, err)
		}
		if !strings.Contains(stdout, "Package:      fletcher") {
			t.Errorf("Info(%q) did not resolve the package, got:\n%s", spec, stdout)
		}
	}
}

func TestInfo_JSON(t *testing.T) {
	useUniverse(t, infoIndex)

	jsonOut, err := captureStdout(t, func() error {
		return commands.Info("fletcher", commands.UniverseOptions{AsJSON: true})
	})
	if err != nil {
		t.Fatalf("Info --json failed: %v", err)
	}

	var resp commands.InfoResponse
	if err := json.Unmarshal([]byte(jsonOut), &resp); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if resp.Name != "fletcher" || resp.Version != "0.5.1" || !resp.IsLatest || len(resp.AllVersions) != 2 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestInfo_Errors(t *testing.T) {
	useUniverse(t, infoIndex)

	if err := commands.Info("nonexistent", commands.UniverseOptions{}); err == nil {
		t.Error("expected an error for an unknown package")
	}

	err := commands.Info("fletcher:9.9.9", commands.UniverseOptions{})
	if err == nil {
		t.Fatal("expected an error for an unknown version")
	}
	if !strings.Contains(err.Error(), `version "9.9.9" not found`) {
		t.Errorf("unexpected error message: %v", err)
	}

	// A name Universe cannot hold must be rejected before the lookup, so the error
	// names the name rule instead of reporting a missing package.
	err = commands.Info("Not_Valid", commands.UniverseOptions{})
	if err == nil {
		t.Fatal("expected an error for a non-kebab-case package name")
	}
	if !strings.Contains(err.Error(), "kebab-case") {
		t.Errorf("expected the Universe name rule to be reported, got: %v", err)
	}

	// Same for a version that is not semver.
	err = commands.Info("fletcher:not-a-version", commands.UniverseOptions{})
	if err == nil {
		t.Fatal("expected an error for a non-semver version")
	}
	if !strings.Contains(err.Error(), "expected semver") {
		t.Errorf("expected the version rule to be reported, got: %v", err)
	}

	if err := commands.Info("", commands.UniverseOptions{}); err == nil {
		t.Error("expected an error for an empty package name")
	}
}
