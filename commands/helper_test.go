package commands_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	origStdout := os.Stdout
	os.Stdout = w
	defer func() {
		os.Stdout = origStdout
	}()

	outChan := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outChan <- buf.String()
	}()

	fnErr := fn()

	w.Close()
	out := <-outChan
	_ = r.Close()

	return out, fnErr
}

func captureStderr(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	origStderr := os.Stderr
	os.Stderr = w
	defer func() {
		os.Stderr = origStderr
	}()

	outChan := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outChan <- buf.String()
	}()

	fnErr := fn()

	w.Close()
	out := <-outChan
	_ = r.Close()

	return out, fnErr
}

func writeTestPackage(t *testing.T, dir string) {
	t.Helper()
	tomlContent := `[package]
name = "demo-pkg"
version = "1.2.3"
entrypoint = "lib.typ"
authors = ["Alice", "Bob"]
license = "MIT"
description = "A demo package for testing metadata"
repository = "https://github.com/example/demo-pkg"
homepage = "https://example.com"
compiler = "0.12.0"
categories = ["utility"]
disciplines = ["computer-science"]
keywords = ["test", "demo"]
exclude = ["tests"]
`
	if err := os.WriteFile(filepath.Join(dir, "typst.toml"), []byte(tomlContent), 0o644); err != nil {
		t.Fatalf("failed to write typst.toml: %v", err)
	}
}
