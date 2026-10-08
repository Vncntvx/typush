package commands_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vncntvx/typush/commands"
	"github.com/Vncntvx/typush/util"
)

func TestPath(t *testing.T) {
	base, err := util.TypstLocalDir()
	if err != nil {
		t.Fatalf("failed to get TypstLocalDir: %v", err)
	}

	// 1. typush path (empty namespace)
	out, err := captureStdout(t, func() error {
		return commands.Path("")
	})
	if err != nil {
		t.Fatalf("Path(\"\") returned error: %v", err)
	}
	if strings.TrimSpace(out) != base {
		t.Errorf("Path(\"\") = %q, expected %q", strings.TrimSpace(out), base)
	}

	// 2. typush path preview
	outPreview, err := captureStdout(t, func() error {
		return commands.Path("preview")
	})
	if err != nil {
		t.Fatalf("Path(\"preview\") returned error: %v", err)
	}
	expectedPreview := filepath.Join(base, "preview")
	if strings.TrimSpace(outPreview) != expectedPreview {
		t.Errorf("Path(\"preview\") = %q, expected %q", strings.TrimSpace(outPreview), expectedPreview)
	}

	// 3. typush path @local (with @ prefix)
	outLocal, err := captureStdout(t, func() error {
		return commands.Path("@local")
	})
	if err != nil {
		t.Fatalf("Path(\"@local\") returned error: %v", err)
	}
	expectedLocal := filepath.Join(base, "local")
	if strings.TrimSpace(outLocal) != expectedLocal {
		t.Errorf("Path(\"@local\") = %q, expected %q", strings.TrimSpace(outLocal), expectedLocal)
	}
}

func TestPath_InvalidNamespace(t *testing.T) {
	invalidNamespaces := []string{
		"../../outside",
		"preview/sub",
		`local\test`,
		"..",
	}

	for _, ns := range invalidNamespaces {
		t.Run(ns, func(t *testing.T) {
			err := commands.Path(ns)
			if err == nil {
				t.Errorf("expected error for invalid namespace %q, got nil", ns)
			}
			if !strings.Contains(err.Error(), "invalid namespace") {
				t.Errorf("expected error message to contain 'invalid namespace', got %v", err)
			}
		})
	}
}
