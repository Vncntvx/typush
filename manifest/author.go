// Package manifest author validation. Port of typst/packages bundler/src/author.rs.
package manifest

import (
	"fmt"
	"strings"
	"unicode"
)

// ValidateAuthor checks "Name" or "Name <contact>" where contact is a GitHub
// handle (@user), URL (http...) or email address.
func ValidateAuthor(name string) error {
	contact := ""
	if i := strings.Index(name, "<"); i >= 0 {
		rest := name[i+1:]
		j := strings.Index(rest, ">")
		if j < 0 {
			return fmt.Errorf("author %q: expected '>'", name)
		}
		contact = rest[:j]
		if strings.Contains(rest[j+1:], "<") || strings.Contains(rest[j+1:], ">") {
			return fmt.Errorf("author %q: unexpected bracket", name)
		}
	}
	if contact == "" && strings.Contains(name, "<") {
		return fmt.Errorf("author %q: expected '>'", name)
	}
	if contact == "" {
		return nil // bare name
	}
	if h, ok := strings.CutPrefix(contact, "@"); ok {
		return validateGitHubHandle(name, h)
	}
	if strings.HasPrefix(contact, "http") {
		return validateAuthorURL(name, contact)
	}
	return validateAuthorEmail(name, contact)
}

func validateGitHubHandle(full, handle string) error {
	if handle == "" {
		return fmt.Errorf("author %q: GitHub handle is invalid: empty", full)
	}
	if len(handle) > 39 {
		return fmt.Errorf("author %q: GitHub handle is invalid: cannot be longer than 39 characters", full)
	}
	for _, c := range handle {
		if !(c < 128 && (unicode.IsLetter(c) || unicode.IsDigit(c)) || c == '-') {
			return fmt.Errorf("author %q: GitHub handle is invalid: must only contain alphanumeric characters and '-'", full)
		}
	}
	if strings.HasPrefix(handle, "-") || strings.HasSuffix(handle, "-") {
		return fmt.Errorf("author %q: GitHub handle is invalid: must not start or end with a hyphen", full)
	}
	if strings.Contains(handle, "--") {
		return fmt.Errorf("author %q: GitHub handle is invalid: cannot contain consecutive hyphens", full)
	}
	return nil
}

func validateAuthorURL(full, url string) error {
	if url == "" {
		return fmt.Errorf("author %q: URL is invalid: empty", full)
	}
	for _, c := range url {
		if !isLegalInURL(c) {
			return fmt.Errorf("author %q: URL is invalid: URL contains invalid characters", full)
		}
	}
	return nil
}

func isLegalInURL(c rune) bool {
	if c < 128 && (unicode.IsLetter(c) || unicode.IsDigit(c)) {
		return true
	}
	return strings.ContainsRune("-_.~:/?#[]@!$&'()*+,;=", c)
}

// validateAuthorEmail is a primitive email check mirroring the bundler's
// test expectations (rejects empty, bare words, bad domains).
func validateAuthorEmail(full, addr string) error {
	if strings.TrimSpace(addr) != addr || addr == "" {
		return fmt.Errorf("author %q: email is invalid", full)
	}
	if strings.Contains(addr, " ") {
		return fmt.Errorf("author %q: email is invalid", full)
	}
	parts := strings.Split(addr, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("author %q: email is invalid", full)
	}
	domain := parts[1]
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") ||
		strings.HasPrefix(domain, "-") || strings.HasSuffix(domain, "-") ||
		!strings.Contains(domain, ".") || strings.Contains(domain, "..") {
		return fmt.Errorf("author %q: email is invalid", full)
	}
	for _, c := range addr {
		if c < 32 || c == '<' || c == '>' {
			return fmt.Errorf("author %q: email is invalid", full)
		}
	}
	return nil
}
