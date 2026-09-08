package commands

import (
	"strings"
	"testing"
)

// The submission body must stay byte-identical in shape to the previously
// accepted submission style: full template comments, pre-checked boxes,
// CRLF line endings.
func TestSubmissionPRBodyUpdate(t *testing.T) {
	s := submission{name: "demo", version: "0.2.0", description: "Fix things.", hasTemplate: true}
	body := s.prBody()
	for _, want := range []string{
		"- [ ] a new package\r\n",
		"- [x] an update for a package\r\n",
		"Description: Fix things.\r\n",
		"I have read and followed the submission guidelines and, in particular, I\r\n",
		"docs/manifest.md#naming-rules",
		"without restriction, after modifying them through normal use.\r\n",
		"Thanks for submitting a package!",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in update body:\n%q", want, body)
		}
	}
	if strings.Contains(body, "Explanation:") {
		t.Errorf("name Explanation block must be omitted:\n%q", body)
	}
	if strings.Contains(body, "\n- ") && !strings.Contains(body, "\r\n- ") {
		t.Errorf("expected CRLF line endings:\n%q", body)
	}
}

func TestSubmissionPRBodyNew(t *testing.T) {
	s := submission{name: "demo", version: "0.1.0", isNewPackage: true, description: "A demo.", hasTemplate: true}
	body := s.prBody()
	if !strings.Contains(body, "- [x] a new package\r\n") {
		t.Errorf("expected new-package box checked:\n%q", body)
	}
	if !strings.Contains(body, "I have read and followed") {
		t.Errorf("expected checklist in new-package body:\n%q", body)
	}
}

func TestSubmissionPRBodyNewNoTemplate(t *testing.T) {
	s := submission{name: "demo", version: "0.1.0", isNewPackage: true, description: "A demo."}
	if body := s.prBody(); strings.Contains(body, "without restriction") {
		t.Errorf("template license box must be absent without a template:\n%q", body)
	}
}
