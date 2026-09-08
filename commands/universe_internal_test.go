package commands

import (
	"strings"
	"testing"
)

func TestSubmissionPRBodyUpdate(t *testing.T) {
	s := submission{name: "demo", version: "0.2.0", description: "Fix things."}
	body := s.prBody()
	if !strings.Contains(body, "- [x] an update for a package") {
		t.Errorf("expected update box checked:\n%s", body)
	}
	if strings.Contains(body, "I have read and followed") {
		t.Errorf("update PR must not carry the new-package checklist:\n%s", body)
	}
	if !strings.Contains(body, "Thanks for submitting a package!") {
		t.Errorf("expected the official template header comment:\n%s", body)
	}
}

func TestSubmissionPRBodyNew(t *testing.T) {
	s := submission{name: "demo", version: "0.1.0", isNewPackage: true, description: "A demo.", hasTemplate: true}
	body := s.prBody()
	if !strings.Contains(body, "- [x] a new package") {
		t.Errorf("expected new-package box checked:\n%s", body)
	}
	for _, want := range []string{
		"I have read and followed",
		"docs/manifest.md#naming-rules",
		"without restriction, after modifying them through normal use.",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in new-package body:\n%s", want, body)
		}
	}
}

func TestSubmissionPRBodyNewNoTemplate(t *testing.T) {
	s := submission{name: "demo", version: "0.1.0", isNewPackage: true, description: "A demo."}
	if body := s.prBody(); strings.Contains(body, "without restriction") {
		t.Errorf("template license box must be absent without a template:\n%s", body)
	}
}
