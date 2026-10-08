package commands_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// capWhich selects which standard stream a capture redirects, so the other one
// stays visible in `go test -v` output when an assertion fails.
type capWhich int

const (
	capStdout capWhich = iota
	capStderr
	capBoth
)

// captureStream runs fn with the selected streams redirected and returns what
// was written to them. The write ends are closed even when fn panics or calls
// t.Fatalf, so the reader goroutines can always finish.
func captureStream(t *testing.T, which capWhich, fn func() error) (stdout, stderr string, err error) {
	t.Helper()
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %v", err)
	}
	origStdout, origStderr := os.Stdout, os.Stderr
	t.Cleanup(func() { os.Stdout, os.Stderr = origStdout, origStderr })
	if which != capStderr {
		os.Stdout = outW
	}
	if which != capStdout {
		os.Stderr = errW
	}

	var outBuf, errBuf bytes.Buffer
	outDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(&outBuf, outR)
		close(outDone)
	}()
	errDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(&errBuf, errR)
		close(errDone)
	}()

	func() {
		defer func() { _ = outW.Close(); _ = errW.Close() }()
		err = fn()
	}()
	<-outDone
	<-errDone
	_ = outR.Close()
	_ = errR.Close()

	return outBuf.String(), errBuf.String(), err
}

// captureStdout returns everything fn wrote to stdout.
func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	out, _, err := captureStream(t, capStdout, fn)
	return out, err
}

// captureStderr returns everything fn wrote to stderr.
func captureStderr(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	_, errOut, err := captureStream(t, capStderr, fn)
	return errOut, err
}

// captureStdio returns everything fn wrote to both streams. Use it instead of
// nesting captureStdout inside captureStderr.
func captureStdio(t *testing.T, fn func() error) (stdout, stderr string, err error) {
	t.Helper()
	return captureStream(t, capBoth, fn)
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
