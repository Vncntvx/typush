package cliutil

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// stdinReader is shared across prompts: creating a new bufio.Reader per
// prompt would over-read and discard buffered piped input.
var stdinReader = bufio.NewReader(os.Stdin)

// Confirm prompts y/N (default from param). Piped stdin is honored;
// EOF falls back to def.
func Confirm(prompt string, def bool) bool {
	defStr := "y/N"
	if def {
		defStr = "Y/n"
	}
	fmt.Fprintf(os.Stderr, "%s [%s]: ", prompt, defStr)
	line, err := stdinReader.ReadString('\n')
	if err != nil {
		fmt.Fprintln(os.Stderr)
		return def
	}
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return def
	}
	return line == "y" || line == "yes"
}

// PromptLine prompts with optional default. Piped stdin is honored;
// EOF falls back to def (or "" when allowEmpty).
func PromptLine(prompt, def string, allowEmpty bool) (string, error) {
	if def != "" {
		fmt.Fprintf(os.Stderr, "%s [%s]: ", prompt, def)
	} else {
		fmt.Fprintf(os.Stderr, "%s: ", prompt)
	}
	line, err := stdinReader.ReadString('\n')
	if err != nil {
		fmt.Fprintln(os.Stderr)
		if def != "" {
			return def, nil
		}
		if allowEmpty {
			return "", nil
		}
		return "", fmt.Errorf("aborted: non-interactive stdin")
	}
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		if def != "" {
			return def, nil
		}
		if !allowEmpty {
			return "", fmt.Errorf("value must not be empty")
		}
	}
	return line, nil
}

// PromptPassword reads a secret without echo.
func PromptPassword(prompt string) (string, error) {
	fmt.Fprintf(os.Stderr, "%s: ", prompt)
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf("aborted: non-interactive stdin")
	}
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// MultiSelect presents a numbered checklist; input like "1,3-5" or empty.
func MultiSelect(prompt string, items []string) ([]int, error) {
	fmt.Fprintf(os.Stderr, "%s (comma-separated numbers, empty = none):\n", prompt)
	for i, it := range items {
		fmt.Fprintf(os.Stderr, "  %2d) %s\n", i+1, it)
	}
	fmt.Fprintf(os.Stderr, "choice: ")
	line, err := stdinReader.ReadString('\n')
	if err != nil {
		fmt.Fprintln(os.Stderr)
		return nil, nil
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}
	var out []int
	seen := map[int]bool{}
	for _, part := range strings.Split(line, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			b := strings.SplitN(part, "-", 2)
			lo, err1 := strconv.Atoi(strings.TrimSpace(b[0]))
			hi, err2 := strconv.Atoi(strings.TrimSpace(b[1]))
			if err1 != nil || err2 != nil {
				return nil, fmt.Errorf("invalid selection %q", part)
			}
			for i := lo; i <= hi; i++ {
				if i >= 1 && i <= len(items) && !seen[i-1] {
					seen[i-1] = true
					out = append(out, i-1)
				}
			}
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 1 || n > len(items) {
			return nil, fmt.Errorf("invalid selection %q", part)
		}
		if !seen[n-1] {
			seen[n-1] = true
			out = append(out, n-1)
		}
	}
	return out, nil
}

var gitVerRe = regexp.MustCompile(`git version (\d+)\.(\d+)\.(\d+)`)

// GitSupportsSparseCheckout reports whether git >= 2.25 (needed for sparse-checkout --cone).
func GitSupportsSparseCheckout() bool {
	out, err := exec.Command("git", "--version").Output()
	if err != nil {
		return false
	}
	m := gitVerRe.FindStringSubmatch(string(out))
	if len(m) != 4 {
		return false
	}
	maj, _ := strconv.Atoi(m[1])
	min, _ := strconv.Atoi(m[2])
	return maj > 2 || (maj == 2 && min >= 25)
}

// RunGit runs git with inherited stdio and returns the error if any.
func RunGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
